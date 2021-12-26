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

package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	api "github.com/p2004a/gbcsdpd/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestParsePubsubMessage(t *testing.T) {
	mpub := &api.MeasurementsPublication{
		Measurements: []*api.Measurement{
			{
				SensorMac:      "01:23:45:67:89:01",
				Temperature:    20.0,
				Humidity:       50.0,
				Pressure:       1024.0,
				BatteryVoltage: 3.0,
			},
			{
				SensorMac:      "01:23:45:67:89:02",
				Temperature:    22.0,
				Humidity:       51.0,
				Pressure:       1024.0,
				BatteryVoltage: 3.0,
			},
		},
	}
	mpubSer, err := proto.Marshal(mpub)
	if err != nil {
		t.Fatalf("Failed to marshal proto data: %v", err)
	}
	pubsubmsgJSON, err := json.Marshal(map[string]interface{}{
		"message": map[string]interface{}{
			"attributes": map[string]string{
				"deviceId":               "testing-device",
				"deviceNumId":            "2955148207840174",
				"deviceRegistryId":       "sensors",
				"deviceRegistryLocation": "europe-west1",
				"projectId":              "some-project-123123",
				"subFolder":              "v1",
			},
			"data":         base64.StdEncoding.EncodeToString(mpubSer),
			"messageId":    "1650549360205915",
			"message_id":   "1650549360205915",
			"publishTime":  "2020-10-22T15:07:36.646Z",
			"publish_time": "2020-10-22T15:07:36.646Z",
		},
		"subscription": "projects/some-project-123123/subscriptions/measurements-subscription",
	})
	if err != nil {
		t.Fatalf("Failed to marshal json data: %v", err)
	}
	msg, err := parsePubsubMessage(pubsubmsgJSON)
	if err != nil {
		t.Fatalf("parsePubsubMessage failed: %v", err)
	}
	expectedMsg := &measurementPubSubMessage{
		DeviceID:              "testing-device",
		DeviceRegistryLocaton: "europe-west1",
		ProjectID:             "some-project-123123",
		PublishTime:           time.Date(2020, time.October, 22, 15, 7, 36, 646000000, time.UTC),
		Measurements:          mpub.Measurements,
	}
	if diff := cmp.Diff(msg, expectedMsg, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected difference:\n%v", diff)
	}
}
