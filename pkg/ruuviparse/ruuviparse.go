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

package ruuviparse

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

// RuuviDataFormat represents version of Ruuvi data format.
type RuuviDataFormat uint8

// List of allowed RuuviDataFormat values.
const (
	UNSPECIFIED RuuviDataFormat = iota
	RAWv1
	RAWv2
)

// RuuviData contains a parsed data from Ruuvi sensor.
type RuuviData struct {
	DataFormat        RuuviDataFormat
	Temperature       *float32         // C
	Humidity          *float32         // RH %
	Pressure          *float32         // hPa
	Acceleration      [3]*float32      // G
	BatteryVoltage    *float32         // V
	TxPower           *float32         // dBm
	MovementCounter   *uint            // counter
	MeasurementSeqNum *uint            // counter
	Mac               net.HardwareAddr // MAC
}

// Parse takes Manufacturer Specific Data field from BLE advertisement for the Ruuvi
// manufacturer and parses the content into RuuviData.
func Parse(data []byte) (rd *RuuviData, err error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("got empty byte slice")
	}
	switch data[0] {
	case 3:
		rd, err = parseRAWv1(data)
		if err != nil {
			err = fmt.Errorf("failed to parse data in RAWv1 format: %v", err)
		}
		return parseRAWv1(data)
	case 5:
		rd, err = parseRAWv2(data)
		if err != nil {
			err = fmt.Errorf("failed to parse data in RAWv2 format: %v", err)
		}
	default:
		err = fmt.Errorf("only RAWv1 and RAWv2 Ruuvi formats supported, got: %d", data[0])
	}
	return
}

func nilF32(v float32) *float32 { return &v }
func nilUint(v uint) *uint      { return &v }

func parseRAWv1(data []byte) (*RuuviData, error) {
	rd := struct {
		DataFormat          uint8
		Humidity            uint8
		Temperature         uint8
		TemperatureFraction uint8
		Pressure            uint16
		AccelX              int16
		AccelY              int16
		AccelZ              int16
		BatteryVoltage      uint16
	}{}
	dataExpectedSize := binary.Size(rd)
	if len(data) != dataExpectedSize {
		return nil, fmt.Errorf("Ruuvi manufacturer data must be exactly %d bytes", dataExpectedSize)
	}

	buffer := bytes.NewBuffer(data)
	err := binary.Read(buffer, binary.BigEndian, &rd)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal struct: %v", err)
	}

	var temp *float32 = nil
	if rd.TemperatureFraction < 100 {
		temp = nilF32(float32(rd.Temperature&0x7F) + float32(rd.TemperatureFraction)/100.0)
		if rd.Temperature&0x80 != 0 {
			*temp *= -1
		}
	}

	return &RuuviData{
		DataFormat:  RAWv1,
		Temperature: temp,
		Humidity:    nilF32(float32(rd.Humidity) / 2.0),
		Pressure:    nilF32((float32(rd.Pressure) + 50000.0) / 100.0),
		Acceleration: [3]*float32{
			nilF32(float32(rd.AccelX) / 1000.0),
			nilF32(float32(rd.AccelY) / 1000.0),
			nilF32(float32(rd.AccelZ) / 1000.0),
		},
		BatteryVoltage: nilF32(float32(rd.BatteryVoltage) / 1000.0),
	}, nil
}

func nilInvF32(invalid int, current int, v float32) *float32 {
	if current == invalid {
		return nil
	}
	return &v
}

func nilInvUint(invalid int, current int, v uint) *uint {
	if current == invalid {
		return nil
	}
	return &v
}

// https://github.com/ruuvi/ruuvi-sensor-protocols/blob/master/dataformat_05.md
func parseRAWv2(data []byte) (*RuuviData, error) {
	rd := struct {
		DataFormat        uint8
		Temperature       int16
		Humidity          uint16
		Pressure          uint16
		AccelX            int16
		AccelY            int16
		AccelZ            int16
		PowerInfo         uint16
		MovementCounter   uint8
		MeasurementSeqNum uint16
		Mac               [6]byte
	}{}
	dataExpectedSize := binary.Size(rd)
	if len(data) != dataExpectedSize {
		return nil, fmt.Errorf("Ruuvi manufacturer data must be exactly %d bytes", dataExpectedSize)
	}

	buffer := bytes.NewBuffer(data)
	err := binary.Read(buffer, binary.BigEndian, &rd)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal struct: %v", err)
	}

	// This is mostly for test vector MAC validation, might want to add more checks.
	mac := rd.Mac[:]
	if rd.Mac == [6]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff} {
		mac = nil
	}

	battery := rd.PowerInfo >> 5
	txPower := rd.PowerInfo & 0x001F

	return &RuuviData{
		DataFormat:  RAWv2,
		Temperature: nilInvF32(-32768, int(rd.Temperature), float32(rd.Temperature)*0.005),
		Humidity:    nilInvF32(65535, int(rd.Humidity), float32(rd.Humidity)*0.0025),
		Pressure:    nilInvF32(65535, int(rd.Pressure), (float32(rd.Pressure)+50000.0)/100.0),
		Acceleration: [3]*float32{
			nilInvF32(-32768, int(rd.AccelX), float32(rd.AccelX)/1000.0),
			nilInvF32(-32768, int(rd.AccelY), float32(rd.AccelY)/1000.0),
			nilInvF32(-32768, int(rd.AccelZ), float32(rd.AccelZ)/1000.0),
		},
		BatteryVoltage:    nilInvF32(2047, int(battery), (float32(battery)+1600.0)/1000.0),
		TxPower:           nilInvF32(31, int(txPower), float32(txPower)*2.0-40.0),
		MovementCounter:   nilInvUint(255, int(rd.MovementCounter), uint(rd.MovementCounter)),
		MeasurementSeqNum: nilInvUint(65535, int(rd.MeasurementSeqNum), uint(rd.MeasurementSeqNum)),
		Mac:               mac,
	}, nil
}
