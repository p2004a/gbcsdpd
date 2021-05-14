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
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"testing"
)

func toB(s string) []byte {
	decoded, err := hex.DecodeString(s)
	if err != nil {
		panic("Got invalid hex string")
	}
	return decoded
}

func f32Ptr(v float32) *float32 { return &v }
func uintPtr(v uint) *uint      { return &v }

func f32PtrToStr(v *float32) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprint(*v)
}

func uintPtrToStr(v *uint) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprint(*v)
}

func macToStr(mac net.HardwareAddr) string {
	if mac == nil {
		return "nil"
	}
	return mac.String()
}

func assertF32Eq(t *testing.T, param string, value, expected *float32) {
	if value == nil || expected == nil {
		if value != expected {
			t.Errorf("%s not equal: value %s, expected %s", param, f32PtrToStr(value), f32PtrToStr(expected))
		}
	} else if math.Abs(float64(*value-*expected)) > 0.0001 {
		t.Errorf("%s not equal: value %s, expected %s", param, f32PtrToStr(value), f32PtrToStr(expected))
	}
}

func assertUintEq(t *testing.T, param string, value, expected *uint) {
	if value == nil || expected == nil {
		if value != expected {
			t.Errorf("%s not equal: value %s, expected %s", param, uintPtrToStr(value), uintPtrToStr(expected))
		}
	} else if *value != *expected {
		t.Errorf("%s not equal: value %s, expected %s", param, uintPtrToStr(value), uintPtrToStr(expected))
	}
}

func TestParsingValid(t *testing.T) {
	cases := []struct {
		name     string
		data     []byte
		expected *RuuviData
	}{
		{"RAWv2 valid", toB("0512FC5394C37C0004FFFC040CAC364200CDCBB8334C884F"), &RuuviData{
			DataFormat:        RAWv2,
			Temperature:       f32Ptr(24.3),
			Pressure:          f32Ptr(1000.44),
			Humidity:          f32Ptr(53.49),
			Acceleration:      [3]*float32{f32Ptr(0.004), f32Ptr(-0.004), f32Ptr(1.036)},
			TxPower:           f32Ptr(4.0),
			BatteryVoltage:    f32Ptr(2.977),
			MovementCounter:   uintPtr(66),
			MeasurementSeqNum: uintPtr(205),
			Mac:               []byte{0xCB, 0xB8, 0x33, 0x4C, 0x88, 0x4F},
		}},
		{"RAWv2 maximum", toB("057FFFFFFEFFFE7FFF7FFF7FFFFFDEFEFFFECBB8334C884F"), &RuuviData{
			DataFormat:        RAWv2,
			Temperature:       f32Ptr(163.835),
			Pressure:          f32Ptr(1155.34),
			Humidity:          f32Ptr(163.8350),
			Acceleration:      [3]*float32{f32Ptr(32.767), f32Ptr(32.767), f32Ptr(32.767)},
			TxPower:           f32Ptr(20),
			BatteryVoltage:    f32Ptr(3.646),
			MovementCounter:   uintPtr(254),
			MeasurementSeqNum: uintPtr(65534),
			Mac:               []byte{0xCB, 0xB8, 0x33, 0x4C, 0x88, 0x4F},
		}},
		{"RAWv2 minimum", toB("058001000000008001800180010000000000CBB8334C884F"), &RuuviData{
			DataFormat:        RAWv2,
			Temperature:       f32Ptr(-163.835),
			Pressure:          f32Ptr(500.00),
			Humidity:          f32Ptr(0.0),
			Acceleration:      [3]*float32{f32Ptr(-32.767), f32Ptr(-32.767), f32Ptr(-32.767)},
			TxPower:           f32Ptr(-40.0),
			BatteryVoltage:    f32Ptr(1.600),
			MovementCounter:   uintPtr(0),
			MeasurementSeqNum: uintPtr(0),
			Mac:               []byte{0xCB, 0xB8, 0x33, 0x4C, 0x88, 0x4F},
		}},
		{"RAWv2 invalid", toB("058000FFFFFFFF800080008000FFFFFFFFFFFFFFFFFFFFFF"), &RuuviData{
			DataFormat:   RAWv2,
			Acceleration: [3]*float32{},
		}},
		{"RAWv1 valid", toB("03291A1ECE1EFC18F94202CA0B53"), &RuuviData{
			DataFormat:     RAWv1,
			Temperature:    f32Ptr(26.3),
			Pressure:       f32Ptr(1027.66),
			Humidity:       f32Ptr(20.5),
			Acceleration:   [3]*float32{f32Ptr(-1.000), f32Ptr(-1.726), f32Ptr(0.714)},
			BatteryVoltage: f32Ptr(2.899),
		}},
		{"RAWv1 maximum", toB("03FF7F63FFFF7FFF7FFF7FFFFFFF"), &RuuviData{
			DataFormat:     RAWv1,
			Temperature:    f32Ptr(127.99),
			Pressure:       f32Ptr(1155.35),
			Humidity:       f32Ptr(127.5),
			Acceleration:   [3]*float32{f32Ptr(32.767), f32Ptr(32.767), f32Ptr(32.767)},
			BatteryVoltage: f32Ptr(65.535),
		}},
		{"RAWv1 minimum", toB("0300FF6300008001800180010000"), &RuuviData{
			DataFormat:     RAWv1,
			Temperature:    f32Ptr(-127.99),
			Pressure:       f32Ptr(500.00),
			Humidity:       f32Ptr(0.0),
			Acceleration:   [3]*float32{f32Ptr(-32.767), f32Ptr(-32.767), f32Ptr(-32.767)},
			BatteryVoltage: f32Ptr(0.0),
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Parse(tc.data)
			if err != nil {
				t.Fatalf("couldn't parse data: %v", err)
			}
			assertF32Eq(t, "temperature", res.Temperature, tc.expected.Temperature)
			assertF32Eq(t, "humidity", res.Humidity, tc.expected.Humidity)
			assertF32Eq(t, "pressure", res.Pressure, tc.expected.Pressure)
			assertF32Eq(t, "acceleration x", res.Acceleration[0], tc.expected.Acceleration[0])
			assertF32Eq(t, "acceleration y", res.Acceleration[1], tc.expected.Acceleration[1])
			assertF32Eq(t, "acceleration z", res.Acceleration[2], tc.expected.Acceleration[2])
			assertF32Eq(t, "tx power", res.TxPower, tc.expected.TxPower)
			assertF32Eq(t, "battery voltage", res.BatteryVoltage, tc.expected.BatteryVoltage)
			assertUintEq(t, "movement counted", res.MovementCounter, tc.expected.MovementCounter)
			assertUintEq(t, "measurement sequence number", res.MeasurementSeqNum, tc.expected.MeasurementSeqNum)
			if res.Mac.String() != tc.expected.Mac.String() {
				t.Errorf("MAC not equal: value %s, expected %s", macToStr(res.Mac), macToStr(tc.expected.Mac))
			}
		})
	}
}

func TestParsingInvalid(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"unsupported format", toB("537FFF")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.data)
			if err == nil {
				t.Fatalf("Expected error, got success")
			}
		})
	}
}
