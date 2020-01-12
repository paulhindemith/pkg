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
	"fmt"

	"github.com/rogpeppe/go-internal/semver"
	"k8s.io/apimachinery/pkg/version"
)

// ServerVersioner is an interface to mock the `ServerVersion`
// method of the Kubernetes client's Discovery interface.
// In an application `kubeClient.Discovery()` can be used to
// suffice this interface.
type ServerVersioner interface {
	ServerVersion() (*version.Info, error)
}

// CheckMinimumVersion checks if the currently installed version of
// Kubernetes is compatible with the minimum version required.
// Returns an error if its not.
//
// A Kubernetes discovery client can be passed in as the versioner
// like `CheckMinimumVersion(kubeClient.Discovery(), "v1.15.0")`.
func CheckMinimumVersion(versioner ServerVersioner, minimumVersion string) error {
	if minimumVersion == "" {
		return errors.New("minimumVersion must be set.")
	}

	v, err := versioner.ServerVersion()
	if err != nil {
		return err
	}
	currentVersion := semver.Canonical(v.String())

	// Compare returns 1 if the first version is greater than the
	// second version.
	if semver.Compare(minimumVersion, currentVersion) == 1 {
		return fmt.Errorf("kubernetes version %q is not compatible, need at least %q",
			currentVersion, minimumVersion)
	}
	return nil
}
