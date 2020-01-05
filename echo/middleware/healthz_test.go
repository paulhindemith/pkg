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

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
)

func TestHealthzMiddleware(t *testing.T) {
	// Setup
	e := echo.New()
  e.Use(K8sHealthzWithConfig(DefaultK8sHealthzConfig))
  req := httptest.NewRequest("GET", "/healthz", nil)
  req.Header.Set("User-Agent", "kube-probe/1.15")
  rec := httptest.NewRecorder()

  e.ServeHTTP(rec, req)

  if http.StatusOK != rec.Code {
    t.Errorf("Unexpected http status code. Expect: %v, but got: %v",
      http.StatusOK, rec.Code)
  }
}
