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

package sinks

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	api "github.com/p2004a/gbcsdpd/api"
	"github.com/p2004a/gbcsdpd/pkg/backoff"
	"github.com/p2004a/gbcsdpd/pkg/config"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func connectMQTTClientWithBackoff(client MQTT.Client) {
	for retryNum := 0; !client.IsConnected(); retryNum++ {
		time.Sleep(backoff.Exponential(retryNum, time.Second, time.Minute*5, 2.0))
		token := client.Connect()
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to connect: %v, retrying...", token.Error())
		}
	}
}

func createMQTTClient(clientID, server string, port int, tls *tls.Config, creds MQTT.CredentialsProvider) (MQTT.Client, error) {
	opts := MQTT.NewClientOptions()
	opts.SetClientID(clientID)
	opts.SetKeepAlive(time.Minute)
	opts.SetProtocolVersion(4) // MQTT 3.1.1
	opts.SetCleanSession(true)
	var brokerAddr string
	if tls != nil {
		opts.SetTLSConfig(tls)
		brokerAddr = fmt.Sprintf("tls://%s:%d", server, port)
	} else {
		brokerAddr = fmt.Sprintf("tcp://%s:%d", server, port)
	}
	opts.AddBroker(brokerAddr)
	opts.SetCredentialsProvider(creds)
	opts.SetConnectionLostHandler(func(client MQTT.Client, err error) {
		log.Printf("Disconnected %s (%v), reconnecting...", brokerAddr, err)
		connectMQTTClientWithBackoff(client)
	})

	client := MQTT.NewClient(opts)
	connectMQTTClientWithBackoff(client)
	return client, nil
}

// MQTTSink publishes measurements to MQTT.
type MQTTSink struct {
	mqttClient MQTT.Client
	rl         *rateLimiter
	topic      string
	format     config.PublicationFormat
}

// Publish is used to push measurement for publication.
func (s *MQTTSink) Publish(m *api.Measurement) {
	s.rl.Publish(m)
}

func (s *MQTTSink) groupPublish(ms []*api.Measurement) {
	pub := &api.MeasurementsPublication{Measurements: ms}
	if s.format == config.BINARY {
		serPub, err := proto.Marshal(pub)
		if err != nil {
			log.Fatalf("Failed to binary encode measurement: %v", err)
		}
		s.mqttClient.Publish(s.topic, 0, false, serPub)
	} else if s.format == config.JSON {
		jsonPub, err := protojson.Marshal(pub)
		if err != nil {
			log.Fatalf("Failed to json encode measurement: %v", err)
		}
		s.mqttClient.Publish(s.topic, 0, false, jsonPub)
	} else {
		log.Fatalf("Unknown data publication format: %v", s.format)
	}
}

// NewMQTTSink creates new MQTTSink.
func NewMQTTSink(c *config.MQTTSink) (*MQTTSink, error) {
	creds := func() (string, string) {
		return c.UserName, c.Password
	}
	mqttClient, err := createMQTTClient(c.ClientID, c.ServerName, c.ServerPort, c.TLSConfig, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %v", err)
	}
	s := &MQTTSink{
		mqttClient: mqttClient,
		topic:      c.Topic,
		format:     c.Format,
	}
	s.rl = newRateLimiter(c.RateLimit, s.groupPublish)
	return s, nil
}
