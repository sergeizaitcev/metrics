// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v4.25.1
// source: metrics/metrics.proto

package metrics

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MetricType int32

const (
	MetricType_UNSPECIFIED MetricType = 0
	MetricType_COUNTER     MetricType = 1
	MetricType_GAUGE       MetricType = 2
)

// Enum value maps for MetricType.
var (
	MetricType_name = map[int32]string{
		0: "UNSPECIFIED",
		1: "COUNTER",
		2: "GAUGE",
	}
	MetricType_value = map[string]int32{
		"UNSPECIFIED": 0,
		"COUNTER":     1,
		"GAUGE":       2,
	}
)

func (x MetricType) Enum() *MetricType {
	p := new(MetricType)
	*p = x
	return p
}

func (x MetricType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (MetricType) Descriptor() protoreflect.EnumDescriptor {
	return file_metrics_metrics_proto_enumTypes[0].Descriptor()
}

func (MetricType) Type() protoreflect.EnumType {
	return &file_metrics_metrics_proto_enumTypes[0]
}

func (x MetricType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use MetricType.Descriptor instead.
func (MetricType) EnumDescriptor() ([]byte, []int) {
	return file_metrics_metrics_proto_rawDescGZIP(), []int{0}
}

type UpdateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metrics []*Metric `protobuf:"bytes,1,rep,name=metrics,proto3" json:"metrics,omitempty"`
}

func (x *UpdateRequest) Reset() {
	*x = UpdateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metrics_metrics_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateRequest) ProtoMessage() {}

func (x *UpdateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_metrics_metrics_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateRequest.ProtoReflect.Descriptor instead.
func (*UpdateRequest) Descriptor() ([]byte, []int) {
	return file_metrics_metrics_proto_rawDescGZIP(), []int{0}
}

func (x *UpdateRequest) GetMetrics() []*Metric {
	if x != nil {
		return x.Metrics
	}
	return nil
}

type Metric struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type  MetricType `protobuf:"varint,1,opt,name=type,proto3,enum=metrics.MetricType" json:"type,omitempty"`
	Name  string     `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Value float64    `protobuf:"fixed64,3,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Metric) Reset() {
	*x = Metric{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metrics_metrics_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metric) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metric) ProtoMessage() {}

func (x *Metric) ProtoReflect() protoreflect.Message {
	mi := &file_metrics_metrics_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Metric.ProtoReflect.Descriptor instead.
func (*Metric) Descriptor() ([]byte, []int) {
	return file_metrics_metrics_proto_rawDescGZIP(), []int{1}
}

func (x *Metric) GetType() MetricType {
	if x != nil {
		return x.Type
	}
	return MetricType_UNSPECIFIED
}

func (x *Metric) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Metric) GetValue() float64 {
	if x != nil {
		return x.Value
	}
	return 0
}

var File_metrics_metrics_proto protoreflect.FileDescriptor

var file_metrics_metrics_proto_rawDesc = []byte{
	0x0a, 0x15, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3a, 0x0a,
	0x0d, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x29,
	0x0a, 0x07, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x0f, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x52, 0x07, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x22, 0x5b, 0x0a, 0x06, 0x4d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x12, 0x27, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x13, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x4d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x01, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2a, 0x35, 0x0a, 0x0a, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x54, 0x79, 0x70, 0x65, 0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46,
	0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x43, 0x4f, 0x55, 0x4e, 0x54, 0x45, 0x52,
	0x10, 0x01, 0x12, 0x09, 0x0a, 0x05, 0x47, 0x41, 0x55, 0x47, 0x45, 0x10, 0x02, 0x32, 0x45, 0x0a,
	0x07, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x12, 0x3a, 0x0a, 0x06, 0x55, 0x70, 0x64, 0x61,
	0x74, 0x65, 0x12, 0x16, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x55, 0x70, 0x64,
	0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x00, 0x42, 0x13, 0x5a, 0x11, 0x2e, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x3b, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_metrics_metrics_proto_rawDescOnce sync.Once
	file_metrics_metrics_proto_rawDescData = file_metrics_metrics_proto_rawDesc
)

func file_metrics_metrics_proto_rawDescGZIP() []byte {
	file_metrics_metrics_proto_rawDescOnce.Do(func() {
		file_metrics_metrics_proto_rawDescData = protoimpl.X.CompressGZIP(file_metrics_metrics_proto_rawDescData)
	})
	return file_metrics_metrics_proto_rawDescData
}

var (
	file_metrics_metrics_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
	file_metrics_metrics_proto_msgTypes  = make([]protoimpl.MessageInfo, 2)
	file_metrics_metrics_proto_goTypes   = []interface{}{
		(MetricType)(0),       // 0: metrics.MetricType
		(*UpdateRequest)(nil), // 1: metrics.UpdateRequest
		(*Metric)(nil),        // 2: metrics.Metric
		(*emptypb.Empty)(nil), // 3: google.protobuf.Empty
	}
)
var file_metrics_metrics_proto_depIdxs = []int32{
	2, // 0: metrics.UpdateRequest.metrics:type_name -> metrics.Metric
	0, // 1: metrics.Metric.type:type_name -> metrics.MetricType
	1, // 2: metrics.Metrics.Update:input_type -> metrics.UpdateRequest
	3, // 3: metrics.Metrics.Update:output_type -> google.protobuf.Empty
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_metrics_metrics_proto_init() }
func file_metrics_metrics_proto_init() {
	if File_metrics_metrics_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_metrics_metrics_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateRequest); i {
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
		file_metrics_metrics_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Metric); i {
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
			RawDescriptor: file_metrics_metrics_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_metrics_metrics_proto_goTypes,
		DependencyIndexes: file_metrics_metrics_proto_depIdxs,
		EnumInfos:         file_metrics_metrics_proto_enumTypes,
		MessageInfos:      file_metrics_metrics_proto_msgTypes,
	}.Build()
	File_metrics_metrics_proto = out.File
	file_metrics_metrics_proto_rawDesc = nil
	file_metrics_metrics_proto_goTypes = nil
	file_metrics_metrics_proto_depIdxs = nil
}
