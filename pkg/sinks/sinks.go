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

	api "github.com/p2004a/gbcsdpd/api"
	"github.com/p2004a/gbcsdpd/pkg/config"
)

// Sink represents an object with Publish method for publishing api.Measurement.
type Sink interface {
	Publish(*api.Measurement)
}

// NewSink creates a new Sink objects based on the config.Sink configuration.
func NewSink(sinkConfig config.Sink) (Sink, error) {
	switch s := sinkConfig.(type) {
	case *config.GCPSink:
		return NewGCPSink(s)
	case *config.StdoutSink:
		return NewStdoutSink(s)
	case *config.MQTTSink:
		return NewMQTTSink(s)
	default:
		return nil, fmt.Errorf("unknown sink config type: %v", sinkConfig)
	}
}
