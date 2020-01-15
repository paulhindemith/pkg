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
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

// K8sProbe returns a ready status.
func K8sProbe(h string, v string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if val := c.Request().Header.Get(h); val != "" {
				if val != v {
					return &echo.HTTPError{
						Code:    http.StatusBadRequest,
						Message: fmt.Sprintf("unexpected probe header value: %q", val),
					}
				}
			}
			return c.String(http.StatusOK, v)
		}
	}
}
