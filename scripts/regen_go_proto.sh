#!/bin/bash
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

TEST=false
while getopts "t" opt; do
  case "${opt}" in
    t)
      TEST=true
      ;;
    *)
      exit 1
      ;;
  esac
done
cd "$(bazel info workspace)"
bazel build //api/...
files=($(bazel aquery 'kind(proto, //api/...)' | grep Outputs | grep "[.]pb[.]go" | sed 's/Outputs: \[//' | sed 's/\]//' | tr "," "\n"))
for src in ${files[@]};
do
  dst="$(echo $src | sed -E 's|.*/github.com/p2004a/gbcsdpd/(.*)|\1|')"
  if [[ $TEST = true ]]; then
    diff -u "$src" "$dst"
  else
    echo "copying $dst"
    cp --no-preserve=mode,ownership "$src" "$dst"
  fi
done
