# Infrastructure

This describes the steps to set up and configure a GCP project that hosts a
receiver of the measurements published by a gbcsdpd instance and a dashboard to
view data.

We use [Terraform](https://www.terraform.io/) to configure resources in the
project but there are still some manual steps required because the created GCP
project is self-contained. We store
[Terraform state](https://www.terraform.io/docs/language/state/index.html) in
the GCS bucket in the project itself, and are not using for example Terraform
Admin Project pattern as described in
[Managing Google Cloud projects with Terraform](https://cloud.google.com/community/tutorials/managing-gcp-projects-with-terraform)
thus we need to create and prepare the project for Terraform manually.

## One time setup

These one-time setup instructions need to be executed only once and the
[Maintenance](#maintanance) section describes how to make changes to it.

### Preparation

Install and configure:

- [Bazel](https://docs.bazel.build/versions/master/install.html)
- [gcloud](https://cloud.google.com/sdk/gcloud)
- [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli)
- [openssl cli](https://wiki.openssl.org/index.php/Command_Line_Utilities)

### Setup GCP Project

1. Source `setup.sh` into shell environment. This script sets up `$PROJECT_ID`,
   `$REGION`, `$CONTAINER_REGISTRY` environment variables and creates
   `terraform.tfvars` file.

   ```sh
   $ source setup.sh
   project id: climate-station-273399
   gcp region: europe-west1
   container registry: eu.gcr.io
   ```

   To restore environment variables after eg. restarting shell just source
   `setup.sh` again and it will use values from existing `terraform.tfvars`.

1. Create a project

   ```sh
   gcloud projects create $PROJECT_ID --name="Climate Station"
   ```

   If you want to create the project in an organization add
   `--organization=ORG_ID` to the command above.

1. Link a billing account to the project

   ```sh
   gcloud beta billing accounts list
   gcloud beta billing projects link --billing-account=XXXXXX-XXXXXX-XXXXXX $PROJECT_ID
   ```

1. Enable the Resource Manager API and Container Registry API

   ```sh
   gcloud --project=$PROJECT_ID services enable cloudresourcemanager.googleapis.com containerregistry.googleapis.com
   ```

1. Create docker authentication settings for pushing images to the container
   registry

   ```sh
   gcloud auth configure-docker
   ```

   Bazel `container_push` rule will use it in the next step.

1. Build and push a [`metricspusher`](../cmd/metricspusher) image to the
   container registry:

   ```sh
   bazel run --define project=$PROJECT_ID --define registry=$CONTAINER_REGISTRY \
       --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
       //cmd/metricspusher:push_metricspusher
   ```

1. Create a service account for Terraform, get a key (saved in `creds.json`) and
   add it to the project's IAM policy:

   ```sh
   gcloud --project=$PROJECT_ID iam service-accounts create terraform
   gcloud --project=$PROJECT_ID iam service-accounts keys create creds.json \
       --iam-account=terraform@${PROJECT_ID}.iam.gserviceaccount.com
   gcloud projects add-iam-policy-binding $PROJECT_ID --role=roles/owner \
       --member=serviceAccount:terraform@${PROJECT_ID}.iam.gserviceaccount.com 
   ```

1. Create a bucket for Terraform state:

   ```sh
   gsutil mb -p $PROJECT_ID -c STANDARD -l $REGION -b on gs://tfstate-${PROJECT_ID}/
   ```

1. Intialize Terraform:

   ```sh
   terraform init -backend-config="bucket=tfstate-${PROJECT_ID}"
   ```

1. Import the Terraform state GCS bucket resource into the Terraform state:

   ```sh
   terraform import google_storage_bucket.tfstate_bucket tfstate-${PROJECT_ID}
   ```

1. Now, after we've finally set up the project and Terraform we can use it to
   set up all other GCP resources:

   ```sh
   terraform apply
   ```

   It sometimes happened to me that this command failed because some resources
   were not ready. Just retry.

### Setup publishing daemon

1. Name our daemon instance somehow, eg. `climate-publisher` or pick the
   hostname of the device it's running on.

   ```sh
   DEVICE_NAME=climate-publisher
   ```

1. Create a key pair for authentication of the publisher with Cloud IoT.

   ```sh
   openssl genpkey -algorithm RSA -out rsa_private.pem -pkeyopt rsa_keygen_bits:2048
   openssl rsa -in rsa_private.pem -pubout -out rsa_public.pem
   ```

1. Add the publisher as a new device to our Cloud IoT registry.

   ```sh
   gcloud --project=$PROJECT_ID iot devices create --region=$REGION --registry=sensors \
       --public-key path=rsa_public.pem,type=rsa-pem $DEVICE_NAME
   ```

1. Append following GCP sink configuration to your `config.toml` daemon
   configuration (see [cmd/gbcsdpd](cmd/gbcsdpd) for details about daemon
   setup).

   ```sh
   cat <<EOF >> config.toml
   [[sinks.gcp]]
   project = "$PROJECT_ID"
   region = "$REGION"
   registry = "sensors"
   device = "$DEVICE_NAME"
   key = "rsa_private.pem"
   rate_limit.max_1_in = "2m"
   EOF
   ```

   Don't forget to keep the generated `rsa_private.pem` next to the
   `config.toml` file.

### Finish configuring monitoring dashboard

Terraform also created a "Climate Station" Cloud Monitoring dashboard for all
the measurements that you can find at
https://console.cloud.google.com/monitoring/dashboards/. It will populate after
deamon starts publishing data.

You can notice that the lines on the graph don't have any friendly names, but
MAC adresses of devices publishing data. To set some friendly names on the
dashboard:

1. Add `sensors` variable to the `terraform.tfvars` that maps MAC to friendly
   name, eg:

   ```
   sensors = {
     "aa:bb:cc:ee:dd:ee" = "balcony"
     "11:33:55:11:55:11" = "living room"
     "42:66:66:99:00:ff" = "bedroom"
   }
   ```

1. Run `terraform apply` to update the monitoring dashboard.

## Maintenance

This doesn't require any ongoing maintenace, so this section is only useful for
doing upgrades of infrastructure or `metricpusher` image.

### Re-setup on a different machine

Once the one-time setup above is done and we would like to make some changes
with Terraform from a different machine, we needs to only:

1. Source `setup.sh` providing known values (or copy `terraform.tfvars` to not have
   to retype those) to set environment variables (remember about optional `sensors` field).
1. Get `terraform@${PROJECT_ID}.iam.gserviceaccount.com` key into `creds.json`.
1. Run `terraform init -backend-config="bucket=tfstate-${PROJECT_ID}"` to
   initialize Terraform.
1. Terraform is ready for any plan/apply commands.

### Updating the Cloud Run services

Push the new image to the container registry using the Bazel command from setup
step 6, and then update Cloud Run service to the new latest image with:

```sh
gcloud --project=$PROJECT_ID run deploy metricspusher --platform managed \
    --region $REGION --image $CONTAINER_REGISTRY/$PROJECT_ID/metricspusher:latest
```
