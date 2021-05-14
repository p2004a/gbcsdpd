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
	"flag"
	"log"
	"math"

	api "github.com/p2004a/gbcsdpd/api"
	"github.com/p2004a/gbcsdpd/pkg/blelistener"
	"github.com/p2004a/gbcsdpd/pkg/config"
	"github.com/p2004a/gbcsdpd/pkg/ruuviparse"
	sinkspkg "github.com/p2004a/gbcsdpd/pkg/sinks"
)

const (
	ruuviManufacturerID = 0x0499
)

func nilToNaN(value *float32) float32 {
	if value == nil {
		return float32(math.NaN())
	}
	return *value
}

func main() {
	configPath := flag.String("config", "", "Path to the TOML config file")
	logTime := flag.Bool("logtime", true, "If true log messages printed to stderr will contain time and date")
	flag.Parse()

	if *logTime {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(0)
	}

	conf, err := config.Read(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var sinks []sinkspkg.Sink
	for _, sinkConfig := range conf.Sinks {
		sink, err := sinkspkg.NewSink(sinkConfig)
		if err != nil {
			log.Fatalf("Failed to create sink: %v", err)
		}
		sinks = append(sinks, sink)
	}

	advListener, err := blelistener.NewAdvListener(conf.Adapter)
	if err != nil {
		log.Fatalf("Failed to listen for BLE advertisements: %v", err)
	}

	for adv := range advListener.Advertisements() {
		data, ok := adv.ManufacturerData[ruuviManufacturerID]
		if !ok {
			continue
		}
		ruuviData, err := ruuviparse.Parse(data)
		if err != nil {
			log.Printf("Failed to parse ruuvi data: %v", err)
		}
		measuement := &api.Measurement{
			SensorMac:      adv.Address.String(),
			Temperature:    nilToNaN(ruuviData.Temperature),
			Humidity:       nilToNaN(ruuviData.Humidity),
			Pressure:       nilToNaN(ruuviData.Pressure),
			BatteryVoltage: nilToNaN(ruuviData.BatteryVoltage),
		}
		for _, sink := range sinks {
			sink.Publish(measuement)
		}
	}
	if advListener.Err != nil {
		log.Fatalf("BLE Advertisement listener failed: %v", advListener.Err)
	}
}
