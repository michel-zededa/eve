// Code generated by protoc-gen-go. DO NOT EDIT.
// source: certs.proto

package certs

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

//This is same as hashAlgorithm in auth/auth.proto
//Keep these two in sync
//XXX: import auth/auth.proto and avoid this duplication
type CertHashAlgorithm int32

const (
	CertHashAlgorithm_HASH_NONE           CertHashAlgorithm = 0
	CertHashAlgorithm_HASH_SHA256_16bytes CertHashAlgorithm = 1
	CertHashAlgorithm_HASH_SHA256_32bytes CertHashAlgorithm = 2
)

var CertHashAlgorithm_name = map[int32]string{
	0: "HASH_NONE",
	1: "HASH_SHA256_16bytes",
	2: "HASH_SHA256_32bytes",
}

var CertHashAlgorithm_value = map[string]int32{
	"HASH_NONE":           0,
	"HASH_SHA256_16bytes": 1,
	"HASH_SHA256_32bytes": 2,
}

func (x CertHashAlgorithm) String() string {
	return proto.EnumName(CertHashAlgorithm_name, int32(x))
}

func (CertHashAlgorithm) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_78c43cca93027bbd, []int{0}
}

type ZCertType int32

const (
	ZCertType_CERT_TYPE_CONTROLLER_SIGNING       ZCertType = 0
	ZCertType_CERT_TYPE_CONTROLLER_INTERMEDIATE  ZCertType = 1
	ZCertType_CERT_TYPE_CONTROLLER_ECDH_EXCHANGE ZCertType = 2
)

var ZCertType_name = map[int32]string{
	0: "CERT_TYPE_CONTROLLER_SIGNING",
	1: "CERT_TYPE_CONTROLLER_INTERMEDIATE",
	2: "CERT_TYPE_CONTROLLER_ECDH_EXCHANGE",
}

var ZCertType_value = map[string]int32{
	"CERT_TYPE_CONTROLLER_SIGNING":       0,
	"CERT_TYPE_CONTROLLER_INTERMEDIATE":  1,
	"CERT_TYPE_CONTROLLER_ECDH_EXCHANGE": 2,
}

func (x ZCertType) String() string {
	return proto.EnumName(ZCertType_name, int32(x))
}

func (ZCertType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_78c43cca93027bbd, []int{1}
}

// This is the response payload for GET /api/v1/edgeDevice/certs
// or /api/v2/edgeDevice/certs
// ZControllerCert carries a set of X.509 certificate and their properties
// from Controller to EVE.
type ZControllerCert struct {
	Certs                []*ZCert `protobuf:"bytes,1,rep,name=certs,proto3" json:"certs,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ZControllerCert) Reset()         { *m = ZControllerCert{} }
func (m *ZControllerCert) String() string { return proto.CompactTextString(m) }
func (*ZControllerCert) ProtoMessage()    {}
func (*ZControllerCert) Descriptor() ([]byte, []int) {
	return fileDescriptor_78c43cca93027bbd, []int{0}
}

func (m *ZControllerCert) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZControllerCert.Unmarshal(m, b)
}
func (m *ZControllerCert) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZControllerCert.Marshal(b, m, deterministic)
}
func (m *ZControllerCert) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZControllerCert.Merge(m, src)
}
func (m *ZControllerCert) XXX_Size() int {
	return xxx_messageInfo_ZControllerCert.Size(m)
}
func (m *ZControllerCert) XXX_DiscardUnknown() {
	xxx_messageInfo_ZControllerCert.DiscardUnknown(m)
}

var xxx_messageInfo_ZControllerCert proto.InternalMessageInfo

func (m *ZControllerCert) GetCerts() []*ZCert {
	if m != nil {
		return m.Certs
	}
	return nil
}

type ZCert struct {
	HashAlgo             CertHashAlgorithm `protobuf:"varint,1,opt,name=hashAlgo,proto3,enum=CertHashAlgorithm" json:"hashAlgo,omitempty"`
	CertHash             []byte            `protobuf:"bytes,2,opt,name=certHash,proto3" json:"certHash,omitempty"`
	Type                 ZCertType         `protobuf:"varint,3,opt,name=type,proto3,enum=ZCertType" json:"type,omitempty"`
	Cert                 []byte            `protobuf:"bytes,4,opt,name=cert,proto3" json:"cert,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ZCert) Reset()         { *m = ZCert{} }
func (m *ZCert) String() string { return proto.CompactTextString(m) }
func (*ZCert) ProtoMessage()    {}
func (*ZCert) Descriptor() ([]byte, []int) {
	return fileDescriptor_78c43cca93027bbd, []int{1}
}

func (m *ZCert) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZCert.Unmarshal(m, b)
}
func (m *ZCert) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZCert.Marshal(b, m, deterministic)
}
func (m *ZCert) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZCert.Merge(m, src)
}
func (m *ZCert) XXX_Size() int {
	return xxx_messageInfo_ZCert.Size(m)
}
func (m *ZCert) XXX_DiscardUnknown() {
	xxx_messageInfo_ZCert.DiscardUnknown(m)
}

var xxx_messageInfo_ZCert proto.InternalMessageInfo

func (m *ZCert) GetHashAlgo() CertHashAlgorithm {
	if m != nil {
		return m.HashAlgo
	}
	return CertHashAlgorithm_HASH_NONE
}

func (m *ZCert) GetCertHash() []byte {
	if m != nil {
		return m.CertHash
	}
	return nil
}

func (m *ZCert) GetType() ZCertType {
	if m != nil {
		return m.Type
	}
	return ZCertType_CERT_TYPE_CONTROLLER_SIGNING
}

func (m *ZCert) GetCert() []byte {
	if m != nil {
		return m.Cert
	}
	return nil
}

func init() {
	proto.RegisterEnum("CertHashAlgorithm", CertHashAlgorithm_name, CertHashAlgorithm_value)
	proto.RegisterEnum("ZCertType", ZCertType_name, ZCertType_value)
	proto.RegisterType((*ZControllerCert)(nil), "ZControllerCert")
	proto.RegisterType((*ZCert)(nil), "ZCert")
}

func init() { proto.RegisterFile("certs.proto", fileDescriptor_78c43cca93027bbd) }

var fileDescriptor_78c43cca93027bbd = []byte{
	// 354 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0x4d, 0x6f, 0xda, 0x40,
	0x10, 0x86, 0x31, 0x5f, 0x82, 0xa1, 0x1f, 0xee, 0xf6, 0x50, 0xab, 0x42, 0xad, 0x4b, 0xd5, 0x0a,
	0x21, 0x75, 0xad, 0x1a, 0x95, 0xbb, 0x6b, 0x36, 0xd8, 0x12, 0x31, 0xd1, 0xe2, 0x43, 0xc2, 0xc5,
	0x02, 0x7b, 0x63, 0x5b, 0x32, 0x59, 0xcb, 0x5e, 0x90, 0x88, 0x72, 0xcf, 0xdf, 0x8e, 0x58, 0x27,
	0x1c, 0x12, 0x6e, 0x3b, 0xcf, 0xfb, 0xec, 0xbc, 0x87, 0x81, 0x5e, 0xc8, 0x0a, 0x51, 0xe2, 0xbc,
	0xe0, 0x82, 0x0f, 0x0c, 0xf8, 0xb8, 0xb2, 0xf9, 0x9d, 0x28, 0x78, 0x96, 0xb1, 0xc2, 0x66, 0x85,
	0x40, 0x7d, 0x68, 0x49, 0x43, 0x53, 0xf4, 0xc6, 0xb0, 0x67, 0xb6, 0xf1, 0xea, 0x88, 0x69, 0x05,
	0x07, 0x8f, 0x0a, 0xb4, 0x24, 0x40, 0x18, 0x3a, 0xc9, 0xba, 0x4c, 0xac, 0x2c, 0xe6, 0x9a, 0xa2,
	0x2b, 0xc3, 0x0f, 0x26, 0xc2, 0x47, 0xc7, 0x79, 0x86, 0x45, 0x2a, 0x92, 0x2d, 0x3d, 0x39, 0xe8,
	0x2b, 0x74, 0x5e, 0x62, 0xad, 0xae, 0x2b, 0xc3, 0x77, 0xf4, 0x34, 0xa3, 0x6f, 0xd0, 0x14, 0x87,
	0x9c, 0x69, 0x0d, 0xb9, 0x07, 0xaa, 0x4a, 0xff, 0x90, 0x33, 0x2a, 0x39, 0x42, 0xd0, 0x3c, 0xba,
	0x5a, 0x53, 0xfe, 0x93, 0xef, 0x91, 0x0f, 0x9f, 0xde, 0xd4, 0xa1, 0xf7, 0xd0, 0x75, 0xac, 0xa5,
	0x13, 0x78, 0x0b, 0x8f, 0xa8, 0x35, 0xf4, 0x05, 0x3e, 0xcb, 0x71, 0xe9, 0x58, 0xe6, 0xbf, 0x49,
	0xf0, 0x77, 0xb2, 0x39, 0x08, 0x56, 0xaa, 0xca, 0xeb, 0x60, 0x6c, 0x56, 0x41, 0x7d, 0xf4, 0x00,
	0xdd, 0x53, 0x39, 0xd2, 0xa1, 0x6f, 0x13, 0xea, 0x07, 0xfe, 0xcd, 0x15, 0x09, 0xec, 0x85, 0xe7,
	0xd3, 0xc5, 0x7c, 0x4e, 0x68, 0xb0, 0x74, 0x67, 0x9e, 0xeb, 0xcd, 0xd4, 0x1a, 0xfa, 0x05, 0x3f,
	0xce, 0x1a, 0xae, 0xe7, 0x13, 0x7a, 0x49, 0xa6, 0xae, 0xe5, 0x13, 0x55, 0x41, 0xbf, 0x61, 0x70,
	0x56, 0x23, 0xf6, 0xd4, 0x09, 0xc8, 0xb5, 0xed, 0x58, 0xde, 0x8c, 0xa8, 0xf5, 0xff, 0x17, 0xf0,
	0x3d, 0xe4, 0x5b, 0x7c, 0xcf, 0x22, 0x16, 0xad, 0x71, 0x98, 0xf1, 0x5d, 0x84, 0x77, 0x25, 0x2b,
	0xf6, 0x69, 0xc8, 0xaa, 0x8b, 0xad, 0x7e, 0xc6, 0xa9, 0x48, 0x76, 0x1b, 0x1c, 0xf2, 0xad, 0x91,
	0xdd, 0xfe, 0x61, 0x51, 0xcc, 0x0c, 0xb6, 0x67, 0xc6, 0x3a, 0x4f, 0x8d, 0x98, 0x1b, 0xf2, 0x4a,
	0x9b, 0xb6, 0x74, 0xc7, 0x4f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x31, 0x84, 0xb0, 0x26, 0xec, 0x01,
	0x00, 0x00,
}
