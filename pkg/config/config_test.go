// Copyright 2021-2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2/google"
)

func readCACerts(t *testing.T, caCertsPath string) *x509.CertPool {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(caCertsPath)
	if err != nil {
		t.Fatalf("Failed to load file %s: %v", caCertsPath, err)
	}
	if !certpool.AppendCertsFromPEM(pemCerts) {
		t.Fatalf("There were no certs in %s", caCertsPath)
	}
	return certpool
}

func readGoogleCredentials(t *testing.T, credsPath string) *google.Credentials {
	credsBytes, err := ioutil.ReadFile(credsPath)
	if err != nil {
		t.Fatalf("Failed to load file %s: %v", credsPath, err)
	}
	creds, err := google.CredentialsFromJSON(context.Background(), credsBytes)
	if err != nil {
		t.Fatalf("Failed to parse creds file %s: %v", credsPath, err)
	}
	return creds
}

func caCertsTrans(cp *x509.CertPool) [][]byte {
	if cp == nil {
		return [][]byte{}
	}
	return cp.Subjects()
}

func cmpConfig(actual, expected *Config) string {
	return cmp.Diff(actual, expected,
		cmp.Transformer("CertPool", caCertsTrans),
		cmpopts.IgnoreUnexported(tls.Config{}),
		cmpopts.IgnoreUnexported(google.Credentials{}),
		cmpopts.IgnoreFields(tls.Config{}, "ClientSessionCache"),
		cmpopts.IgnoreFields(google.Credentials{}, "TokenSource"),
	)
}

func TestParsingCorrect(t *testing.T) {
	config, err := Read("testdata/test1/config.toml")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}
	expectedConfig := &Config{
		Adapter: "hci1",
		Sinks: []Sink{
			&MQTTSink{
				Name:       "mqtt sink 1",
				RateLimit:  &RateLimit{Max1In: 5 * time.Second},
				Topic:      "/measurements",
				ClientID:   "my-pusher",
				UserName:   "alibaba",
				Password:   "open sesame",
				Format:     JSON,
				ServerName: "localhost",
				ServerPort: 8883,
				TLSConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					ClientSessionCache: tls.NewLRUClientSessionCache(10),
					InsecureSkipVerify: true,
					ServerName:         "tls_overriden.gcp.com",
					RootCAs:            readCACerts(t, "testdata/test1/myCa.pem"),
				},
			},
			&CloudPubSubSink{
				Name:      "cloud pubsub sink 1",
				Device:    "device2",
				Project:   "project2",
				Topic:     "topic1",
				Creds:     readGoogleCredentials(t, "testdata/test1/creds.json"),
				RateLimit: &RateLimit{Max1In: 120 * time.Second},
			},
			&StdoutSink{
				Name:      "stdout sink 1",
				RateLimit: &RateLimit{Max1In: 90 * time.Second},
			},
			&StdoutSink{
				Name:      "stdout sink 2",
				RateLimit: &RateLimit{Max1In: 10 * time.Second},
			},
		},
	}
	if diff := cmpConfig(config, expectedConfig); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}

func TestParsingEmpty(t *testing.T) {
	config, err := Read("")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}
	expectedConfig := &Config{
		Adapter: "hci0",
		Sinks: []Sink{
			&StdoutSink{
				Name: "default-sink",
			},
		},
	}
	if diff := cmpConfig(config, expectedConfig); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}
