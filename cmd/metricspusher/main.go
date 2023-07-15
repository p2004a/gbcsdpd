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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	gbcsdpdapipb "github.com/p2004a/gbcsdpd/api"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// https://cloud.google.com/pubsub/docs/reference/rpc/google.pubsub.v1#pubsubmessage
type inputPubSubMessage struct {
	Message struct {
		Data        []byte            `json:"data,omitempty"`
		Attributes  map[string]string `json:"attributes"`
		MessageID   string            `json:"message_id"`
		PublishTime time.Time         `json:"publish_time"`
		OrderingKey string            `json:"ordering_key"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type measurementPubSubMessage struct {
	DeviceID, DeviceRegistryLocaton, ProjectID string
	PublishTime                                time.Time
	Measurements                               []*gbcsdpdapipb.Measurement
}

func parsePubsubMessage(data []byte) (*measurementPubSubMessage, error) {
	var msg inputPubSubMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %v", err)
	}
	deviceID := msg.Message.Attributes["deviceId"]
	deviceRegistryLocation := msg.Message.Attributes["deviceRegistryLocation"]
	projectID := msg.Message.Attributes["projectId"]
	subFolder := msg.Message.Attributes["subFolder"]
	if deviceID == "" || deviceRegistryLocation == "" || projectID == "" {
		return nil, fmt.Errorf("One of the required Attributes was not present: %v", msg.Message.Attributes)
	}
	if subFolder != "v1" {
		return nil, fmt.Errorf("Only the v1 version of measurements is supported, got: %s", subFolder)
	}
	measuementPub := &gbcsdpdapipb.MeasurementsPublication{}
	if err := proto.Unmarshal(msg.Message.Data, measuementPub); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal MeasurementsPublication from pubsub data: %v", err)
	}
	return &measurementPubSubMessage{
		DeviceID:              deviceID,
		DeviceRegistryLocaton: deviceRegistryLocation,
		ProjectID:             projectID,
		PublishTime:           msg.Message.PublishTime,
		Measurements:          measuementPub.Measurements,
	}, nil
}

func appendMeasurementTimeSeries(
	ts []*monitoringpb.TimeSeries,
	resource *monitoredrespb.MonitoredResource,
	dimension string, t time.Time, value float32,
) []*monitoringpb.TimeSeries {
	if math.IsNaN(float64(value)) {
		return ts
	}
	return append(ts, &monitoringpb.TimeSeries{
		Metric: &metricpb.Metric{
			Type: "custom.googleapis.com/sensor/measurement/" + dimension,
		},
		Resource: resource,
		Points: []*monitoringpb.Point{
			{
				Interval: &monitoringpb.TimeInterval{
					EndTime: timestamppb.New(t),
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DoubleValue{
						DoubleValue: float64(value),
					},
				},
			},
		},
	})
}

func main() {
	http.HandleFunc("/", handlePubSub)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handlePubSub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Got %s, not POST request for URL: %v", r.Method, r.URL)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Body ioutil.ReadAll: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	psmsg, err := parsePubsubMessage(body)
	if err != nil {
		log.Printf("Failed to parse pubsub message: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Creates a client.
	client, err := monitoring.NewMetricClient(r.Context())
	if err != nil {
		log.Printf("Failed to create monitoring client: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	var ts []*monitoringpb.TimeSeries
	for _, m := range psmsg.Measurements {
		// https://cloud.google.com/monitoring/api/resources#tag_generic_node
		res := &monitoredrespb.MonitoredResource{
			Type: "generic_node",
			Labels: map[string]string{
				"project_id": psmsg.ProjectID,
				"location":   psmsg.DeviceRegistryLocaton,
				"namespace":  psmsg.DeviceID,
				"node_id":    m.SensorMac,
			},
		}
		ts = appendMeasurementTimeSeries(ts, res, "temperature", psmsg.PublishTime, m.Temperature)
		ts = appendMeasurementTimeSeries(ts, res, "humidity", psmsg.PublishTime, m.Humidity)
		ts = appendMeasurementTimeSeries(ts, res, "pressure", psmsg.PublishTime, m.Pressure)
		ts = appendMeasurementTimeSeries(ts, res, "battery", psmsg.PublishTime, m.BatteryVoltage)
	}

	// Writes time series data.
	if err := client.CreateTimeSeries(r.Context(), &monitoringpb.CreateTimeSeriesRequest{
		Name:       monitoring.MetricProjectPath(psmsg.ProjectID),
		TimeSeries: ts,
	}); err != nil {
		// If we receive INVALID_ARGUMENT, it likely means that we pushed timeseries out-of-order
		// because pubsub delivered them to us out-of-order. Let's ACK message with 202 in this case
		// to prevent from retrying incorrect request.
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				log.Printf("Got INVALID_ARGUMENT from CreateTimeSeries, assuming got data from pubsub out-of-order, full error: %v", err)
				w.WriteHeader(http.StatusAccepted)
				return
			}
		}
		log.Printf("Failed to write time series data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
