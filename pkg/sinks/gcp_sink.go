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
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/p2004a/gbcsdpd/pkg/config"
)

// NewGCPSink creates new GCPSink.
func NewGCPSink(c *config.GCPSink) (*MQTTSink, error) {
	clientID := fmt.Sprintf("projects/%s/locations/%s/registries/%s/devices/%s", c.Project, c.Region, c.Registry, c.Device)
	creds := func() (username string, password string) {
		username = "unused"

		claims := &jwt.StandardClaims{
			Audience:  c.Project,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(20 * time.Minute).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		password, err := token.SignedString(c.Key)
		if err != nil {
			log.Fatalf("Failed to sign token: %v", err)
		}
		return
	}
	mqttClient, err := createMQTTClient(clientID, c.ServerName, c.ServerPort, c.TLSConfig, creds)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %v", err)
	}
	s := &MQTTSink{
		mqttClient: mqttClient,
		topic:      fmt.Sprintf("/devices/%s/events/v1", c.Device),
		format:     config.BINARY,
	}
	s.rl = newRateLimiter(c.RateLimit, s.groupPublish)
	return s, nil
}
