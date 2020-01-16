#!/bin/bash

# Copyright 2020 Paulhindemith
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

# Download and unpack Knative
KNATIVE_VERSION=0.11.1
# Contains core, hpa, istio
SERVING_DOWNLOAD_URL=https://github.com/knative/serving/releases/download/v${KNATIVE_VERSION}/serving.yaml

cd $(dirname $0)

wget --no-check-certificate $SERVING_DOWNLOAD_URL
if [ $? != 0 ]; then
  echo "Failed to download knative package"
  exit 1
fi
