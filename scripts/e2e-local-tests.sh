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
# https://github.com/knative/test-infra/blob/c65d7bf63c2dde11173923127b91d87965d61533/scripts/e2e-tests.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# cb948e5f09baf4a5b79dd375eb8b71cca420f322

source ${REPO_ROOT_DIR:=$(git rev-parse --show-toplevel)}/vendor/knative.dev/test-infra/scripts/library.sh

[[ -z ${KO_DOCKER_REPO} ]] && echo "KO_DOCKER_REPO environment variable is not set." && exit 1
[[ -z ${E2E_KIND_CONFIG_PATH} ]] && echo "E2E_KIND_CONFIG_PATH is not set." && exit 1


# Configurable parameters
readonly TEST_RESULT_FILE=/tmp/${E2E_BASE_NAME}-e2e-result

# Tear down the test resources.
function teardown_test_resources() {
  header "Tearing down test environment"
  function_exists test_teardown && test_teardown
  (( ! SKIP_KNATIVE_TEARDOWN )) && function_exists knative_teardown && knative_teardown
}

# Run the given E2E tests. Assume tests are tagged e2e, unless `-tags=XXX` is passed.
# Parameters: $1..$n - any go test flags, then directories containing the tests to run.
function go_test_e2e() {
  local test_options=""
  local go_options=""
  [[ ! " $@" == *" -tags="* ]] && go_options="-tags=e2e"
  report_go_test -v -race -count=1 ${go_options} $@ ${test_options}
}

# Dump info about the test cluster. If dump_extra_cluster_info() is defined, calls it too.
# This is intended to be called when a test fails to provide debugging information.
function dump_cluster_state() {
  echo "***************************************"
  echo "***         E2E TEST FAILED         ***"
  echo "***    Start of information dump    ***"
  echo "***************************************"

  local output="${ARTIFACTS}/k8s.dump.txt"
  echo ">>> The dump is located at ${output}"

  for crd in $(kubectl api-resources --verbs=list -o name | sort); do
    local count="$(kubectl get $crd --all-namespaces --no-headers 2>/dev/null | wc -l)"
    echo ">>> ${crd} (${count} objects)"
    if [[ "${count}" > "0" ]]; then
      echo ">>> ${crd} (${count} objects)" >> ${output}

      echo ">>> Listing" >> ${output}
      kubectl get ${crd} --all-namespaces >> ${output}

      echo ">>> Details" >> ${output}
      if [[ "${crd}" == "secrets" ]]; then
        echo "Secrets are ignored for security reasons" >> ${output}
      else
        kubectl get ${crd} --all-namespaces -o yaml >> ${output}
      fi
    fi
  done

  if function_exists dump_extra_cluster_state; then
    echo ">>> Extra dump" >> ${output}
    dump_extra_cluster_state >> ${output}
  fi
  echo "***************************************"
  echo "***         E2E TEST FAILED         ***"
  echo "***     End of information dump     ***"
  echo "***************************************"
}

# Create a test cluster with kubetest and call the current script again.
function create_test_cluster() {
  # Fail fast during setup.
  set -o errexit
  set -o pipefail

  if function_exists cluster_setup; then
    cluster_setup || fail_test "cluster setup failed"
  fi

  if [[ "${E2E_CLUSTER_VERSION:="latest"}" != "latest" ]]; then
    E2E_KIND_NODE_IMAGE="kindest/node:v${E2E_CLUSTER_VERSION}"
  fi

  echo "- E2E_KIND_NODE_IMAGE=${E2E_KIND_NODE_IMAGE:="kindest/node:latest"}"
  echo "- E2E_KIND_CONFIG_PATH=${E2E_KIND_CONFIG_PATH}"

  readonly E2E_KIND_NODE_IMAGE
  readonly E2E_KIND_CONFIG_PATH

  # Smallest cluster required to run the end-to-end-tests
  local CLUSTER_CREATION_ARGS=(
    --deployment=kind
    --kind-node-image="${E2E_KIND_NODE_IMAGE}"
    --kind-config-path="${E2E_KIND_CONFIG_PATH}"
    --test=false
    --up
  )
  # Assume test failed (see details in set_test_return_code()).
  set_test_return_code 1
  echo "Test script is ${E2E_SCRIPT}"
  # Set arguments for this script again
  local test_cmd_args="--run-tests"
  (( SKIP_KNATIVE_SETUP )) && test_cmd_args+=" --skip-knative-setup"
  [[ -n "${E2E_SCRIPT_CUSTOM_FLAGS[@]}" ]] && test_cmd_args+=" ${E2E_SCRIPT_CUSTOM_FLAGS[@]}"
  if (( ! SKIP_CLUSTER_TEARDOWN )); then
    local extra_flags=(--down)
  fi

  # Set a minimal kubernetes environment that satisfies kubetest
  # TODO(adrcunha): Remove once https://github.com/kubernetes/test-infra/issues/13029 is fixed.
  local kubedir="$(mktemp -d -t kubernetes.XXXXXXXXXX)"
  local test_wrapper="${kubedir}/e2e-test.sh"
  mkdir ${kubedir}/cluster
  ln -s "$(which kubectl)" ${kubedir}/cluster/kubectl.sh
  echo "#!/usr/bin/env bash" > ${test_wrapper}
  echo "cd $(pwd) && set -x" >> ${test_wrapper}
  echo "${E2E_SCRIPT} ${test_cmd_args}" >> ${test_wrapper}
  chmod +x ${test_wrapper}
  cd ${kubedir}

  # Create cluster and run the tests
  create_test_cluster_with_retries "${CLUSTER_CREATION_ARGS[@]}" \
    --test-cmd "${test_wrapper}" \
    ${extra_flags[@]} \
    ${EXTRA_KUBETEST_FLAGS[@]}
  echo "Test subprocess exited with code $?"
  # Ignore any errors below, this is a best-effort cleanup and shouldn't affect the test result.
  set +o errexit
  (( ! SKIP_CLUSTER_TEARDOWN )) && function_exists cluster_teardown && cluster_teardown
  local result=$(get_test_return_code)
  echo "Artifacts were written to ${ARTIFACTS}"
  echo "Test result code is ${result}"
  exit ${result}
}

# Parameters: $1..$n - any kubetest flags.
function create_test_cluster_with_retries() {
  local cluster_creation_log=/tmp/${E2E_BASE_NAME}-cluster_creation-log
  header "Creating test cluster ${E2E_CLUSTER_VERSION}"
  # Don't fail test for kubetest, as it might incorrectly report test failure
  # if teardown fails (for details, see success() below)
  set +o errexit
  run_go_tool k8s.io/test-infra/kubetest \
    kubetest "$@" 2>&1 | tee ${cluster_creation_log}

  # Exit if test succeeded
  [[ "$(get_test_return_code)" == "0" ]] && return 0
  return 1
}

# Setup the test cluster for running the tests.
function setup_test_cluster() {
  # Fail fast during setup.
  set -o errexit
  set -o pipefail

  header "Test cluster setup"
  kubectl get nodes

  header "Setting up test cluster"

  local k8s_cluster=$(kubectl config current-context)

  # Use default namespace for all subsequent kubectl commands in this context
  kubectl config set-context ${k8s_cluster} --namespace=default

  export KO_DATA_PATH="${REPO_ROOT_DIR}/.git"

  echo "- Cluster is ${k8s_cluster}"
  echo "- Docker is ${KO_DOCKER_REPO}"
  echo "- KO_DATA_PATH is ${KO_DATA_PATH}"


  trap teardown_test_resources EXIT

  # Handle failures ourselves, so we can dump useful info.
  set +o errexit
  set +o pipefail

  if (( ! SKIP_KNATIVE_SETUP )) && function_exists knative_setup; then
    knative_setup || fail_test "Knative setup failed"
  fi
  if function_exists test_setup; then
    test_setup || fail_test "test setup failed"
  fi
}

# Gets the exit of the test script.
# For more details, see set_test_return_code().
function get_test_return_code() {
  echo $(cat ${TEST_RESULT_FILE})
}

# Set the return code that the test script will return.
# Parameters: $1 - return code (0-255)
function set_test_return_code() {
  # kubetest teardown might fail and thus incorrectly report failure of the
  # script, even if the tests pass.
  # We store the real test result to return it later, ignoring any teardown
  # failure in kubetest.
  echo -n "$1"> ${TEST_RESULT_FILE}
}

# Signal (as return code and in the logs) that all E2E tests passed.
function success() {
  set_test_return_code 0
  echo "**************************************"
  echo "***        E2E TESTS PASSED        ***"
  echo "**************************************"
  exit 0
}

# Exit test, dumping current state info.
# Parameters: $1 - error message (optional).
function fail_test() {
  set_test_return_code 1
  [[ -n $1 ]] && echo "ERROR: $1"
  dump_cluster_state
  exit 1
}

RUN_TESTS=0
SKIP_CLUSTER_SETUP=0
SKIP_KNATIVE_SETUP=0
SKIP_KNATIVE_TEARDOWN=0
SKIP_CLUSTER_TEARDOWN=0
E2E_SCRIPT=""
E2E_CLUSTER_VERSION=""
GKE_ADDONS=""
EXTRA_KUBETEST_FLAGS=()
E2E_SCRIPT_CUSTOM_FLAGS=()


# Parse flags and initialize the test cluster.
function initialize() {

  E2E_SCRIPT="$(get_canonical_path $0)"

  cd ${REPO_ROOT_DIR}
  while [[ $# -ne 0 ]]; do
    local parameter=$1
    # Try parsing flag as a custom one.
    if function_exists parse_flags; then
      parse_flags $@
      local skip=$?
      if [[ ${skip} -ne 0 ]]; then
        # Skip parsed flag (and possibly argument) and continue
        # Also save it to it's passed through to the test script
        for ((i=1;i<=skip;i++)); do
          E2E_SCRIPT_CUSTOM_FLAGS+=("$1")
          shift
        done
        continue
      fi
    fi

    # Try parsing flag as a standard one.
    case ${parameter} in
      --run-tests) RUN_TESTS=1 ;;
      --skip-cluster-setup) SKIP_CLUSTER_SETUP=1 ;;
      --skip-knative-setup) SKIP_KNATIVE_SETUP=1 ;;
      --skip-knative-teardown) SKIP_KNATIVE_TEARDOWN=1 ;;
      --skip-cluster-teardown) SKIP_CLUSTER_TEARDOWN=1 ;;
      *)
        [[ $# -ge 2 ]] || abort "missing parameter after $1"
        shift
        case ${parameter} in
          --cluster-version) E2E_CLUSTER_VERSION=$1 ;;
          --kubetest-flag) EXTRA_KUBETEST_FLAGS+=($1) ;;
          *) abort "unknown option ${parameter}" ;;
        esac
    esac
    shift
  done

  readonly RUN_TESTS
  readonly EXTRA_KUBETEST_FLAGS
  readonly SKIP_CLUSTER_SETUP
  readonly SKIP_KNATIVE_SETUP
  readonly SKIP_KNATIVE_TEARDOWN
  readonly SKIP_CLUSTER_TEARDOWN

  if (( ! RUN_TESTS )); then
    create_test_cluster
  else
    setup_test_cluster
  fi
}
