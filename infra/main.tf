/**
 * Copyright 2021-2023 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

terraform {
  required_version = ">= 1"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.73.1"
    }
  }
}

provider "google" {
  credentials = var.creds
  project     = var.project
  region      = var.region
}

data "google_project" "project" {
}

resource "google_storage_bucket" "tfstate_bucket" {
  name                        = "tfstate-${var.project}"
  location                    = var.region
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }
  lifecycle_rule {
    condition {
      age        = "14"
      with_state = "ARCHIVED"
    }
    action {
      type = "Delete"
    }
  }
}

resource "google_project_service" "service" {
  for_each = toset(["pubsub", "cloudiot", "monitoring", "iam", "run"])
  service  = "${each.value}.googleapis.com"
}

resource "google_pubsub_topic" "measurements_topic" {
  name       = "measurements"
  depends_on = [google_project_service.service["pubsub"]]
}

resource "google_service_account" "measurements-publisher" {
  account_id   = "measurements-publisher"
  display_name = "Measurements Publisher"
  description  = "Service account used for pushing measurements via Pub/Sub"

  depends_on = [google_project_service.service["iam"], google_project_service.service["pubsub"]]
}

data "google_iam_policy" "measurements_topic-policy" {
  binding {
    role    = "roles/pubsub.publisher"
    members = [google_service_account.measurements-publisher.member]
  }
}

resource "google_pubsub_topic_iam_policy" "measurements_topic-iam" {
  project     = google_pubsub_topic.measurements_topic.project
  topic       = google_pubsub_topic.measurements_topic.name
  policy_data = data.google_iam_policy.measurements_topic-policy.policy_data
}

resource "google_cloudiot_registry" "sensors_registry" {
  name = "sensors"

  event_notification_configs {
    pubsub_topic_name = google_pubsub_topic.measurements_topic.id
    subfolder_matches = ""
  }

  mqtt_config = {
    mqtt_enabled_state = "MQTT_ENABLED"
  }

  http_config = {
    http_enabled_state = "HTTP_ENABLED"
  }

  depends_on = [google_project_service.service["cloudiot"]]
}

locals {
  metrics = [
    { name = "temperature", unit = "{C}", pretty = "Temperature" },
    { name = "humidity", unit = "%%{RH}", pretty = "Humidity" },
    { name = "pressure", unit = "{hPa}", pretty = "Pressure" },
    { name = "battery", unit = "{V}", pretty = "Battery voltage" },
  ]
}

resource "google_monitoring_metric_descriptor" "metric" {
  for_each = { for metric in local.metrics : metric.name => metric }

  description  = "Sensor measurement of ${each.key}"
  display_name = title(each.key)
  type         = "custom.googleapis.com/sensor/measurement/${each.key}"
  metric_kind  = "GAUGE"
  value_type   = "DOUBLE"
  unit         = each.value.unit

  depends_on = [google_project_service.service["monitoring"]]
}

locals {
  mac_renamer = length(var.sensors) == 0 ? "" : "\n| map [sensor: ${join("", [for mac, name in var.sensors : "if(sensor == '${mac}', '${name}', "])}sensor${join("", [for _ in var.sensors : ")"])}]"
}

resource "google_monitoring_dashboard" "climate-station-dashboard" {
  dashboard_json = jsonencode({
    "displayName" : "Climate Station",
    "gridLayout" : {
      "columns" : "2",
      "widgets" : [for metric in local.metrics : {
        "title" : metric.pretty,
        "xyChart" : {
          "chartOptions" : {
            "mode" : "COLOR"
          },
          "dataSets" : [
            {
              "plotType" : "LINE",
              "timeSeriesQuery" : {
                "timeSeriesQueryLanguage" : "fetch generic_node\n| metric 'custom.googleapis.com/sensor/measurement/${metric.name}'\n| map rename[sensor: resource.node_id]${local.mac_renamer}"
              },
              "targetAxis" : "Y1"
            }
          ],
          "timeshiftDuration" : "0s",
          "yAxis" : {
            "label" : "y1Axis",
            "scale" : "LINEAR"
          }
        }
      }]
    }
  })

  depends_on = [google_project_service.service["monitoring"], google_monitoring_metric_descriptor.metric]
}

resource "google_service_account" "metricspusher-invoker" {
  account_id   = "metricspusher-invoker"
  display_name = "metricspusher invoker"
  description  = "Service account allowed to invoke the metricspusher Cloud Run service"

  depends_on = [google_project_service.service["iam"], google_project_service.service["pubsub"]]
}

data "google_iam_policy" "metricspusher-invoker-policy" {
  binding {
    role    = "roles/iam.serviceAccountTokenCreator"
    members = ["serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"]
  }
}

resource "google_service_account_iam_policy" "metricspusher-invoker-iam" {
  service_account_id = google_service_account.metricspusher-invoker.name
  policy_data        = data.google_iam_policy.metricspusher-invoker-policy.policy_data
}

resource "google_service_account" "metricspusher" {
  account_id   = "metricspusher"
  display_name = "metricspusher"
  description  = "Service account running the metricspusher Cloud Run service"

  depends_on = [google_project_service.service["iam"]]
}

resource "google_project_iam_member" "metricspusher-metrics-writer" {
  project = var.project
  role    = "roles/monitoring.metricWriter"
  member  = google_service_account.metricspusher.member
}

resource "google_cloud_run_service" "metricspusher-service" {
  name     = "metricspusher"
  location = var.region

  template {
    spec {
      containers {
        image = "${var.container_registry}/${var.project}/metricspusher:latest"
      }
      service_account_name = google_service_account.metricspusher.email
    }
    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = "2"
      }
      labels = {
        "run.googleapis.com/startupProbeType" = "Default"
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  autogenerate_revision_name = true

  depends_on = [google_project_service.service["run"]]
}

data "google_iam_policy" "metricspusher-service-policy" {
  binding {
    role    = "roles/run.invoker"
    members = [google_service_account.metricspusher-invoker.member]
  }
}

resource "google_cloud_run_service_iam_policy" "metricspusher-service-iam" {
  location = google_cloud_run_service.metricspusher-service.location
  project  = google_cloud_run_service.metricspusher-service.project
  service  = google_cloud_run_service.metricspusher-service.name

  policy_data = data.google_iam_policy.metricspusher-service-policy.policy_data
}

resource "google_pubsub_subscription" "measurements-subscription" {
  name  = "measurements-subscription"
  topic = google_pubsub_topic.measurements_topic.name

  ack_deadline_seconds       = 20
  message_retention_duration = "3600s"

  push_config {
    push_endpoint = google_cloud_run_service.metricspusher-service.status[0].url
    oidc_token {
      service_account_email = google_service_account.metricspusher-invoker.email
    }
    # Currently there is only one version and keeping it causes permament diff.
    # attributes = {
    #   x-goog-version = "v1"
    # }
  }

  depends_on = [
    google_service_account_iam_policy.metricspusher-invoker-iam,
    google_cloud_run_service_iam_policy.metricspusher-service-iam,
  ]
}
