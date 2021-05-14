# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [ ! -f "terraform.tfvars" ]; then
    cat << EOF > terraform.tfvars
project            = "$(bash -c "read -e -p 'project id: ' -i 'climate-station-$(printf "%06d" $(shuf -i 0-999999 -n 1))' R; echo \$R")"
region             = "$(bash -c 'read -e -p "gcp region: " -i "europe-west1" R; echo $R')"
container_registry = "$(bash -c 'read -e -p "container registry: " -i "eu.gcr.io" R; echo $R')"
creds              = "creds.json"
EOF
fi
source <(awk '{ m[$1] = $3 } END { print "PROJECT_ID=" m["project"]; print "REGION=" m["region"]; print "CONTAINER_REGISTRY=" m["container_registry"] }' terraform.tfvars)
