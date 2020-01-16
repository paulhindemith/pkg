// +build e2e

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
*/

package e2e

import (
	"testing"

	"knative.dev/serving/test"
	v1test "knative.dev/serving/test/v1"
	pkgTest "knative.dev/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/paulhindemith/pkg/test/e2e"
)

func TestHelloWorld(t *testing.T) {
	t.Parallel()

	clients := e2e.SetupWithNamespace(t, "hello-world")

	names := test.ResourceNames{
		Route: "hello",
		Config: "hello",
		Service: "hello",
		Image:   "hello-cb02de24fa4a72f88a592e14da85d6d8",
	}

	var err error

	t.Log("Waiting for the Service")
	names.Revision, err = v1test.WaitForServiceLatestRevision(clients, names)
	if err != nil {
		t.Fatalf("Service %s was not updated with the Revision: %v", names.Service, err)
	}

	service, err := clients.ServingClient.Services.Get(names.Service, metav1.GetOptions{})

	url := service.Status.URL.URL()
	url.Path = "/hello"

	var opt interface{}
	if _, err := pkgTest.WaitForEndpointState(
		clients.KubeClient,
		t.Logf,
		url,
		pkgTest.MatchesAllOf(pkgTest.IsStatusOK, pkgTest.MatchesBody("Hello, World!")),
		"HelloWorldServesText",
		test.ServingFlags.ResolvableDomain,
		opt); err != nil {
		t.Fatalf("The endpoint %s for Route %s didn't serve the expected text %q: %v", url, names.Route, "Hello, World!", err)
	}

}
