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
	"testing"
	"time"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	vegeta "github.com/tsenart/vegeta/lib"
)

const (
	latency = 39
	code = "200"
	method       = "GET"
	url = "https://localhost:8080"
	warmup1       = "1"
	warmup0      = "0"
)

type Bucket struct {
	CumulativeCount uint64
	UpperBound      float64
}

type Label struct {
	Name  string
	Value string
}

func TestReporterReport(t *testing.T) {
	tests := []struct {
		name                string
		latency             float64
		expectedSampleCount uint64
		expectedSampleSum   float64
		expectedBucket      Bucket
		expectedLabel       []*Label
	}{
		{
			name:    "{warmup:true,firstattempt:true}",
			latency: latency,
			expectedLabel: []*Label{
				{
					Name:  CodeLabel,
					Value: code,
				},
				{
					Name:  MethodLabel,
					Value: method,
				},
				{
					Name:  URLLabel,
					Value: url,
				},
				{
					Name:  WarmupLabel,
					Value: warmup1,
				},
			},
			expectedSampleCount: 1,
			expectedSampleSum:   39,
			expectedBucket: Bucket{
				CumulativeCount: 1,
				UpperBound:      40,
			},
		},
		{
			name:    "{warmup:true,firstattempt:false}",
			latency: latency,
			expectedLabel: []*Label{
				{
					Name:  CodeLabel,
					Value: code,
				},
				{
					Name:  MethodLabel,
					Value: method,
				},
				{
					Name:  URLLabel,
					Value: url,
				},
				{
					Name:  WarmupLabel,
					Value: warmup0,
				},
			},
			expectedSampleCount: 1,
			expectedSampleSum:   39,
			expectedBucket: Bucket{
				CumulativeCount: 1,
				UpperBound:      40,
			},
		},
	}

	warmup := true
	reporter, err := NewPrometheusStatsReporter(warmup)
	if err != nil {
		t.Errorf("Something went wrong with creating a reporter, '%v'.", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			codeint, _ := strconv.Atoi(code)
			reporter.Report(&vegeta.Result{
				Code: uint16(codeint),
				Latency: latency*time.Millisecond,
				Method: method,
				URL: url,
			})
			writer := reporter.observer(code, method, url, warmup).(prometheus.Metric)
			checkData(t, writer, test.expectedSampleCount, test.expectedSampleSum, test.expectedBucket, test.expectedLabel)
			warmup = false

		})
	}
}

func checkData(t *testing.T, writer prometheus.Metric, expectedSampleCount uint64, expectedSampleSum float64, expectedBucket Bucket, expectedLabel []*Label) {
	m := dto.Metric{}
	if err := writer.Write(&m); err != nil {
		t.Fatalf("hv.writer().Write() error = %v", err)
	}

	if expectedSampleCount != *m.Histogram.SampleCount {
		t.Errorf("Got %v for SampleCount value, wanted %v", *m.Histogram.SampleCount, expectedSampleCount)
	}
	if expectedSampleSum != *m.Histogram.SampleSum {
		t.Errorf("Got %v for SampleSum value, wanted %v", *m.Histogram.SampleSum, expectedSampleSum)
	}
	found := false
	for _, bt := range m.Histogram.Bucket {
		if *bt.UpperBound == expectedBucket.UpperBound {
			if *bt.CumulativeCount != expectedBucket.CumulativeCount {
				t.Errorf("Got %v for Histogram CumulativeCount value, wanted %v", *bt.CumulativeCount, expectedBucket.CumulativeCount)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("Unknown UpperBound %v", expectedBucket.UpperBound)
	}
LOOP:
	for _, el := range expectedLabel {
		for _, al := range m.Label {
			if *al.Name == el.Name {
				if *al.Value != el.Value {
					t.Errorf("Got %s for Label %s value, wanted %s", *al.Value, el.Name, el.Value)
				}
				continue LOOP
			}
		}
		t.Errorf("Label %s is not found", el.Name)
	}
}
