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

package hello

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHelloHandler(t *testing.T) {
	router := NewRouter()

	req := httptest.NewRequest("GET", "/hello", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if http.StatusOK != rec.Code {
		t.Errorf("Unexpected http status code. Expect: %v, but got: %v",
			http.StatusOK, rec.Code)
	}
	if helloMessage != rec.Body.String() {
		t.Errorf("Unexpected http response body. Expect: %v, but got: %v",
			helloMessage, rec.Body.String())
	}
}
