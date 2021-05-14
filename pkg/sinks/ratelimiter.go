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
	"math/rand"
	"time"

	api "github.com/p2004a/gbcsdpd/api"
	"github.com/p2004a/gbcsdpd/pkg/config"
)

type groupPublish func([]*api.Measurement)

type rateLimiter struct {
	config       *config.RateLimit
	measurements chan *api.Measurement
	cb           groupPublish
}

func (rl *rateLimiter) Publish(m *api.Measurement) {
	if rl.config != nil {
		rl.measurements <- m
	} else {
		rl.cb([]*api.Measurement{m})
	}
}

func (rl *rateLimiter) nextWaitDuration() time.Duration {
	d := float64(rl.config.Max1In)
	jitter := (rand.Float64()*0.4 - 0.2) * d
	return time.Duration(d + jitter)
}

func (rl *rateLimiter) limiter() {
	deadline := time.After(rl.nextWaitDuration())
	mset := make(map[string]*api.Measurement)
	for {
		select {
		case m := <-rl.measurements:
			mset[m.SensorMac] = m
		case <-deadline:
			if len(mset) > 0 {
				var ms []*api.Measurement
				for _, m := range mset {
					ms = append(ms, m)
				}
				rl.cb(ms)
				mset = make(map[string]*api.Measurement)
			}
			deadline = time.After(rl.nextWaitDuration())
		}
	}
}

// Config can be nil, then the limier will basically copy requests to cb
func newRateLimiter(config *config.RateLimit, cb groupPublish) *rateLimiter {
	rl := &rateLimiter{
		config:       config,
		measurements: make(chan *api.Measurement, 4),
		cb:           cb,
	}
	if rl.config != nil {
		go rl.limiter()
	}
	return rl
}
