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

	"knative.dev/serving/pkg/network"
  "github.com/labstack/echo"
)

type (
	// K8sHealthzConfig defines the config for K8sHealthz middleware.
	K8sHealthzConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper func(echo.Context) bool
    HealthCheck func() error
	}
)

var (
	// DefaultK8sHealthzConfig is the default K8sHealthz middleware config.
	DefaultK8sHealthzConfig = K8sHealthzConfig{
		Skipper: func(c echo.Context) bool {
      return !network.IsKubeletProbe(c.Request())
    },
    HealthCheck: func() error {return nil},
	}
)

// K8sHealthz returns a X-Request-ID middleware.
func K8sHealthz() echo.MiddlewareFunc {
	return K8sHealthzWithConfig(DefaultK8sHealthzConfig)
}

// K8sHealthzWithConfig returns a X-Request-ID middleware with config.
func K8sHealthzWithConfig(config K8sHealthzConfig) echo.MiddlewareFunc {
  if config.Skipper == nil {
		config.Skipper = DefaultK8sHealthzConfig.Skipper
	}
  if config.HealthCheck == nil {
		config.HealthCheck = DefaultK8sHealthzConfig.HealthCheck
	}
  return func(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
      if config.Skipper(c) {
  			return next(c)
  		}
      if err := config.HealthCheck(); err != nil {
        // use zap.
  			c.Logger().Warnf("Healthcheck failed: %v", err)
        return &echo.HTTPError{
    			Code:     http.StatusInternalServerError,
    			Message:  err.Error(),
    			Internal: err,
    		}
  		}
      return c.String(http.StatusOK, "")
    }
  }
}
