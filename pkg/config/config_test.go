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
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func readKey(t *testing.T, keyPath string) *rsa.PrivateKey {
	privateKeyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to load file %s: %v", keyPath, err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		t.Fatalf("Failed to parse pem file %s: %v", keyPath, err)
	}
	return privateKey
}

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
		cmpopts.IgnoreFields(tls.Config{}, "ClientSessionCache"))
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
				TLSConfig:  nil,
			},
			&GCPSink{
				Name:       "gcp sink 1",
				Project:    "project1",
				Region:     "europe-west1",
				Registry:   "registry1",
				Device:     "device1",
				ServerName: "test.gcp.com",
				ServerPort: 9999,
				Key:        readKey(t, "testdata/test1/key.pem"),
				RateLimit:  &RateLimit{Max1In: 60 * time.Second},
				TLSConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					ClientSessionCache: tls.NewLRUClientSessionCache(10),
					InsecureSkipVerify: true,
					ServerName:         "tls_overriden.gcp.com",
					RootCAs:            readCACerts(t, "testdata/test1/myCa.pem"),
				},
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

func TestBaseGCP(t *testing.T) {
	config, err := Read("testdata/gcpbase/config.toml")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}
	expectedConfig := &Config{
		Adapter: "hci0",
		Sinks: []Sink{
			&GCPSink{
				Name:       "unnamed-gcp-sink-0",
				Project:    "project2",
				Region:     "asia-east1",
				Registry:   "registry2",
				Device:     "device2",
				ServerName: "mqtt.googleapis.com",
				ServerPort: 8883,
				Key:        readKey(t, "testdata/gcpbase/key.pem"),
				TLSConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					ClientSessionCache: tls.NewLRUClientSessionCache(10),
					InsecureSkipVerify: false,
					ServerName:         "mqtt.googleapis.com",
				},
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
