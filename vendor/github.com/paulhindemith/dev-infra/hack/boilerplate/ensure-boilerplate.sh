#!/bin/bash

# Copyright 2018 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Modifications Copyright 2020 Paulhindemith
#
# The original source code can be referenced from the link below.
# https://github.com/knative/serving/blob/9f64f866d633b5cf7ffc4e50e3bc327fd9a3a924/hack/boilerplate/add-boilerplate.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# 550faea6bb43f0d7fa6a214dc29b5e9760bfe066

readonly ROOT_DIR=$(git rev-parse --show-toplevel)

set -o errexit
set -o nounset

if [ -z "$1" ]; then
  echo "** Internal error in ensure-boilerplate.sh, argument is not given."
  exit 1
fi

function ensure_boilerplate() {
  grep -r -L -P "Copyright \d+ $3" $2  \
    | grep -P "\.$1\$" \
    | xargs -I {} sh -c \
    "cat $(dirname ${BASH_SOURCE[0]})/boilerplate.$1.txt {} > /tmp/boilerplate && mv /tmp/boilerplate {}"
  if [[ $1 = "sh" ]]; then
    chmod 755 $2
  fi
}

function main() {
  local target_files
  target_files="$(find ${ROOT_DIR} -type f -name "*.sh" -o -name "*.go" -o -name "*.yaml" | grep -v "/vendor/")" || exit 1

  for fi in $(echo $target_files); do
    case "$fi" in
      *.go ) ensure_boilerplate go ${fi} $1 ;;
      *.sh ) ensure_boilerplate sh ${fi} $1 ;;
      *.yaml ) ensure_boilerplate yaml ${fi} $1 ;;
      * ) exit 1 ;;
    esac
  done
}

main "$1"
