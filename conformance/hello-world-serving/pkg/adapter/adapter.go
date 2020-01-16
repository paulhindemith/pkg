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

package adapter

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	pkglogging "knative.dev/pkg/logging"

	"github.com/paulhindemith/pkg/adapter"
	"github.com/paulhindemith/pkg/conformance/hello-world-serving/pkg/hello"
	"github.com/paulhindemith/pkg/echo/middleware"
)

// adapterConfig defines the config for Adapter from environement variable.
type adapterConfig struct {
	Name      string `envconfig:"NAME" required:"true"`
	Namespace string `envconfig:"SYSTEM_NAMESPACE" required:"true"`
}

// Adapter has config, logger from ctx, state of closed.
type Adapter struct {
	config *adapterConfig
	logger *zap.Logger
	mutex  sync.Mutex
	closed bool
}

// NewAdapter returns Adapter.
func NewAdapter(ctx context.Context) adapter.Adapter {
	var config adapterConfig
	envconfig.Process("", &config)
	return &Adapter{
		config: &config,
		logger: pkglogging.FromContext(ctx).Desugar(),
	}
}

// Start is defined in adapter.Adapter.
func (a *Adapter) Start(stopCh <-chan struct{}) {
	a.logger.Info("Starting with config: ",
		zap.String("Name", a.config.Name),
		zap.String("Namespace", a.config.Namespace),
	)

	e := hello.NewRouter()

	e.Use(middleware.K8sProbe("hello", "world"))

	errCh := make(chan error, 1)
	// Start server
	go func() {
		if err := e.Start(":8080"); err != http.ErrServerClosed {
			a.logger.Info("shutting down the server")
			errCh <- fmt.Errorf("%s server failed: %w", a.config.Name, err)
		}
	}()

	// Wait for the signal to drain or receving error.
	select {
	case <-stopCh:
		a.logger.Info("Received SIGTERM")
	case err := <-errCh:
		a.logger.Error("Failed to run Receive Server.", zap.String("error", err.Error()))
	}
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		a.logger.Fatal("Failed to shutdown Receive Server.", zap.String("error", err.Error()))
		// Do not handle error.
		return
	}
	a.logger.Info("Servers shutdown.")
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.closed = true
}
