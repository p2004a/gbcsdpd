// Copyright 2023 Google LLC
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
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	api "github.com/p2004a/gbcsdpd/api"
	"github.com/p2004a/gbcsdpd/pkg/config"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
)

// CloudPubSubSink publishes measurements to Google Cloud Pub/Sub.
type CloudPubSubSink struct {
	config *config.CloudPubSubSink
	client *pubsub.Client
	topic  *pubsub.Topic
	rl     *rateLimiter
}

// Publish is used to push measurement for publication.
func (s *CloudPubSubSink) Publish(m *api.Measurement) {
	s.rl.Publish(m)
}

func (s *CloudPubSubSink) groupPublish(ms []*api.Measurement) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pub := &api.MeasurementsPublication{Measurements: ms}
	serPub, err := proto.Marshal(pub)
	if err != nil {
		log.Fatalf("Failed to binary encode measurement: %v", err)
	}
	// The attributes are named for compatibility with the IoT Core way of publishing.
	_, err = s.topic.Publish(context.Background(), &pubsub.Message{
		Data: serPub,
		Attributes: map[string]string{
			"deviceId":               s.config.Device,
			"deviceRegistryLocation": "global",
			"projectId":              s.client.Project(),
			"subFolder":              "v1",
		},
	}).Get(ctx)
	if err != nil {
		log.Printf("Failed to publish measurement: %v", err)
	}
}

// NewCloudPubSubSink creates new CloudPubSubSink.
func NewCloudPubSubSink(config *config.CloudPubSubSink) (*CloudPubSubSink, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, config.Project, option.WithCredentials(config.Creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %v", err)
	}
	topic := client.Topic(config.Topic)
	topic.PublishSettings.Timeout = 30 * time.Second
	s := &CloudPubSubSink{
		config: config,
		topic:  topic,
		client: client,
	}
	s.rl = newRateLimiter(config.RateLimit, s.groupPublish)
	return s, nil
}
