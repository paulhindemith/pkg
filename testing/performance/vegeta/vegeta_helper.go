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
	"time"
	"log"
	"net"
	"net/http"
	"strconv"
	"crypto/tls"

	vegeta "github.com/tsenart/vegeta/lib"
)

type Client struct {
	StartedAt time.Time
	EndedAt time.Time
	name string
	period time.Duration
	rate *vegeta.Rate
	targetter vegeta.Targeter
	attacker *vegeta.Attacker
	report func(res *vegeta.Result)
}

type Config struct {
	Name string
  Freqency int
	Per time.Duration
	Period time.Duration
	URL string
	ActAddress string
	ActPort int
	Method string
	Timeout time.Duration
}

var (
	dialContext           = (&net.Dialer{}).DialContext
)

func spoofing(endpoint string, port int) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
			spoofed := endpoint + ":" + strconv.Itoa(port)
			return dialContext(ctx, network, spoofed)
		}
}

func NewClient(cfg *Config) (*Client, error) {
	return &Client {
		name: cfg.Name,
		period: cfg.Period,
		rate: &vegeta.Rate{
			Freq: cfg.Freqency,
			Per: cfg.Per,
		},
		targetter: vegeta.NewStaticTargeter(vegeta.Target{
			Method: cfg.Method,
			URL:    cfg.URL,
		}),
		attacker: vegeta.NewAttacker(
			vegeta.Client(&http.Client {
				Transport: &http.Transport{
					DialContext: spoofing(cfg.ActAddress, cfg.ActPort),
					TLSClientConfig: &tls.Config{ InsecureSkipVerify: true },
				},
			}),
			vegeta.Timeout(cfg.Timeout),
		),
	}, nil
}

func (c *Client) RegisterReporter(f func(res *vegeta.Result)) {
	c.report = f
}

func (c *Client) Finished() bool {
	return !c.EndedAt.IsZero()
}

func (c *Client) Start(finish context.CancelFunc) {
	log.Print("Starting Vegeta atack...")
	c.StartedAt = time.Now()
	results := c.attacker.Attack(c.targetter, c.rate, c.period, c.name)
	for res := range results {
		c.report(res)
	}
	c.EndedAt = time.Now()
	log.Print("Finished Vegeta atack.")
	finish()
}
