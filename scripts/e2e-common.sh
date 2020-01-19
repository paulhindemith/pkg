#!/usr/bin/env bash

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
# https://github.com/knative/serving/tree/1957dff9cea156d158cfaa2fde3c6de76dfe6de1/test/e2e-common.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# b65b5ef3737a22497b2358b08ec8c91556c23a0d


source ${REPO_ROOT_DIR:=$(git rev-parse --show-toplevel)}/scripts/e2e-local-tests.sh

[[ -z ${TEST_NAMESPACE} ]] && echo "TEST_NAMESPACE is not set." && exit 1


CERT_MANAGER_VERSION="0.12-latest"
ISTIO_VERSION="1.4-latest"
KNATIVE_VERSION="0.11-latest"
DOCKER_CONFIG_FILE=""

UNINSTALL_LIST=()

# Parse our custom flags.
function parse_flags() {
  case "$1" in
    --istio-version)
      [[ $2 =~ ^[0-9]+\.[0-9]+(\.[0-9]+|\-latest)$ ]] || abort "istio-version format must be '[0-9].[0-9].[0-9]' or '[0-9].[0-9]-latest"
      readonly ISTIO_VERSION=$2
      return 2
      ;;
    --knative-version)
      [[ $2 =~ ^[0-9]+\.[0-9]+(\.[0-9]+|\-latest)$ ]] || abort "knative-version format must be '[0-9].[0-9].[0-9]' or '[0-9].[0-9]-latest"
      readonly KNATIVE_VERSION=$2
      return 2
      ;;
    --cert-manager-version)
      [[ $2 =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || abort "version format must be '[0-9].[0-9].[0-9]'"
      readonly CERT_MANAGER_VERSION=$2
      return 2
      ;;
    --https)
      readonly HTTPS=1
      return 1
      ;;
    --local)
      readonly TEST_ENV="local"
      return 1
      ;;
    --namespace)
      readonly TEST_NAMESPACE=$2
      return 2
      ;;
    --dockerconfigjson)
      readonly DOCKER_CONFIG_FILE=$2
      return 2
      ;;
  esac
  return 0
}

# Installs Knative Serving in the current cluster, and waits for it to be ready.
# If no parameters are passed, installs the current source-based build, unless custom
# YAML files were passed using the --custom-yamls flag.
# Parameters: $1 - Knative Serving YAML file
#             $2 - Knative Monitoring YAML file (optional)
function install_knative_serving() {
  install_knative_serving_standard "$1" "$2"
}

function install_istio() {
  local istio_base="${REPO_ROOT_DIR}/vendor/github.com/paulhindemith/dev-infra/third_party/istio/istio-${ISTIO_VERSION}"
  INSTALL_ISTIO_CRD_YAML="${istio_base}/istio-crds.yaml"
  case ${TEST_ENV:="local"} in
    "local" ) INSTALL_ISTIO_YAML="${istio_base}/istio-local.yaml" ;;
  esac

  echo "Istio CRD YAML: ${INSTALL_ISTIO_CRD_YAML}"
  echo "Istio YAML: ${INSTALL_ISTIO_YAML}"

  echo ">> Bringing up Istio"
  echo ">> Running Istio CRD installer"
  kubectl apply -f "${INSTALL_ISTIO_CRD_YAML}" || return 1
  wait_until_batch_job_complete istio-system || return 1
  UNINSTALL_LIST+=( "${INSTALL_ISTIO_CRD_YAML}" )

  echo ">> Running Istio"
  kubectl apply -f "${INSTALL_ISTIO_YAML}" || return 1
  UNINSTALL_LIST+=( "${INSTALL_ISTIO_YAML}" )

  echo ">> Patching Istio"
  # There are reports of Envoy failing (503) when istio-pilot is overloaded.
  # We generously add more pilot instances here to reduce flakes.
  if kubectl get hpa -n istio-system istio-pilot 2>/dev/null; then
    kubectl patch hpa -n istio-system istio-pilot \
            --patch '{"spec": {"minReplicas": 3, "maxReplicas": 10, "targetCPUUtilizationPercentage": 60}}' || return 1
  else
    # Some versions of Istio don't provide an HPA for pilot.
    kubectl autoscale -n istio-system deploy istio-pilot --min=3 --max=10 --cpu-percent=60 || return 1
  fi

  # If the yaml for the Istio Ingress controller is passed, then install it.
  if [[ -n "$1" ]]; then
    echo ">> Installing Istio Ingress"
    echo "Istio Ingress YAML: ${1}"
    # We apply a filter here because when we're installing from a pre-built
    # bundle then the whole bundle it passed here.  We use ko because it has
    # better filtering support for CRDs.
    ko apply -f "${1}" --selector=networking.knative.dev/ingress-provider=istio || return 1
    UNINSTALL_LIST+=( "${1}" )
  fi
}

# Installs Knative Serving in the current cluster, and waits for it to be ready.
# If no parameters are passed, installs the current source-based build.
# Parameters: $1 - Knative Serving YAML file
#             $2 - Knative Monitoring YAML file (optional)
function install_knative_serving_standard() {
  readonly INSTALL_CERT_MANAGER_YAML="${REPO_ROOT_DIR}/vendor/github.com/paulhindemith/dev-infra/third_party/cert-manager/cert-manager-${CERT_MANAGER_VERSION}/cert-manager.yaml"
  readonly INSTALL_KNATIVE_SERVING_YAML="${REPO_ROOT_DIR}/vendor/github.com/paulhindemith/dev-infra/third_party/knative/knative-${KNATIVE_VERSION}/serving.yaml"

  echo ">> Installing Cert-Manager"
  echo "Cert Manager YAML: ${INSTALL_CERT_MANAGER_YAML}"
  kubectl apply -f "${INSTALL_CERT_MANAGER_YAML}" --validate=false || return 1
  UNINSTALL_LIST+=( "${INSTALL_CERT_MANAGER_YAML}" )

  echo ">> Installing Knative serving"
  echo "Knative YAML: ${INSTALL_KNATIVE_SERVING_YAML}"
  # If we are installing from provided yaml, then only install non-istio bits here,
  # and if we choose to install istio below, then pass the whole file as the rest.
  # We use ko because it has better filtering support for CRDs.
  ko apply -f "${INSTALL_KNATIVE_SERVING_YAML}" --selector=networking.knative.dev/ingress-provider!=istio || return 1
  UNINSTALL_LIST+=( "${INSTALL_KNATIVE_SERVING_YAML}" )
  SERVING_ISTIO_YAML="${INSTALL_KNATIVE_SERVING_YAML}"

  install_istio "${SERVING_ISTIO_YAML}"

  echo ">> Configuring the default Ingress: istio.ingress.networking.knative.dev"
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
name: config-network
namespace: knative-serving
labels:
  serving.knative.dev/release: devel
data:
  ingress.class: "istio.ingress.networking.knative.dev"
EOF

  echo ">> Turning on prometheus"
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-observability
  namespace: knative-serving
data:
  profiling.enable: "false"
  metrics.request-metrics-backend-destination: prometheus
EOF

  echo ">> Patching activator HPA"
  # We set min replicas to 2 for testing multiple activator pods.
  kubectl -n knative-serving patch hpa activator --patch '{"spec":{"minReplicas":2}}' || return 1
}

# Check if we should use --https.
function use_https() {
  if (( HTTPS )); then
    echo "--https"
  else
    echo ""
  fi
}

# Check if we should use --https.
function use_https() {
  if (( HTTPS )); then
    echo "--https"
  else
    echo ""
  fi
}

# Uninstalls Knative Serving from the current cluster.
function knative_teardown() {
  if [[ -z "${UNINSTALL_LIST[@]}" ]]; then
    echo "install_knative_serving() was not called, nothing to uninstall"
    return 0
  fi
  echo ">> Uninstalling Knative serving"
  for i in ${!UNINSTALL_LIST[@]}; do
    # We uninstall elements in the reverse of the order they were installed.
    local YAML="${UNINSTALL_LIST[$(( ${#array[@]} - $i ))]}"
    echo ">> Bringing down YAML: ${YAML}"
    kubectl delete --ignore-not-found=true -f "${YAML}" || return 1
  done
}

# Create test resources and images
function test_setup() {
  echo ">> Setting up logging..."

  # Install kail if needed.
  if ! which kail > /dev/null; then
    bash <( curl -sfL https://raw.githubusercontent.com/boz/kail/master/godownloader.sh) -b "$GOPATH/bin"
  fi

  # Capture all logs.
  kail > ${ARTIFACTS}/k8s.log.txt &
  local kail_pid=$!
  # Clean up kail so it doesn't interfere with job shutting down
  trap "kill $kail_pid || true" EXIT

  echo ">> Creating test resources (test/config/)"
  ko apply ${KO_FLAGS} -f ${TEST_CONFIG_DIR:-test/config/} || return 1

  echo ">> Waiting for Serving components to be running..."
  wait_until_pods_running knative-serving || return 1

  echo ">> Waiting for Ingress provider to be running..."
  if [[ -n "${ISTIO_VERSION}" ]]; then
    wait_until_pods_running istio-system || return 1
  fi

  echo ">> Setting secrets for docker private registry"
  if [[ -n "${DOCKER_CONFIG_FILE}" ]]; then
    kubectl create secret -n ${TEST_NAMESPACE} generic regcred \
      --from-file=.dockerconfigjson=${DOCKER_CONFIG_FILE} \
      --type=kubernetes.io/dockerconfigjson
  fi

  echo ">> Call post_test_setup"
  if function_exists post_test_setup; then
    post_test_setup || fail_test "post test setup failed"
  fi
}

# Delete test resources
function test_teardown() {
  echo ">> Removing test resources (test/config/)"
  ko delete --ignore-not-found=true --now -f test/config/
  echo ">> Ensuring test namespaces are clean"
  kubectl delete all --all --ignore-not-found --now --timeout 60s -n ${TEST_NAMESPACE}
  kubectl delete --ignore-not-found --now --timeout 60s namespace ${TEST_NAMESPACE}
}
