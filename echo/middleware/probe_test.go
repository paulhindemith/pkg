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

func TestProbeMiddleware(t *testing.T) {
	// Setup
	probeHeader := "hello"
	probeHeaderValue := "world"
	e := echo.New()
	e.Use(K8sProbe(probeHeader, probeHeaderValue))
	req := httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set(probeHeader, probeHeaderValue)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if http.StatusOK != rec.Code {
		t.Errorf("Unexpected http status code. Expect: %v, but got: %v",
			http.StatusOK, rec.Code)
	}
	if probeHeaderValue != rec.Body.String() {
		t.Errorf("Unexpected http response body. Expect: %v, but got: %v",
			probeHeaderValue, rec.Body.String())
	}
}
