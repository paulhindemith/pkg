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

REPO_ROOT_DIR=$(git rev-parse --show-toplevel)
readonly TEST_NAMESPACE="hello-world"
readonly E2E_KIND_CONFIG_PATH="${REPO_ROOT_DIR}/conformance/hello-world-serving/test/kind-config.yaml"
readonly TEST_CONFIG_DIR="${REPO_ROOT_DIR}/conformance/hello-world-serving/test/config"

source $(dirname $0)/e2e-common.sh

function knative_setup() {
  install_knative_serving
}

# Can setup followings command and argument
# ./create-dev-cluster.sh --local --skip-knative-teardown --skip-cluster-teardown --cluster-version "1.17.0"
# export KUBECONFIG="$HOME/.kube/kind-config-kind-kubetest"
initialize $@

success
