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
# limitations under the License.export DISABLE_MD_LINTING=1

set -o errexit
PRESUBMIT_TEST_FAIL_FAST=1

source $(dirname $0)/../vendor/knative.dev/test-infra/scripts/presubmit-tests.sh

echo ">> ./vendor/github.com/paulhindemith/dev-infra/hack/boilerplate/ensure-boilerplate.sh"
./vendor/github.com/paulhindemith/dev-infra/hack/boilerplate/ensure-boilerplate.sh Paulhindemith

echo ">> go fmt ./..."
go fmt ./...

echo ">> go vet ./..."
go vet ./...

main --build-tests --unit-tests
