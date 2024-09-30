// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v5.27.1
// source: pkg/iac/scanners/terraformplan/snapshot/planproto/planfile.proto

package planproto

import (
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

type DynamicValue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Msgpack []byte `protobuf:"bytes,1,opt,name=msgpack,proto3" json:"msgpack,omitempty"`
}

func (x *DynamicValue) Reset() {
	*x = DynamicValue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DynamicValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DynamicValue) ProtoMessage() {}

func (x *DynamicValue) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DynamicValue.ProtoReflect.Descriptor instead.
func (*DynamicValue) Descriptor() ([]byte, []int) {
	return file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescGZIP(), []int{0}
}

func (x *DynamicValue) GetMsgpack() []byte {
	if x != nil {
		return x.Msgpack
	}
	return nil
}

type Plan struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Variables map[string]*DynamicValue `protobuf:"bytes,2,rep,name=variables,proto3" json:"variables,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Plan) Reset() {
	*x = Plan{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Plan) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Plan) ProtoMessage() {}

func (x *Plan) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Plan.ProtoReflect.Descriptor instead.
func (*Plan) Descriptor() ([]byte, []int) {
	return file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescGZIP(), []int{1}
}

func (x *Plan) GetVariables() map[string]*DynamicValue {
	if x != nil {
		return x.Variables
	}
	return nil
}

var File_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto protoreflect.FileDescriptor

var file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDesc = []byte{
	0x0a, 0x40, 0x70, 0x6b, 0x67, 0x2f, 0x69, 0x61, 0x63, 0x2f, 0x73, 0x63, 0x61, 0x6e, 0x6e, 0x65,
	0x72, 0x73, 0x2f, 0x74, 0x65, 0x72, 0x72, 0x61, 0x66, 0x6f, 0x72, 0x6d, 0x70, 0x6c, 0x61, 0x6e,
	0x2f, 0x73, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x2f, 0x70, 0x6c, 0x61, 0x6e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x6c, 0x61, 0x6e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x06, 0x74, 0x66, 0x70, 0x6c, 0x61, 0x6e, 0x22, 0x28, 0x0a, 0x0c, 0x44, 0x79,
	0x6e, 0x61, 0x6d, 0x69, 0x63, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x73,
	0x67, 0x70, 0x61, 0x63, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x6d, 0x73, 0x67,
	0x70, 0x61, 0x63, 0x6b, 0x22, 0x95, 0x01, 0x0a, 0x04, 0x50, 0x6c, 0x61, 0x6e, 0x12, 0x39, 0x0a,
	0x09, 0x76, 0x61, 0x72, 0x69, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x1b, 0x2e, 0x74, 0x66, 0x70, 0x6c, 0x61, 0x6e, 0x2e, 0x50, 0x6c, 0x61, 0x6e, 0x2e, 0x56,
	0x61, 0x72, 0x69, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x09, 0x76,
	0x61, 0x72, 0x69, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x1a, 0x52, 0x0a, 0x0e, 0x56, 0x61, 0x72, 0x69,
	0x61, 0x62, 0x6c, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65,
	0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x2a, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x74, 0x66,
	0x70, 0x6c, 0x61, 0x6e, 0x2e, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x4d, 0x5a, 0x4b,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x71, 0x75, 0x61, 0x73,
	0x65, 0x63, 0x75, 0x72, 0x69, 0x74, 0x79, 0x2f, 0x74, 0x72, 0x69, 0x76, 0x79, 0x2f, 0x69, 0x61,
	0x63, 0x2f, 0x73, 0x63, 0x61, 0x6e, 0x6e, 0x65, 0x72, 0x73, 0x2f, 0x74, 0x65, 0x72, 0x72, 0x61,
	0x66, 0x6f, 0x72, 0x6d, 0x70, 0x6c, 0x61, 0x6e, 0x2f, 0x73, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f,
	0x74, 0x2f, 0x70, 0x6c, 0x61, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescOnce sync.Once
	file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescData = file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDesc
)

func file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescGZIP() []byte {
	file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescOnce.Do(func() {
		file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescData)
	})
	return file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDescData
}

var file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_goTypes = []interface{}{
	(*DynamicValue)(nil), // 0: tfplan.DynamicValue
	(*Plan)(nil),         // 1: tfplan.Plan
	nil,                  // 2: tfplan.Plan.VariablesEntry
}
var file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_depIdxs = []int32{
	2, // 0: tfplan.Plan.variables:type_name -> tfplan.Plan.VariablesEntry
	0, // 1: tfplan.Plan.VariablesEntry.value:type_name -> tfplan.DynamicValue
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_init() }
func file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_init() {
	if File_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DynamicValue); i {
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
		file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Plan); i {
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
			RawDescriptor: file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_goTypes,
		DependencyIndexes: file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_depIdxs,
		MessageInfos:      file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_msgTypes,
	}.Build()
	File_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto = out.File
	file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_rawDesc = nil
	file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_goTypes = nil
	file_pkg_iac_scanners_terraformplan_snapshot_planproto_planfile_proto_depIdxs = nil
}
