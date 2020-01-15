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
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	pkglogging "knative.dev/pkg/logging"
	"knative.dev/pkg/profiling"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubernetes "k8s.io/client-go/kubernetes"

	"github.com/kelseyhightower/envconfig"
	"github.com/paulhindemith/pkg/logkey"
	"github.com/paulhindemith/pkg/version"
)

type SystemConfig struct {
	PodName                    string `split_words:"true"`
	ProfilePort                int    `default:"8018" split_words:"true"`
	LoggingConfigMapName       string `default:"config-logging" split_words:"true"`
	ObservabilityConfigMapName string `default:"config-observability" split_words:"true"`
	KubernetesMinVersion       string `split_words:"true" require:"true"`
}

// Adapter must have Start method.
type Adapter interface {
	Start(stopCh <-chan struct{})
}

// AdapterConstructor returns Adapter.
type AdapterConstructor func(ctx context.Context) Adapter

// GetLoggingConfig returns configmap typed pkglogging.Config from ApiServer.
func GetLoggingConfig(kc kubernetes.Interface, loggingConfigMapName string) (*pkglogging.Config, error) {
	loggingConfigMap, err := kc.CoreV1().ConfigMaps(system.Namespace()).Get(loggingConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pkglogging.NewConfigFromConfigMap(loggingConfigMap)
}

func Main(component string, ctor AdapterConstructor) {
	var (
		masterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
		kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	)
	flag.Parse()

	// Set up signals so we handle the first shutdown signal gracefully.
	ctx := signals.NewContext()

	cfg, err := sharedmain.GetConfig(*masterURL, *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	for _, ci := range injection.Default.GetClients() {
		ctx = ci(ctx, cfg)
	}

	MainWithContext(component, ctor, ctx)
}

// MainWithContext
// - watches profiling and logging configmap
// - defines zap logger
// - runs injected AdapterConstructor which is called Receive Server
// - runs Profiling Server
// - Sends Signal to injected AdapterConstructor and shutdown Profiling Server when received
func MainWithContext(component string, ctor AdapterConstructor, ctx context.Context) {
	log.Printf("Registering %d clients", len(injection.Default.GetClients()))
	log.Printf("Registering %d informer factories", len(injection.Default.GetInformerFactories()))
	log.Printf("Registering %d informers", len(injection.Default.GetInformers()))

	var err error

	// Set Environment Variable.
	var sc SystemConfig
	if err = envconfig.Process("system", &sc); err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%v", sc)

	// Set kube client.
	kc := kubeclient.Get(ctx)

	// We sometimes startup faster than we can reach kube-api. Poll on failure to prevent us terminating
	if perr := wait.PollImmediate(time.Second, 60*time.Second, func() (bool, error) {
		err := version.CheckMinimumVersion(kc.Discovery(), sc.KubernetesMinVersion)
		if err != nil {
			log.Printf("Failed to get k8s version %v", err)
		}
		return err == nil, nil
	}); perr != nil {
		log.Fatal("Timed out attempting to get k8s version: ", err)
	}

	// Set up our logger.
	loggingConfig, err := GetLoggingConfig(kc, sc.LoggingConfigMapName)
	if err != nil {
		log.Fatal("Error loading/parsing logging configuration: ", err)
	}
	logger, atomicLevel := pkglogging.NewLoggerFromConfig(loggingConfig, component)

	logger = logger.With(zap.String(logkey.Name, component),
		zap.String(logkey.Pod, sc.PodName))
	ctx = pkglogging.WithLogger(ctx, logger)
	defer flush(logger)

	profilingHandler := profiling.NewHandler(logger, false)

	configMapWatcher := configmap.NewInformedWatcher(kc, system.Namespace())
	configMapWatcher.Watch(sc.LoggingConfigMapName, pkglogging.UpdateLevelFromConfigMap(logger, atomicLevel, component))
	configMapWatcher.Watch(sc.ObservabilityConfigMapName, profilingHandler.UpdateFromConfigMap)

	if err = configMapWatcher.Start(ctx.Done()); err != nil {
		logger.Fatalw("Failed to start configuration manager", zap.Error(err))
	}

	adapter := ctor(ctx)

	logger.Info("Starting Receive Adapter")
	go adapter.Start(ctx.Done())

	ps := &http.Server{
		Addr:    ":" + strconv.Itoa(sc.ProfilePort),
		Handler: profilingHandler,
	}

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(ps.ListenAndServe)

	// This will block until either a signal arrives or one of the grouped functions
	// returns an error.
	<-egCtx.Done()

	ps.Shutdown(context.Background())
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
