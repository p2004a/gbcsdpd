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
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/oauth2/google"
)

// Sink represents the configuration for Sink.
type Sink interface{}

// Config contains the full parsed configuration for the application.
type Config struct {
	Adapter string
	Sinks   []Sink
}

// RateLimit is configruation for the rate limiting of sinks.
type RateLimit struct {
	Max1In time.Duration
}

// PublicationFormat represents format of message published on MQTT topic.
type PublicationFormat int

const (
	// BINARY means data is a proto serialized to binary.
	BINARY PublicationFormat = iota

	// JSON means data is in json format.
	JSON
)

// MQTTSink is configuration for the sink.MQTTSink.
type MQTTSink struct {
	Name, Topic, ClientID, UserName, Password string
	Format                                    PublicationFormat
	RateLimit                                 *RateLimit
	ServerName                                string
	ServerPort                                int
	TLSConfig                                 *tls.Config
}

// GCPSink is configuration for the sink.GCPSink.
type GCPSink struct {
	Name, Project, Region, Registry, Device string
	Key                                     *rsa.PrivateKey
	RateLimit                               *RateLimit
	ServerName                              string
	ServerPort                              int
	TLSConfig                               *tls.Config
}

type CloudPubSubSink struct {
	Name, Project, Topic, Device string
	RateLimit                    *RateLimit
	Creds                        *google.Credentials
}

// StdoutSink is configuration for sink.StdoutSink.
type StdoutSink struct {
	Name      string
	RateLimit *RateLimit
}

var (
	projectIDRE, registryDeviceIDsRE, cloudPubSubTopicRE, cloudIoTRegionRE, clientIDRE *regexp.Regexp
)

func init() {
	projectIDRE = regexp.MustCompile(`[-a-z0-9]{6,30}`)
	registryDeviceIDsRE = regexp.MustCompile(`[a-zA-Z][-a-zA-Z0-9._+~%]{2,254}`)
	cloudPubSubTopicRE = regexp.MustCompile(`[a-zA-Z][-a-zA-Z0-9._+~%]{2,254}`)
	cloudIoTRegionRE = regexp.MustCompile(`us-central1|europe-west1|asia-east1`)
	clientIDRE = regexp.MustCompile(`[0-9a-zA-Z]{0,23}`)
}

func joinPathWithAbs(basePath, filePath string) string {
	if path.IsAbs(filePath) {
		return filePath
	}
	return path.Join(basePath, filePath)
}

func parseRateLimit(rateLimit *fRateLimit) (*RateLimit, error) {
	if rateLimit == nil {
		return nil, nil
	}
	res := &RateLimit{}
	var err error
	res.Max1In, err = time.ParseDuration(rateLimit.Max1In)
	if err != nil {
		return nil, fmt.Errorf("failed to parse max_1_in as duration: %v", err)
	}
	if res.Max1In < time.Second {
		return nil, fmt.Errorf("max_1_in must be more then 1s")
	}
	return res, nil
}

func parseTLSConfig(config *fTLSConfig, defaultServerName string, basePath string) (*tls.Config, error) {
	res := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ClientSessionCache: tls.NewLRUClientSessionCache(10),
		InsecureSkipVerify: config.SkipVerify,
	}
	if config.CACerts != nil {
		certpool := x509.NewCertPool()
		pemCerts, err := ioutil.ReadFile(joinPathWithAbs(basePath, *config.CACerts))
		if err != nil {
			return nil, fmt.Errorf("failed to read '%s' cacerts file: %v", *config.CACerts, err)
		}
		if !certpool.AppendCertsFromPEM(pemCerts) {
			return nil, fmt.Errorf("there weren't any falid certs to add in the given '%s' file", *config.CACerts)
		}
		res.RootCAs = certpool
	}
	if config.ServerName != nil {
		res.ServerName = *config.ServerName
	} else {
		res.ServerName = defaultServerName
	}
	return res, nil
}

func parseMQTTSink(basePath string, sinkID int, sink *fMQTTSink) (*MQTTSink, error) {
	res := &MQTTSink{}
	if sink.Name == "" {
		res.Name = fmt.Sprintf("unnamed-mqtt-sink-%d", sinkID)
	} else {
		res.Name = sink.Name
	}

	if len(sink.Topic) < 1 || len(sink.Topic) > 65535 || sink.Topic[0] == '$' || strings.ContainsAny(sink.Topic, "+#\u0000") {
		return nil, fmt.Errorf("sink %s: Topic is not in valid format, see https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html#_Toc398718106, given: '%s'", sink.Name, sink.Topic)
	}
	res.Topic = sink.Topic

	if !clientIDRE.MatchString(sink.ClientID) {
		return nil, fmt.Errorf("sink %s: Client ID is not in valid format, see https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html#_Toc398718031, given: '%s'", res.Name, sink.ClientID)
	}
	res.ClientID = sink.ClientID

	if len(sink.UserName) > 65535 || len(sink.Password) > 65535 {
		return nil, fmt.Errorf("sink %s: Max len of UserName and Password fields is 65535", res.Name)
	}
	res.UserName = sink.UserName
	res.Password = sink.Password

	if sink.Format == nil || *sink.Format == "BINARY" {
		res.Format = BINARY
	} else if *sink.Format == "JSON" {
		res.Format = JSON
	} else {
		return nil, fmt.Errorf("sink %s: Format have to be either BINARY or JSON, given: '%s'", res.Name, *sink.Format)
	}

	rateLimit, err := parseRateLimit(sink.RateLimit)
	if err != nil {
		return nil, fmt.Errorf("sink %s: Failed to parse rate limit: %v", sink.Name, err)
	}
	res.RateLimit = rateLimit

	if len(sink.ServerName) == 0 {
		return nil, fmt.Errorf("sink %s: server_name is a required field", sink.Name)
	}
	res.ServerName = sink.ServerName
	if sink.ServerPort == nil {
		res.ServerPort = 8883
	} else {
		res.ServerPort = *sink.ServerPort
	}

	if sink.EnableTLS == nil || *sink.EnableTLS {
		tlsConfig, err := parseTLSConfig(&sink.TLS, res.ServerName, basePath)
		if err != nil {
			return nil, fmt.Errorf("sink %s: Failed to parse tls config: %v", sink.Name, err)
		}
		res.TLSConfig = tlsConfig
	}

	return res, nil
}

func parseGCPSink(basePath string, sinkID int, sink *fGCPSink) (*GCPSink, error) {
	res := &GCPSink{}
	if sink.Name == "" {
		res.Name = fmt.Sprintf("unnamed-gcp-sink-%d", sinkID)
	} else {
		res.Name = sink.Name
	}

	if !projectIDRE.MatchString(sink.Project) {
		return nil, fmt.Errorf("sink %s: Project ID must meet requirements in https://cloud.google.com/resource-manager/docs/creating-managing-projects#before_you_begin, given: '%s'", res.Name, sink.Project)
	}
	res.Project = sink.Project

	if !cloudIoTRegionRE.MatchString(sink.Region) {
		return nil, fmt.Errorf("sink %s: Region must be one of us-central1, europe-west1, and asia-east1. See https://cloud.google.com/iot/docs/requirements#permitted_characters_and_size_requirements, given: '%s'", res.Name, sink.Region)
	}
	res.Region = sink.Region

	if !registryDeviceIDsRE.MatchString(sink.Registry) {
		return nil, fmt.Errorf("sink %s: Registry ID much meet requirements in https://cloud.google.com/iot/docs/requirements#permitted_characters_and_size_requirements, given: '%s'", res.Name, sink.Registry)
	}
	res.Registry = sink.Registry

	if !registryDeviceIDsRE.MatchString(sink.Device) {
		return nil, fmt.Errorf("sink %s: Device ID much meet requirements in https://cloud.google.com/iot/docs/requirements#permitted_characters_and_size_requirements, given: '%s'", res.Name, sink.Device)
	}
	res.Device = sink.Device

	if sink.Key == "" {
		return nil, fmt.Errorf("sink %s: The private key path must be specified", res.Name)
	} else if privateKeyBytes, err := ioutil.ReadFile(joinPathWithAbs(basePath, sink.Key)); err != nil {
		return nil, fmt.Errorf("sink %s: Failed to read '%s' key file: %v", sink.Name, sink.Key, err)
	} else if privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes); err != nil {
		return nil, fmt.Errorf("sink %s: Failed to parse '%s' PEM key file: %v", sink.Name, sink.Key, err)
	} else {
		res.Key = privateKey
	}

	rateLimit, err := parseRateLimit(sink.RateLimit)
	if err != nil {
		return nil, fmt.Errorf("sink %s: Failed to parse rate limit: %v", sink.Name, err)
	}
	res.RateLimit = rateLimit

	if sink.ServerName == nil {
		res.ServerName = "mqtt.googleapis.com"
	} else {
		res.ServerName = *sink.ServerName
	}
	if sink.ServerPort == nil {
		res.ServerPort = 8883
	} else {
		res.ServerPort = *sink.ServerPort
	}

	tlsConfig, err := parseTLSConfig(&sink.TLS, res.ServerName, basePath)
	if err != nil {
		return nil, fmt.Errorf("sink %s: Failed to parse tls config: %v", sink.Name, err)
	}
	res.TLSConfig = tlsConfig

	return res, nil
}

func parseCloudPubSubSink(basePath string, sinkID int, sink *fCloudPubSubSink) (*CloudPubSubSink, error) {
	ctx := context.Background()
	res := &CloudPubSubSink{}
	if sink.Name == "" {
		res.Name = fmt.Sprintf("unnamed-cloud_pubsub-sink-%d", sinkID)
	} else {
		res.Name = sink.Name
	}

	if !registryDeviceIDsRE.MatchString(sink.Device) {
		return nil, fmt.Errorf("sink %s: Device must match %s, given: '%s'", registryDeviceIDsRE.String(), res.Name, sink.Device)
	}
	res.Device = sink.Device

	if !cloudPubSubTopicRE.MatchString(sink.Topic) {
		return nil, fmt.Errorf("sink %s: Topic must meet requirements in https://cloud.google.com/pubsub/docs/create-topic#resource_names, given: '%s'", res.Name, sink.Topic)
	}
	res.Topic = sink.Topic

	if sink.Creds == nil {
		creds, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("sink %s: No credentials specified, and failed to find default credentials: %v", sink.Name, err)
		}
		res.Creds = creds
	} else if credsBytes, err := ioutil.ReadFile(joinPathWithAbs(basePath, *sink.Creds)); err != nil {
		return nil, fmt.Errorf("sink %s: Failed to read '%s' creds file: %v", sink.Name, *sink.Creds, err)
	} else if creds, err := google.CredentialsFromJSON(ctx, credsBytes, "https://www.googleapis.com/auth/pubsub"); err != nil {
		return nil, fmt.Errorf("sink %s: Failed to parse '%s' PEM key file: %v", sink.Name, *sink.Creds, err)
	} else {
		res.Creds = creds
	}

	if sink.Project == nil {
		res.Project = pubsub.DetectProjectID
	} else if !projectIDRE.MatchString(*sink.Project) {
		return nil, fmt.Errorf("sink %s: Project ID must meet requirements in https://cloud.google.com/resource-manager/docs/creating-managing-projects#before_you_begin, given: '%s'", res.Name, *sink.Project)
	} else {
		res.Project = *sink.Project
	}

	rateLimit, err := parseRateLimit(sink.RateLimit)
	if err != nil {
		return nil, fmt.Errorf("sink %s: Failed to parse rate limit: %v", sink.Name, err)
	}
	res.RateLimit = rateLimit

	return res, nil
}

func parseStdoutSink(sinkID int, sink *fStdoutSink) (*StdoutSink, error) {
	res := &StdoutSink{}
	if sink.Name == "" {
		res.Name = fmt.Sprintf("unnamed-stdout-sink-%d", sinkID)
	} else {
		res.Name = sink.Name
	}
	rateLimit, err := parseRateLimit(sink.RateLimit)
	if err != nil {
		return nil, fmt.Errorf("sink %s: Failed to parse rate limit: %v", sink.Name, err)
	}
	res.RateLimit = rateLimit
	return res, nil
}

// Read reads a configuration file defined in config_format.go and
// parses it into easily digestable Config struct.
func Read(configPath string) (*Config, error) {
	fconfig := fConfig{}
	// If config path is empty, assume empty config file.
	if configPath != "" {
		configBytes, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read '%s' file: %v", configPath, err)
		}
		if err := toml.Unmarshal(configBytes, &fconfig); err != nil {
			return nil, fmt.Errorf("failed to parse '%s' file: %v", configPath, err)
		}
	} else {
		if toml.Unmarshal([]byte{}, &fconfig) != nil {
			panic("failed to parse empty config")
		}
	}

	config := &Config{}
	if fconfig.Adapter == nil {
		config.Adapter = "hci0"
	} else {
		config.Adapter = *fconfig.Adapter
	}
	for i, sink := range fconfig.Sinks.MQTT {
		mqttSink, err := parseMQTTSink(path.Dir(configPath), i, sink)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MQTT sink config: %v", err)
		}
		config.Sinks = append(config.Sinks, mqttSink)
	}
	for i, sink := range fconfig.Sinks.GCP {
		gcpSink, err := parseGCPSink(path.Dir(configPath), i, sink)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GCP sink config: %v", err)
		}
		config.Sinks = append(config.Sinks, gcpSink)
	}
	for i, sink := range fconfig.Sinks.CloudPubSub {
		cloudPubSubSink, err := parseCloudPubSubSink(path.Dir(configPath), i, sink)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Cloud Pub/Sub sink config: %v", err)
		}
		config.Sinks = append(config.Sinks, cloudPubSubSink)
	}
	for i, sink := range fconfig.Sinks.Stdout {
		stdoutSink, err := parseStdoutSink(i, sink)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stdout sink config: %v", err)
		}
		config.Sinks = append(config.Sinks, stdoutSink)
	}

	if len(config.Sinks) == 0 {
		config.Sinks = append(config.Sinks, &StdoutSink{
			Name: "default-sink",
		})
	}

	return config, nil
}
