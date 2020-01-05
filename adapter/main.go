/*
Copyright 2018 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    https://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2020 Paulhindemith

The original source code can be referenced from the link below.
https://github.com/knative/pkg/blob/80da64a31cc4cec93ce633f0634387fcffadddda/injection/sharedmain/main.go
The change history can be obtained by looking at the differences from the
following commit that added as the original source code.
1df0da6786f61de259f1e78ab475d84132ca350e
*/

package adapter

import (
  "context"
	"log"
  "time"
  "os"
	"net/http"

  "go.uber.org/zap"
  "golang.org/x/sync/errgroup"

  "knative.dev/pkg/profiling"
  "knative.dev/pkg/metrics"
  "knative.dev/pkg/signals"
  "knative.dev/pkg/configmap"
  "knative.dev/pkg/system"
  "knative.dev/pkg/injection"
  pkglogging "knative.dev/pkg/logging"
  "knative.dev/pkg/version"
  kubeclient "knative.dev/pkg/client/injection/kube/client"

  "k8s.io/apimachinery/pkg/util/wait"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  kubernetes "k8s.io/client-go/kubernetes"

  "github.com/paulhindemith/pkg/logkey"
)

const (
  PodNameEnvKey = "SYSTEM_POD_NAME"
)

type Adapter interface {
	Start(stopCh <-chan struct{}) error
}

type AdapterConstructor func(ctx context.Context) Adapter

func GetLoggingConfig(kc kubernetes.Interface) (*pkglogging.Config, error) {
  loggingConfigMap, err := kc.CoreV1().ConfigMaps(system.Namespace()).Get(pkglogging.ConfigMapName(), metav1.GetOptions{})
  if err != nil {
    return nil, err
  }
  return pkglogging.NewConfigFromConfigMap(loggingConfigMap)
}

func Main(component string, ctor AdapterConstructor) {
  // Set up signals so we handle the first shutdown signal gracefully.
	ctx := signals.NewContext()
  kc := kubeclient.Get(ctx)
	MainWithClient(component, ctor, ctx, kc)
}

func MainWithClient(component string, ctor AdapterConstructor, ctx context.Context, kc kubernetes.Interface) {
  log.Printf("Registering %d clients", len(injection.Default.GetClients()))
	log.Printf("Registering %d informer factories", len(injection.Default.GetInformerFactories()))
	log.Printf("Registering %d informers", len(injection.Default.GetInformers()))

  var err error

  // We sometimes startup faster than we can reach kube-api. Poll on failure to prevent us terminating
	if perr := wait.PollImmediate(time.Second, 60*time.Second, func() (bool, error) {
    err := version.CheckMinimumVersion(kc.Discovery())
		if err != nil {
			log.Printf("Failed to get k8s version %v", err)
		}
		return err == nil, nil
	}); perr != nil {
		log.Fatal("Timed out attempting to get k8s version: ", err)
	}

	// Set up our logger.
  loggingConfig, err := GetLoggingConfig(kc)
	if err != nil {
		log.Fatal("Error loading/parsing logging configuration: ", err)
	}
  logger, atomicLevel := pkglogging.NewLoggerFromConfig(loggingConfig, component)

	logger = logger.With(zap.String(logkey.Name, component),
		zap.String(logkey.Pod, os.Getenv(PodNameEnvKey)))
	ctx = pkglogging.WithLogger(ctx, logger)
	defer flush(logger)

  profilingHandler := profiling.NewHandler(logger, false)

  configMapWatcher := configmap.NewInformedWatcher(kc, system.Namespace())
  configMapWatcher.Watch(pkglogging.ConfigMapName(), pkglogging.UpdateLevelFromConfigMap(logger, atomicLevel, component))
  configMapWatcher.Watch(metrics.ConfigMapName(),profilingHandler.UpdateFromConfigMap)

  if err = configMapWatcher.Start(ctx.Done()); err != nil {
  	logger.Fatalw("Failed to start configuration manager", zap.Error(err))
  }

  adapter := ctor(ctx)

  logger.Info("Starting Receive Adapter")
  go adapter.Start(ctx.Done())

  eg, egCtx := errgroup.WithContext(ctx)
  profilingServer := profiling.NewServer(profilingHandler)
  eg.Go(profilingServer.ListenAndServe)

  // This will block until either a signal arrives or one of the grouped functions
	// returns an error.
	<-egCtx.Done()

	profilingServer.Shutdown(context.Background())
	// Don't forward ErrServerClosed as that indicates we're already shutting down.
	if err := eg.Wait(); err != nil && err != http.ErrServerClosed {
		logger.Errorw("Error while running server", zap.Error(err))
	} else {
    logger.Info("Shutdowned Profiling Server")
  }
}

func flush(logger *zap.SugaredLogger) {
	logger.Sync()
  os.Stdout.Sync()
	os.Stderr.Sync()
}
