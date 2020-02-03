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
  "time"
  "context"
  "fmt"
  "log"
  "strings"
  "io/ioutil"
  "net/http"
  "crypto/tls"
  "crypto/x509"
  "golang.org/x/sync/errgroup"

  promapi "github.com/prometheus/client_golang/api"
  promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
  "k8s.io/apimachinery/pkg/util/wait"

)

type Clients struct {
  promClients map[string]promv1.API
}

type Config struct {
  Name string
  Endpoint string
  ClientCertFile string
  ClientKeyFile string
  CAFile string
}

func NewClients(cfgs ...*Config) (*Clients, error) {
  var (
    err error
    promCfg *promapi.Config
    promapiClient promapi.Client
    clients = &Clients{
      promClients: map[string]promv1.API{},
    }
  )

  for _, c := range cfgs {
    switch {
    case c.ClientKeyFile != "":
      if promCfg, err = CreateClientCertAuthPromConfig(c); err != nil {
        return nil, err
      }
    default:
      promCfg = &promapi.Config{Address: c.Endpoint}
    }

    if promapiClient, err = promapi.NewClient(*promCfg); err != nil {
      return nil, err
    }
    clients.set(c.Name, promv1.NewAPI(promapiClient))
  }
  return clients, nil
}

func CreateClientCertAuthPromConfig(c *Config) (*promapi.Config, error) {
  var (
    caCert []byte
    cliCert tls.Certificate
    err error
  )
  if cliCert, err = tls.LoadX509KeyPair(c.ClientCertFile, c.ClientKeyFile); err != nil {
    return nil, err
  }
  if caCert, err = ioutil.ReadFile(c.CAFile); err != nil {
    return nil, err
  }
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cliCert},
		RootCAs:      caCertPool,
	}
  return &promapi.Config {
    Address: c.Endpoint,
    RoundTripper: &http.Transport{TLSClientConfig: tlsConfig},
  }, nil
}

func (cs *Clients) set(name string, c promv1.API) error {
  if _, ok := cs.promClients[name]; ok {
    return fmt.Errorf("%s has already been set.", name)
  }
  cs.promClients[name] = c
  return nil
}

func (cs *Clients) Get(name string) promv1.API {
  return cs.promClients[name]
}

func (cs *Clients) WaitForReady(testName string) error {
  eg := errgroup.Group{}
  for _, c := range cs.promClients {
    eg.Go(func() error {
        return wait.PollImmediate(time.Second, 60*time.Second, func() (bool, error) {
          var (
            res promv1.TargetsResult
            err error
          )
          if res, err = c.Targets(context.Background()); err != nil {
            log.Print(err)
            switch {
            case strings.Contains(err.Error(), "404"):
              return false, nil
            default:
              return false, fmt.Errorf("Could not connect to Prometheus Server. %v", err)
            }
          }
          return TargetReady(testName, &res)
        })
    })
  }
  if err := eg.Wait(); err != nil {
    return err
  }
  return nil
}

func TargetReady(testName string, targets *promv1.TargetsResult) (bool, error) {
	for _, t := range targets.Active {
    tn, _ := t.Labels["test_name"]
		log.Printf("Expect label: %s, Got: %s.", tn, testName)
		if string(tn) == testName {
			log.Printf("Target is found. Label: %s, HealthStatus: %s", testName, t.Health)
			return t.Health == promv1.HealthGood, nil
		}
	}
	log.Printf("Target is not found in %d Active Target(s).", len(targets.Active))
	return false, nil
}
