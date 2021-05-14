/**
 * Copyright 2021 Google LLC
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

variable "project" {
  description = "The GCP project id"
  type        = string
}

variable "region" {
  description = "GCP region to use for resources"
  type        = string
}

variable "container_registry" {
  description = "Name of the container registry where images are stored"
  type        = string
}

variable "creds" {
  description = "Path to the file with terraform service account credentials"
  type        = string
}

variable "sensors" {
  description = "Mapping from sensor MAC to friendly name for monitoring"
  type        = map(string)
  default     = {}
}
