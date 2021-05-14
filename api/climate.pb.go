// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.17.0
// source: api/climate.proto

package gbcsdpd_api_v1

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type Measurement struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SensorMac      string  `protobuf:"bytes,1,opt,name=sensor_mac,json=sensorMac,proto3" json:"sensor_mac,omitempty"`
	Temperature    float32 `protobuf:"fixed32,10,opt,name=temperature,proto3" json:"temperature,omitempty"`
	Humidity       float32 `protobuf:"fixed32,11,opt,name=humidity,proto3" json:"humidity,omitempty"`
	Pressure       float32 `protobuf:"fixed32,12,opt,name=pressure,proto3" json:"pressure,omitempty"`
	BatteryVoltage float32 `protobuf:"fixed32,20,opt,name=battery_voltage,json=batteryVoltage,proto3" json:"battery_voltage,omitempty"`
}

func (x *Measurement) Reset() {
	*x = Measurement{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_climate_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Measurement) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Measurement) ProtoMessage() {}

func (x *Measurement) ProtoReflect() protoreflect.Message {
	mi := &file_api_climate_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Measurement.ProtoReflect.Descriptor instead.
func (*Measurement) Descriptor() ([]byte, []int) {
	return file_api_climate_proto_rawDescGZIP(), []int{0}
}

func (x *Measurement) GetSensorMac() string {
	if x != nil {
		return x.SensorMac
	}
	return ""
}

func (x *Measurement) GetTemperature() float32 {
	if x != nil {
		return x.Temperature
	}
	return 0
}

func (x *Measurement) GetHumidity() float32 {
	if x != nil {
		return x.Humidity
	}
	return 0
}

func (x *Measurement) GetPressure() float32 {
	if x != nil {
		return x.Pressure
	}
	return 0
}

func (x *Measurement) GetBatteryVoltage() float32 {
	if x != nil {
		return x.BatteryVoltage
	}
	return 0
}

type MeasurementsPublication struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Measurements []*Measurement `protobuf:"bytes,1,rep,name=measurements,proto3" json:"measurements,omitempty"`
}

func (x *MeasurementsPublication) Reset() {
	*x = MeasurementsPublication{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_climate_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MeasurementsPublication) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MeasurementsPublication) ProtoMessage() {}

func (x *MeasurementsPublication) ProtoReflect() protoreflect.Message {
	mi := &file_api_climate_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MeasurementsPublication.ProtoReflect.Descriptor instead.
func (*MeasurementsPublication) Descriptor() ([]byte, []int) {
	return file_api_climate_proto_rawDescGZIP(), []int{1}
}

func (x *MeasurementsPublication) GetMeasurements() []*Measurement {
	if x != nil {
		return x.Measurements
	}
	return nil
}

var File_api_climate_proto protoreflect.FileDescriptor

var file_api_climate_proto_rawDesc = []byte{
	0x0a, 0x11, 0x61, 0x70, 0x69, 0x2f, 0x63, 0x6c, 0x69, 0x6d, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x67, 0x62, 0x63, 0x73, 0x64, 0x70, 0x64, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x76, 0x31, 0x22, 0xaf, 0x01, 0x0a, 0x0b, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x6d,
	0x65, 0x6e, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x6e, 0x73, 0x6f, 0x72, 0x5f, 0x6d, 0x61,
	0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x65, 0x6e, 0x73, 0x6f, 0x72, 0x4d,
	0x61, 0x63, 0x12, 0x20, 0x0a, 0x0b, 0x74, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x02, 0x52, 0x0b, 0x74, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x68, 0x75, 0x6d, 0x69, 0x64, 0x69, 0x74, 0x79,
	0x18, 0x0b, 0x20, 0x01, 0x28, 0x02, 0x52, 0x08, 0x68, 0x75, 0x6d, 0x69, 0x64, 0x69, 0x74, 0x79,
	0x12, 0x1a, 0x0a, 0x08, 0x70, 0x72, 0x65, 0x73, 0x73, 0x75, 0x72, 0x65, 0x18, 0x0c, 0x20, 0x01,
	0x28, 0x02, 0x52, 0x08, 0x70, 0x72, 0x65, 0x73, 0x73, 0x75, 0x72, 0x65, 0x12, 0x27, 0x0a, 0x0f,
	0x62, 0x61, 0x74, 0x74, 0x65, 0x72, 0x79, 0x5f, 0x76, 0x6f, 0x6c, 0x74, 0x61, 0x67, 0x65, 0x18,
	0x14, 0x20, 0x01, 0x28, 0x02, 0x52, 0x0e, 0x62, 0x61, 0x74, 0x74, 0x65, 0x72, 0x79, 0x56, 0x6f,
	0x6c, 0x74, 0x61, 0x67, 0x65, 0x22, 0x5a, 0x0a, 0x17, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65,
	0x6d, 0x65, 0x6e, 0x74, 0x73, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x3f, 0x0a, 0x0c, 0x6d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x67, 0x62, 0x63, 0x73, 0x64, 0x70, 0x64,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x6d,
	0x65, 0x6e, 0x74, 0x52, 0x0c, 0x6d, 0x65, 0x61, 0x73, 0x75, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74,
	0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_climate_proto_rawDescOnce sync.Once
	file_api_climate_proto_rawDescData = file_api_climate_proto_rawDesc
)

func file_api_climate_proto_rawDescGZIP() []byte {
	file_api_climate_proto_rawDescOnce.Do(func() {
		file_api_climate_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_climate_proto_rawDescData)
	})
	return file_api_climate_proto_rawDescData
}

var file_api_climate_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_api_climate_proto_goTypes = []interface{}{
	(*Measurement)(nil),             // 0: gbcsdpd.api.v1.Measurement
	(*MeasurementsPublication)(nil), // 1: gbcsdpd.api.v1.MeasurementsPublication
}
var file_api_climate_proto_depIdxs = []int32{
	0, // 0: gbcsdpd.api.v1.MeasurementsPublication.measurements:type_name -> gbcsdpd.api.v1.Measurement
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_api_climate_proto_init() }
func file_api_climate_proto_init() {
	if File_api_climate_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_climate_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Measurement); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_climate_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MeasurementsPublication); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_climate_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_climate_proto_goTypes,
		DependencyIndexes: file_api_climate_proto_depIdxs,
		MessageInfos:      file_api_climate_proto_msgTypes,
	}.Build()
	File_api_climate_proto = out.File
	file_api_climate_proto_rawDesc = nil
	file_api_climate_proto_goTypes = nil
	file_api_climate_proto_depIdxs = nil
}