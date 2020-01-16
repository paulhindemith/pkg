#!/bin/bash

# Copyright 2019 The Knative Authors
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
# https://github.com/knative/serving/blob/bde1912a8c69bbf7e6fc1109ac9f0e7b34c04c9c/third_party/cert-manager-0.12.0/download-cert-manager.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# 3909cea027c5856a91111ada38f4e8227660b67b

# Download and unpack cert-manager
CERT_MANAGER_VERSION=0.12.0
YAML_URL=https://github.com/jetstack/cert-manager/releases/download/v${CERT_MANAGER_VERSION}/cert-manager.yaml

cd $(dirname $0)

# Download the cert-manager yaml file
wget $YAML_URL

if [ $? != 0 ]; then
  echo "Failed to download cert-manager package"
  exit 1
fi
