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
	"sort"

	api "github.com/p2004a/gbcsdpd/api"
	"github.com/p2004a/gbcsdpd/pkg/config"
)

// StdoutSink publishes measurements on standard output.
type StdoutSink struct {
	config *config.StdoutSink
	rl     *rateLimiter
}

// Publish is used to push measurement for publication.
func (s *StdoutSink) Publish(m *api.Measurement) {
	s.rl.Publish(m)
}

type byMac []*api.Measurement

func (a byMac) Len() int           { return len(a) }
func (a byMac) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byMac) Less(i, j int) bool { return a[i].SensorMac < a[j].SensorMac }

func (s *StdoutSink) groupPublish(ms []*api.Measurement) {
	sort.Sort(byMac(ms))
	for _, m := range ms {
		fmt.Printf("[%s] %s = %2.2f°C, %3.2f%%, %4.2fhPa, %1.2fV\n", s.config.Name,
			m.SensorMac, m.Temperature, m.Humidity, m.Pressure, m.BatteryVoltage)
	}
}

// NewStdoutSink creates new StdoutSink.
func NewStdoutSink(config *config.StdoutSink) (*StdoutSink, error) {
	s := &StdoutSink{config: config}
	s.rl = newRateLimiter(config.RateLimit, s.groupPublish)
	return s, nil
}
