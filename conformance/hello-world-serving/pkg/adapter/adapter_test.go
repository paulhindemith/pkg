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
	"net/http"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	pkglogging "knative.dev/pkg/logging"
)

var (
	startAdapter   = false
	receivedSignal = false
)

func TestAdapter(t *testing.T) {
	os.Setenv("NAME", "test")
	os.Setenv("SYSTEM_NAMESPACE", "ns")

	ctx := context.Background()
	ctx = pkglogging.WithLogger(ctx, zap.NewExample().Sugar())
	a, _ := NewAdapter(ctx).(*Adapter)

	ctx, cancel := context.WithCancel(ctx)
	go a.Start(ctx.Done())

	req, _ := http.NewRequest("GET", "http://localhost:8080/hello", nil)
	httpClient := http.Client{}
	wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
		resp, _ := httpClient.Do(req)
		if resp != nil {
			return http.StatusOK == resp.StatusCode, nil
		}
		return false, nil
	})

	req, _ = http.NewRequest("GET", "http://localhost:8080/healthz", nil)
	req.Header.Set("hello", "world")
	wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
		resp, _ := httpClient.Do(req)
		if resp != nil {
			return http.StatusOK == resp.StatusCode, nil
		}
		return false, nil
	})
	// It implies sending Signal.
	if a.closed {
		t.Fatal("Unexpect closed Receive Server.")
	}
	cancel()
	wait.PollImmediate(10*time.Millisecond, 5*time.Second, func() (bool, error) {
		a.mutex.Lock()
		defer a.mutex.Unlock()
		return a.closed, nil
	})
}
