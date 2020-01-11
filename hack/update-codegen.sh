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
# https://github.com/knative/pkg/tree/4deb5d83d26170faeef8e54e9ae4cd9b04ed81f8/hack/update-codegen.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# 25b1b61dd5c1fb6ba1c84c58b00343db4304da3a

set -o nounset
set -o pipefail

source $(dirname $0)/../vendor/knative.dev/test-infra/scripts/library.sh

# Knative Injection
${REPO_ROOT_DIR}/vendor/knative.dev/pkg/hack/generate-knative.sh "injection" \
  github.com/paulhindemith/pkg/client knative.dev/pkg/apis \
  "duck:v1alpha1,v1beta1,v1" \
  --go-header-file ${REPO_ROOT_DIR}/vendor/github.com/paulhindemith/dev-infra/hack/boilerplate/boilerplate.go.txt

OUTPUT_PKG="github.com/paulhindemith/pkg/client/injection/kube" \
VERSIONED_CLIENTSET_PKG="k8s.io/client-go/kubernetes" \
EXTERNAL_INFORMER_PKG="k8s.io/client-go/informers" \
  ${REPO_ROOT_DIR}/vendor/knative.dev/pkg/hack/generate-knative.sh "injection" \
    k8s.io/client-go \
    k8s.io/api \
    "admissionregistration:v1beta1 apps:v1 autoscaling:v1,v2beta1 batch:v1,v1beta1 core:v1 rbac:v1" \
    --go-header-file ${REPO_ROOT_DIR}/vendor/github.com/paulhindemith/dev-infra/hack/boilerplate/boilerplate.go.txt

# Make sure our dependencies are up-to-date
${REPO_ROOT_DIR}/hack/update-deps.sh
