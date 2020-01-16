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
# https://github.com/knative/serving/tree/1957dff9cea156d158cfaa2fde3c6de76dfe6de1/test/e2e-tests.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# b65b5ef3737a22497b2358b08ec8c91556c23a0d

REPO_ROOT_DIR=$(git rev-parse --show-toplevel)
readonly TEST_NAMESPACE="hello-world"
readonly E2E_KIND_CONFIG_PATH="${REPO_ROOT_DIR}/conformance/hello-world-serving/test/kind-config.yaml"
readonly TEST_CONFIG_DIR="${REPO_ROOT_DIR}/conformance/hello-world-serving/test/config"
readonly E2E_CLUSTER_VERSION="1.17.0"

source ${REPO_ROOT_DIR}/scripts/e2e-common.sh


function knative_setup() {
  install_knative_serving
}

function post_test_setup() {
  echo $(dirname $0)/config
  ko apply -f ${REPO_ROOT_DIR}/conformance/hello-world-serving/config
}

# Script entry point.

initialize $@

# Run the tests
header "Running tests"

failed=0

# Run tests serially in the mesh scenario
parallelism=""
(( MESH )) && parallelism="-parallel 1"

# Run tests local
e2e_args=""
if [[ ${TEST_ENV}="local" ]]; then
  e2e_args+=" --kubeconfig $(get_canonical_path "${HOME}/.kube/kind-config-kind-kubetest")"
  e2e_args+=" --ingressendpoint localhost"
fi

# Run conformance and e2e tests.
go_test_e2e -timeout=30m \
  "${REPO_ROOT_DIR}/conformance/hello-world-serving/test/e2e" \
  ${parallelism} \
  ${e2e_args} \
  "$(use_https)" || failed=1

(( failed )) && dump_cluster_state
(( failed )) && fail_test

success
