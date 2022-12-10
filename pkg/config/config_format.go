// Copyright 2021 Google LLC
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

// Represents the config file, root for unmarshaling the TOML config
type fConfig struct {
	// Name of the bluetooth adapter to listen for publications, eg hci0
	Adapter *string `toml:"adapter"` // default: hci0

	// If none sinks are defined, a single default Stdout sink is created
	Sinks fSinks `toml:"sinks"`
}

// Struct holds list of sinks for publications
type fSinks struct {
	MQTT   []*fMQTTSink   `toml:"mqtt"`
	GCP    []*fGCPSink    `toml:"gcp"`
	Stdout []*fStdoutSink `toml:"stdout"`
}

// Configruation for publishing to stdout
type fStdoutSink struct {
	// Optional name of sink
	Name string `toml:"name"`

	RateLimit *fRateLimit `toml:"rate_limit"`
}

// Configuration for publishing to generic MQTT server
type fMQTTSink struct {
	// Optional name of sink
	Name string `toml:"name"`

	RateLimit *fRateLimit `toml:"rate_limit"`

	// MQTT topic name
	Topic string `toml:"topic"`

	// Client ID to send to the server, can be left as an empty string
	ClientID string `toml:"client_id"`

	// Username for authentication
	UserName string `toml:"username"`

	// Password for authentication. Spec allows for binary data here, but this config
	// doesn't as TOML doesn't have a native type for it
	Password string `toml:"password"`

	// Format of published `MeasurementsPublication` message. Can be either BINARY or JSON
	Format *string `toml:"format"` // default: BINARY

	// Server name to connect to
	ServerName string `toml:"server_name"`

	// Server port to connect to
	ServerPort *int `toml:"server_port"` // default: 8883

	// Whatever to use TLS to connect to the server
	EnableTLS *bool `toml:"enable_tls"` // default: true

	// TLS configuration for connection, used when EnableTLS is true.
	TLS fTLSConfig `toml:"tls"`
}

// Configruation for publishing to Google Cloud IoT Core
type fGCPSink struct {
	// Optional name of sink
	Name string `toml:"name"`

	RateLimit *fRateLimit `toml:"rate_limit"`

	// Project Id
	Project string `toml:"project"`

	// Region to contact
	Region string `toml:"region"`

	// Device registry ID
	Registry string `toml:"registry"`

	// Device ID
	Device string `toml:"device"`

	// Path to device private key in PEM format
	Key string `toml:"key"`

	// GCP server name to connect to
	ServerName *string `toml:"server_name"` // default: mqtt.googleapis.com

	// GCP server port to connect to
	ServerPort *int `toml:"server_port"` // default: 8883

	// TLS configuration for connecting to GCP
	TLS fTLSConfig `toml:"tls"`
}

// Configuration for publishing rate limitting
type fRateLimit struct {
	// Specifies rate limit to publish max 1 publication in duration.
	// Duration is string in the format for `time.ParseDuration`, eg: 60s, 1m10s
	Max1In string `toml:"max_1_in"`
}

type fTLSConfig struct {
	// root certificate authorities, if empty, the systems default is used
	CACerts *string `toml:"ca_certs"`

	// Controls if we should verify the server certificate
	SkipVerify bool `toml:"skip_verify"` // default: false

	// Allows to set the server name, by default inherits ServerName from parent
	ServerName *string `toml:"server_name"`
}
