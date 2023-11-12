// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v4.25.0
// source: example.proto

package tests

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

type MemberSheetRow struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID      int32          `protobuf:"varint,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Name    string         `protobuf:"bytes,2,opt,name=Name,proto3" json:"Name,omitempty"`
	Address string         `protobuf:"bytes,3,opt,name=Address,proto3" json:"Address,omitempty"`
	Phone   *PhoneNumber   `protobuf:"bytes,4,opt,name=Phone,proto3" json:"Phone,omitempty"`
	Cities  []string       `protobuf:"bytes,5,rep,name=Cities,proto3" json:"Cities,omitempty"`
	PP      []*PhoneNumber `protobuf:"bytes,6,rep,name=PP,proto3" json:"PP,omitempty"`
}

func (x *MemberSheetRow) Reset() {
	*x = MemberSheetRow{}
	if protoimpl.UnsafeEnabled {
		mi := &file_example_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MemberSheetRow) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MemberSheetRow) ProtoMessage() {}

func (x *MemberSheetRow) ProtoReflect() protoreflect.Message {
	mi := &file_example_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MemberSheetRow.ProtoReflect.Descriptor instead.
func (*MemberSheetRow) Descriptor() ([]byte, []int) {
	return file_example_proto_rawDescGZIP(), []int{0}
}

func (x *MemberSheetRow) GetID() int32 {
	if x != nil {
		return x.ID
	}
	return 0
}

func (x *MemberSheetRow) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *MemberSheetRow) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *MemberSheetRow) GetPhone() *PhoneNumber {
	if x != nil {
		return x.Phone
	}
	return nil
}

func (x *MemberSheetRow) GetCities() []string {
	if x != nil {
		return x.Cities
	}
	return nil
}

func (x *MemberSheetRow) GetPP() []*PhoneNumber {
	if x != nil {
		return x.PP
	}
	return nil
}

type MemberSheet struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	List []*MemberSheetRow `protobuf:"bytes,1,rep,name=List,proto3" json:"List,omitempty"`
}

func (x *MemberSheet) Reset() {
	*x = MemberSheet{}
	if protoimpl.UnsafeEnabled {
		mi := &file_example_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MemberSheet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MemberSheet) ProtoMessage() {}

func (x *MemberSheet) ProtoReflect() protoreflect.Message {
	mi := &file_example_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MemberSheet.ProtoReflect.Descriptor instead.
func (*MemberSheet) Descriptor() ([]byte, []int) {
	return file_example_proto_rawDescGZIP(), []int{1}
}

func (x *MemberSheet) GetList() []*MemberSheetRow {
	if x != nil {
		return x.List
	}
	return nil
}

var File_example_proto protoreflect.FileDescriptor

var file_example_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x0a, 0x64, 0x65, 0x70, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa8, 0x01, 0x0a, 0x0e,
	0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x53, 0x68, 0x65, 0x65, 0x74, 0x52, 0x6f, 0x77, 0x12, 0x0e,
	0x0a, 0x02, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02, 0x49, 0x44, 0x12, 0x12,
	0x0a, 0x04, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x4e, 0x61,
	0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x22, 0x0a, 0x05,
	0x50, 0x68, 0x6f, 0x6e, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x50, 0x68,
	0x6f, 0x6e, 0x65, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x05, 0x50, 0x68, 0x6f, 0x6e, 0x65,
	0x12, 0x16, 0x0a, 0x06, 0x43, 0x69, 0x74, 0x69, 0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x06, 0x43, 0x69, 0x74, 0x69, 0x65, 0x73, 0x12, 0x1c, 0x0a, 0x02, 0x50, 0x50, 0x18, 0x06,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x50, 0x68, 0x6f, 0x6e, 0x65, 0x4e, 0x75, 0x6d, 0x62,
	0x65, 0x72, 0x52, 0x02, 0x50, 0x50, 0x22, 0x32, 0x0a, 0x0b, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72,
	0x53, 0x68, 0x65, 0x65, 0x74, 0x12, 0x23, 0x0a, 0x04, 0x4c, 0x69, 0x73, 0x74, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x53, 0x68, 0x65, 0x65,
	0x74, 0x52, 0x6f, 0x77, 0x52, 0x04, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x25, 0x5a, 0x23, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x61, 0x73, 0x68, 0x65, 0x6e, 0x67,
	0x79, 0x65, 0x61, 0x68, 0x2f, 0x78, 0x6c, 0x73, 0x63, 0x66, 0x67, 0x2f, 0x74, 0x65, 0x73, 0x74,
	0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_example_proto_rawDescOnce sync.Once
	file_example_proto_rawDescData = file_example_proto_rawDesc
)

func file_example_proto_rawDescGZIP() []byte {
	file_example_proto_rawDescOnce.Do(func() {
		file_example_proto_rawDescData = protoimpl.X.CompressGZIP(file_example_proto_rawDescData)
	})
	return file_example_proto_rawDescData
}

var file_example_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_example_proto_goTypes = []interface{}{
	(*MemberSheetRow)(nil), // 0: MemberSheetRow
	(*MemberSheet)(nil),    // 1: MemberSheet
	(*PhoneNumber)(nil),    // 2: PhoneNumber
}
var file_example_proto_depIdxs = []int32{
	2, // 0: MemberSheetRow.Phone:type_name -> PhoneNumber
	2, // 1: MemberSheetRow.PP:type_name -> PhoneNumber
	0, // 2: MemberSheet.List:type_name -> MemberSheetRow
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_example_proto_init() }
func file_example_proto_init() {
	if File_example_proto != nil {
		return
	}
	file_deps_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_example_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MemberSheetRow); i {
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
		file_example_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MemberSheet); i {
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
			RawDescriptor: file_example_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_example_proto_goTypes,
		DependencyIndexes: file_example_proto_depIdxs,
		MessageInfos:      file_example_proto_msgTypes,
	}.Build()
	File_example_proto = out.File
	file_example_proto_rawDesc = nil
	file_example_proto_goTypes = nil
	file_example_proto_depIdxs = nil
}
