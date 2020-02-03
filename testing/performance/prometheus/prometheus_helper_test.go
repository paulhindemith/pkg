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

package prometheus

import (
  "context"
  "testing"
  "net/http"

  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promauto"
  "github.com/prometheus/client_golang/prometheus/promhttp"

)

func TestPrometheusHelper(t *testing.T) {
  var (
    c *Clients
    err error
  )
  if c, err = NewClients(
    &Config{
      Name: "unit-test",
      Endpoint: "https://localhost:443",
      ClientCertFile: "/root/client.crt",
      ClientKeyFile : "/root/client.key",
      CAFile: "/root/ca.crt",
    }); err != nil {
    t.Fatal(err)
  }

  var opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
    Name: "test",
    Help: "test",
  })
  opsProcessed.Inc()

  m := http.NewServeMux()
  s := &http.Server{Addr: ":8080", Handler: m}
  m.Handle("/metrics", promhttp.Handler())
  go func() {
    if err = s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
      t.Fatal(err)
    }
  }()
  defer s.Shutdown(context.Background())
  if err = c.WaitForReady("unit-test"); err != nil {
    t.Fatal(err)
  }
}
