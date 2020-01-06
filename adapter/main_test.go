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
	"log"
	"time"
	"context"
	"os"
	"syscall"
	"testing"
	"net/http"
	"strings"
	"sync"
	"golang.org/x/sync/errgroup"

	"knative.dev/pkg/signals"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/apimachinery/pkg/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var (
	startAdapter = false
	receivedSignal = false
)

type testAdapter struct {
	mutex sync.Mutex
}

func createConfigmap(name string, data map[string]string, kc kubernetes.Interface) error {
	_, err := kc.CoreV1().ConfigMaps("ns").Create(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
        Name:      name,
        Namespace: "ns",
    },
		Data: data,
	})
	return err
}

func updateConfigmap(name string, data map[string]string, kc kubernetes.Interface) error {
	_, err := kc.CoreV1().ConfigMaps("ns").Update(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
        Name:      name,
        Namespace: "ns",
    },
		Data: data,
	})
	return err
}

func TestMainWithClient(t *testing.T) {
	os.Setenv("SYSTEM_POD_NAME", "pn")
	os.Setenv("SYSTEM_NAMESPACE", "ns")
	os.Setenv("CONFIG_LOGGING_NAME", "config-logging")
	os.Setenv("CONFIG_OBSERVABILITY_NAME", "config-observability")
	os.Setenv("KUBERNETES_MIN_VERSION", "v1.15.0")
	os.Setenv("KO_DATA_PATH", "../.git/")

	ctx := signals.NewContext()
	kubeClient := fakeclientset.NewSimpleClientset()
	fakeDiscovery, _ := kubeClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{
		GitVersion: "v1.15.0",
	}

	// add fake config
	createConfigmap("config-logging", map[string]string{}, kubeClient)
	createConfigmap("config-observability", map[string]string{"profiling.enable": "true"}, kubeClient)

	ta := testAdapter{}

	go MainWithClient(
		"mycomponent",
		func(ctx context.Context) Adapter {
			return &ta
		},
		ctx,
		kubeClient,
	)
	eg := errgroup.Group{}
	// ----------------------------------------------------------------------------
	// Receive Adapter should have started.
	eg.Go(func() error {
		return wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
			ta.mutex.Lock()
			defer ta.mutex.Unlock()
			return startAdapter, nil
		})
	})
	// ----------------------------------------------------------------------------
	// Profiling Server should have started.
	// Ready for profiling request.
	req, _ := http.NewRequest("GET", "http://localhost:8008/debug/pprof/heap", nil)
	httpClient := http.Client{}
	eg.Go(func() error {
		return wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
			resp, _ := httpClient.Do(req)
			if resp != nil {
				return http.StatusOK == resp.StatusCode, nil
			}
			return false, nil
		})
	})

	if err := eg.Wait(); err != nil {
	    t.Fatal(err)
	}
	// ----------------------------------------------------------------------------
	// Profiling Server should have not been running.
	// To be disabled Profiling Server.
	updateConfigmap("config-observability", map[string]string{"profiling.enable": "false"}, kubeClient)
	// then
	wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
    resp, _ := httpClient.Do(req)
		if resp != nil {
			return http.StatusNotFound == resp.StatusCode, nil
		}
		return false, nil
	})
	// ----------------------------------------------------------------------------
	// Ensure Profiling Server restarting.
	// To be enabled Profiling Server.
	updateConfigmap("config-observability", map[string]string{"profiling.enable": "true"}, kubeClient)
	// then
	wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
    resp, _ := httpClient.Do(req)
		if resp != nil {
			return http.StatusOK == resp.StatusCode, nil
		}
		return false, nil
	})
	// ----------------------------------------------------------------------------
	// Each Server should gracefully shutdown.
	// Send Signal.
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(syscall.SIGTERM)

	eg.Go(func() error {
		return wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
			ta.mutex.Lock()
			defer ta.mutex.Unlock()
			return receivedSignal, nil
		})
	})
	eg.Go(func() error {
		return wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
			_, err := httpClient.Do(req)
			return strings.Contains(err.Error(), "connection refused"), nil
		})
	})

	if err := eg.Wait(); err != nil {
			t.Fatal(err)
	}
}

func (m *testAdapter) Start(stopCh <-chan struct{}) {
	m.mutex.Lock()
	startAdapter = true
	m.mutex.Unlock()
	<-stopCh
	log.Print("Shutdown Receive Adapter")
	m.mutex.Lock()
	receivedSignal = true
	m.mutex.Unlock()
}
