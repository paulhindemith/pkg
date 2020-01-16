/*
Copyright 2020 Paulhindemith

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2020 Paulhindemith

The original source code can be referenced from the link below.
https://github.com/knative/serving/blob/68eac43dcaedd2eaed50b1b15f08b6cd28b093c0/test/e2e/e2e.go
The change history can be obtained by looking at the differences from the
following commit that added as the original source code.
c3bf5f353a7875a2ce2d083a190dffd5872a48a4
*/

package e2e

import (
  "testing"

  "knative.dev/serving/test"
  pkgTest "knative.dev/pkg/test"
)

// Setup creates the client objects needed in the e2e tests.
func Setup(t *testing.T) *test.Clients {
	return SetupWithNamespace(t, test.ServingNamespace)
}

// SetupWithNamespace creates the client objects needed in the e2e tests under the specified namespace.
func SetupWithNamespace(t *testing.T, namespace string) *test.Clients {
	pkgTest.SetupLoggingFlags()
	clients, err := test.NewClients(
		pkgTest.Flags.Kubeconfig,
		pkgTest.Flags.Cluster,
		namespace)
	if err != nil {
		t.Fatalf("Couldn't initialize clients: %v", err)
	}
	return clients
}
