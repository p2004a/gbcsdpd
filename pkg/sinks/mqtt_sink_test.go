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
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/fhmq/hmq/broker"
	"github.com/google/go-cmp/cmp"
	api "github.com/p2004a/gbcsdpd/api"
	"github.com/p2004a/gbcsdpd/pkg/config"
)

const testerClientID = "testerclient"

func pickFreePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	err = listener.Close()
	if err != nil {
		panic(err)
	}
	return port
}

// Implements fhmq/hmq/plugins/auth Auth interface
type singleUserAuth struct {
	UserName, Password, ClientID string
}

func (a *singleUserAuth) CheckConnect(clientID, username, password string) bool {
	return (a.ClientID == clientID && a.UserName == username && a.Password == password) || clientID == testerClientID
}

func (a *singleUserAuth) CheckACL(action, clientID, username, ip, topic string) bool {
	return true
}

type msg struct {
	Topic   string
	Payload []byte
}

func subscribeToAllTopics(t *testing.T, address string) <-chan msg {
	opts := MQTT.NewClientOptions()
	opts.SetClientID(testerClientID)
	opts.SetProtocolVersion(4) // MQTT 3.1.1
	opts.AddBroker(address)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(time.Millisecond * 5)
	opts.SetCleanSession(false)
	opts.SetResumeSubs(true)

	client := MQTT.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("failed to connect to hmq broker: %v", token.Error())
	}

	sub := make(chan msg, 20)
	messageHandler := func(c MQTT.Client, m MQTT.Message) {
		sub <- msg{Topic: m.Topic(), Payload: m.Payload()}
		m.Ack()
	}
	if token := client.Subscribe("#", 1, messageHandler); token.Wait() && token.Error() != nil {
		t.Fatalf("failed to subscribe to hmq broker: %v", token.Error())
	}
	return sub
}

func TestBorkerIntegration(t *testing.T) {
	testClientID := "pusher"
	testUserName := "bob"
	testPassword := "ilovealice"
	sensorMac := "01:23:45:67:89:AB"
	measuementsTopic := "/measurements"

	// Create and start MQTT broker.
	port := pickFreePort()
	b, err := broker.NewBroker(&broker.Config{
		Worker: 1,
		Host:   "127.0.0.1",
		Port:   fmt.Sprintf("%d", port),
		Plugin: broker.Plugins{
			Auth: &singleUserAuth{
				UserName: testUserName,
				Password: testPassword,
				ClientID: testClientID,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create broker: %v", err)
	}
	b.Start()

	brokerAddr := fmt.Sprintf("tcp://127.0.0.1:%d", port)

	sink, err := NewMQTTSink(&config.MQTTSink{
		Name:       "sink",
		Topic:      measuementsTopic,
		ClientID:   testClientID,
		UserName:   testUserName,
		Password:   testPassword,
		Format:     config.JSON,
		ServerName: "127.0.0.1",
		ServerPort: port,
	})
	if err != nil {
		t.Fatalf("Failed to create mqtt sink: %v", err)
	}

	go func() {
		for {
			time.Sleep(100 * time.Microsecond)
			sink.Publish(&api.Measurement{
				SensorMac:   sensorMac,
				Temperature: 10.0,
				Humidity:    60.0,
			})
		}
	}()

	count := 0
	for msg := range subscribeToAllTopics(t, brokerAddr) {
		// Skip all the broker system messages.
		if strings.HasPrefix(msg.Topic, "$") {
			continue
		}
		count++
		if count == 10 {
			break
		}
		if msg.Topic != measuementsTopic {
			t.Errorf("Received message on wrong topic. got: %s expected: %s", msg.Topic, measuementsTopic)
		}

		type Measurement struct {
			Mac   string  `json:"sensorMac"`
			Temp  float32 `json:"temperature"`
			Humid float32 `json:"humidity"`
		}
		type Payload struct {
			Measurements []Measurement `json:"measurements"`
		}
		var payload Payload

		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Errorf("failed to unmarshal json message: %v", err)
			continue
		}
		if diff := cmp.Diff(payload, Payload{
			Measurements: []Measurement{{
				Mac:   sensorMac,
				Temp:  10.0,
				Humid: 60.0,
			}},
		}); diff != "" {
			t.Errorf("unexpected difference in received message:\n%v", diff)
		}
	}
}
