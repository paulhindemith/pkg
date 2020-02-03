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
	"strconv"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	vegeta "github.com/tsenart/vegeta/lib"

)

const (
	CodeLabel  = "code"
	MethodLabel = "method"
	URLLabel = "url"
	WarmupLabel = "warmup"
)

var (
	latencyDistribution = []float64{5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300, 350, 400, 450, 500, 600, 700, 800, 900, 1000, 2000, 5000, 10000, 20000, 50000, 100000}

	metricLabelNames = []string{
		CodeLabel,
		MethodLabel,
		URLLabel,
		WarmupLabel,
	}

	// For backwards compatibility, the name is kept as `operations_per_second`.
	latencyHV = newHV(
		"vegeta_request_latency",
		"vegeta request latency")
)

func newHV(n, h string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: n, Help: h, Buckets: latencyDistribution},
		metricLabelNames,
	)
}

// PrometheusStatsReporter structure represents a prometheus stats reporter.
type PrometheusStatsReporter struct {
	handler   http.Handler
	latency *prometheus.HistogramVec
	warmup bool
	firstAttempt bool // Only used if warmup is true
}

// NewPrometheusStatsReporter creates a reporter that collects and reports vegita metrics.
func NewPrometheusStatsReporter(warmup bool) (*PrometheusStatsReporter, error) {
	registry := prometheus.NewRegistry()
	for _, hv := range []*prometheus.HistogramVec{latencyHV} {
		if err := registry.Register(hv); err != nil {
			return nil, fmt.Errorf("register metric failed: %w", err)
		}
	}

	return &PrometheusStatsReporter{
		handler:   promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		latency: latencyHV,
		warmup: warmup,
		firstAttempt: true,
	}, nil
}

// Report captures request metrics.
func (r *PrometheusStatsReporter) Report(res *vegeta.Result) {
	var (
		latency float64 = res.Latency.Seconds() * 1000
		code string = strconv.FormatUint(uint64(res.Code), 10)
		method string = res.Method
		url string = res.URL
	)

	r.observer(code, method, url, r.warmup && r.firstAttempt).Observe(latency)
	r.firstAttempt=false
}

// writer is used for test.
func (r *PrometheusStatsReporter) observer(code string, method string, url string, warmup bool) prometheus.Observer {
	if warmup {
		return r.latency.WithLabelValues(code, method, url, "1").(prometheus.Observer)
	}
	return r.latency.WithLabelValues(code, method, url, "0").(prometheus.Observer)
}

// Handler returns an uninstrumented http.Handler used to serve stats registered by this
// PrometheusStatsReporter.
func (r *PrometheusStatsReporter) Handler() http.Handler {
	return r.handler
}
