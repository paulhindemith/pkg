#!/usr/bin/env bash

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
# https://github.com/knative/serving/tree/cde38c6a3ba0b6a15a9f0fb07aaf635efe871787/third_party/istio-1.4.2/download-istio.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# 744c505646ec81b9916eae2ff71561a5ee9deff6


# Download and unpack Istio
ISTIO_VERSION=1.4.2
DOWNLOAD_URL=https://github.com/istio/istio/releases/download/${ISTIO_VERSION}/istio-${ISTIO_VERSION}-linux.tar.gz

cd $(dirname $0)
../ensure-helm-installed.sh

wget --no-check-certificate $DOWNLOAD_URL
if [ $? != 0 ]; then
  echo "Failed to download istio package"
  exit 1
fi
tar xzf istio-${ISTIO_VERSION}-linux.tar.gz

( # subshell in downloaded directory
cd istio-${ISTIO_VERSION} || exit

# Create CRDs template
helm template --namespace=istio-system \
  install/kubernetes/helm/istio-init \
  `# Removing trailing whitespaces to make automation happy` \
  | sed 's/[ \t]*$//' \
  > ../istio-crds.yaml

# An even lighter template, with just pilot/gateway and small resource requests.
# Based on install/kubernetes/helm/istio/values-istio-minimal.yaml
helm template --namespace=istio-system install/kubernetes/helm/istio --values ../values-local.yaml \
  `# Removing trailing whitespaces to make automation happy` \
  | sed 's/[ \t]*$//' \
  > ../istio-local.yaml
)

# Clean up.
rm -rf istio-${ISTIO_VERSION}
rm istio-${ISTIO_VERSION}-linux.tar.gz

# Add in the `istio-system` namespace to reduce number of commands.
patch istio-crds.yaml namespace.yaml.patch
patch istio-local.yaml namespace.yaml.patch
