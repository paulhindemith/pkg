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
# https://github.com/knative/serving/blob/0dd798db6f946906c4512311e40107e1c61a2a25/hack/update-deps.sh
# The change history can be obtained by looking at the differences from the
# following commit that added as the original source code.
# aad80ea8fad061c6e44dec40cdf1eb219a602350

REPO_ROOT_DIR=$(git rev-parse --show-toplevel)
source ${REPO_ROOT_DIR}/vendor/knative.dev/test-infra/scripts/library.sh

set -o errexit
set -o nounset
set -o pipefail

cd ${REPO_ROOT_DIR}

# Ensure we have everything we need under vendor/
echo ">> dep ensure"
dep ensure

rm -rf $(find vendor/ -name 'OWNERS')
rm -rf $(find vendor/ -name '*_test.go')

echo ">> update_licenses"
update_licenses conformance/hello-world-serving/third_party/VENDOR-LICENSE "./conformance/hello-world-serving/cmd/*"
