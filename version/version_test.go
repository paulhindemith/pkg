/*
Copyright 2019 The Knative Authors
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
https://github.com/knative/pkg/tree/daee70aa95b5f4d190df8a2f37ce7117365e3b02/version
The change history can be obtained by looking at the differences from the
following commit that added as the original source code.
0444bb6558ec8a6f246ed15f1fab6754c634f959
*/

package version

import (
	"errors"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/version"
)

type testVersioner struct {
	version string
	err     error
}

func (t *testVersioner) ServerVersion() (*version.Info, error) {
	return &version.Info{GitVersion: t.version}, t.err
}

func TestVersionCheck(t *testing.T) {
	tests := []struct {
		name                 string
		actualVersion        *testVersioner
		expectMinimumVersion string
		wantError            bool
	}{{
		name:                 "greater version (patch)",
		actualVersion:        &testVersioner{version: "v1.15.1"},
		expectMinimumVersion: "v1.15.0",
	}, {
		name:                 "greater version (minor)",
		actualVersion:        &testVersioner{version: "v1.16.0"},
		expectMinimumVersion: "v1.15.0",
	}, {
		name:                 "same version",
		actualVersion:        &testVersioner{version: "v1.15.0"},
		expectMinimumVersion: "v1.15.0",
	}, {
		name:                 "smaller version",
		actualVersion:        &testVersioner{version: "v1.14.3"},
		expectMinimumVersion: "v1.15.0",
		wantError:            true,
	}, {
		name:          "not set",
		actualVersion: &testVersioner{version: "v1.15.0"},
		wantError:     true,
	}, {
		name:                 "error while fetching",
		actualVersion:        &testVersioner{err: errors.New("random error")},
		expectMinimumVersion: "v1.15.0",
		wantError:            true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv("SYSTEM_KUBERNETES_MIN_VERSION", test.expectMinimumVersion)
			defer os.Setenv("SYSTEM_KUBERNETES_MIN_VERSION", "")

			err := CheckMinimumVersion(test.actualVersion, os.Getenv("SYSTEM_KUBERNETES_MIN_VERSION"))
			if err == nil && test.wantError {
				t.Errorf("Expected an error for minimum: %q, actual: %v", os.Getenv("SYSTEM_KUBERNETES_MIN_VERSION"), test.actualVersion)
			}

			if err != nil && !test.wantError {
				t.Errorf("Expected no error but got %v for minimum: %q, actual: %v", err, os.Getenv("SYSTEM_KUBERNETES_MIN_VERSION"), test.actualVersion)
			}
		})
	}
}
