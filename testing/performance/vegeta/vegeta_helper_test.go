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

package vegeta

import (
	"context"
	"fmt"
	"time"
	"strconv"
	"testing"
	"net/url"
	"net/http"
	"net/http/httptest"
	vegeta "github.com/tsenart/vegeta/lib"
)

func TestVegeta(t *testing.T) {
	var (
		u *url.URL
		err error
		c *Client
		port int
		result *vegeta.Result
		msg = "Hello, vegeta"
	)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, msg)
	}))
	defer ts.Close()
	if u, err = url.Parse(ts.URL); err != nil {
		t.Fatal(err)
	}
	port, _ = strconv.Atoi(u.Port())
	if c, err = NewClient(&Config{
		Name: "TestVegeta",
		Freqency: 1,
		Per: 1*time.Millisecond,
		Period: 1*time.Millisecond,
		URL: fmt.Sprintf("https://%s", u.Hostname()),
		ActAddress: u.Hostname(),
		ActPort: port,
		Method: http.MethodGet,
		Timeout: 1*time.Second,
	}); err != nil {
		t.Fatal(err)
	}

	c.RegisterReporter(func(res *vegeta.Result){
		result = res
	})
	ctx, finish := context.WithCancel(context.Background())
	c.Start(finish)
	<-ctx.Done()
	if !c.Finished() {
		t.Fatal("vegeta must finish.")
	}
	gotStatusCode := strconv.FormatUint(uint64(result.Code), 10)
	if gotStatusCode != "200" {
		t.Fatalf("http status is wrong. Expect: %s, Got: %s.", "200", gotStatusCode)
	}
	if string(result.Body) != msg {
		t.Fatalf("Body is wrong. Expect: %s, Got: %s.", msg, string(result.Body))
	}
}
