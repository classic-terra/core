// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: terra/wasm/v1beta1/wasm.proto

package legacy

import (
	bytes "bytes"
	encoding_json "encoding/json"
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/gogo/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = proto.Marshal
	_ = fmt.Errorf
	_ = math.Inf
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// CodeInfo is data for the uploaded contract WASM code
type LegacyCodeInfo struct {
	// CodeID is the sequentially increasing unique identifier
	CodeID uint64 `protobuf:"varint,1,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty" yaml:"code_id"`
	// CodeHash is the unique identifier created by wasmvm
	CodeHash []byte `protobuf:"bytes,2,opt,name=code_hash,json=codeHash,proto3" json:"code_hash,omitempty" yaml:"code_hash"`
	// Creator address who initially stored the code
	Creator string `protobuf:"bytes,3,opt,name=creator,proto3" json:"creator,omitempty" yaml:"creator"`
}

func (m *LegacyCodeInfo) Reset()         { *m = LegacyCodeInfo{} }
func (m *LegacyCodeInfo) String() string { return proto.CompactTextString(m) }
func (*LegacyCodeInfo) ProtoMessage()    {}
func (*LegacyCodeInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bd5d0123068c880, []int{0}
}

func (m *LegacyCodeInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *LegacyCodeInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LegacyCodeInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}

func (m *LegacyCodeInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LegacyCodeInfo.Merge(m, src)
}

func (m *LegacyCodeInfo) XXX_Size() int {
	return m.Size()
}

func (m *LegacyCodeInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_LegacyCodeInfo.DiscardUnknown(m)
}

var xxx_messageInfo_LegacyCodeInfo proto.InternalMessageInfo

func (m *LegacyCodeInfo) GetCodeID() uint64 {
	if m != nil {
		return m.CodeID
	}
	return 0
}

func (m *LegacyCodeInfo) GetCodeHash() []byte {
	if m != nil {
		return m.CodeHash
	}
	return nil
}

func (m *LegacyCodeInfo) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

// ContractInfo stores a WASM contract instance
type LegacyContractInfo struct {
	// Address is the address of the contract
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty" yaml:"address"`
	// Creator is the contract creator address
	Creator string `protobuf:"bytes,2,opt,name=creator,proto3" json:"creator,omitempty" yaml:"creator"`
	// Admin is who can execute the contract migration
	Admin string `protobuf:"bytes,3,opt,name=admin,proto3" json:"admin,omitempty" yaml:"admin"`
	// CodeID is the reference to the stored Wasm code
	CodeID uint64 `protobuf:"varint,4,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty" yaml:"code_id"`
	// InitMsg is the raw message used when instantiating a contract
	InitMsg encoding_json.RawMessage `protobuf:"bytes,5,opt,name=init_msg,json=initMsg,proto3,casttype=encoding/json.RawMessage" json:"init_msg,omitempty" yaml:"init_msg"`
}

func (m *LegacyContractInfo) Reset()         { *m = LegacyContractInfo{} }
func (m *LegacyContractInfo) String() string { return proto.CompactTextString(m) }
func (*LegacyContractInfo) ProtoMessage()    {}
func (*LegacyContractInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bd5d0123068c880, []int{1}
}

func (m *LegacyContractInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *LegacyContractInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LegacyContractInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}

func (m *LegacyContractInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LegacyContractInfo.Merge(m, src)
}

func (m *LegacyContractInfo) XXX_Size() int {
	return m.Size()
}

func (m *LegacyContractInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_LegacyContractInfo.DiscardUnknown(m)
}

var xxx_messageInfo_LegacyContractInfo proto.InternalMessageInfo

func (m *LegacyContractInfo) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *LegacyContractInfo) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *LegacyContractInfo) GetAdmin() string {
	if m != nil {
		return m.Admin
	}
	return ""
}

func (m *LegacyContractInfo) GetCodeID() uint64 {
	if m != nil {
		return m.CodeID
	}
	return 0
}

func (m *LegacyContractInfo) GetInitMsg() encoding_json.RawMessage {
	if m != nil {
		return m.InitMsg
	}
	return nil
}

func init() {
	proto.RegisterType((*LegacyCodeInfo)(nil), "terra.wasm.v1beta1.LegacyCodeInfo")
	proto.RegisterType((*LegacyContractInfo)(nil), "terra.wasm.v1beta1.LegacyContractInfo")
}

func init() { proto.RegisterFile("terra/wasm/v1beta1/wasm.proto", fileDescriptor_2bd5d0123068c880) }

var fileDescriptor_2bd5d0123068c880 = []byte{
	// 407 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0x31, 0x6f, 0xd4, 0x30,
	0x14, 0xc7, 0xcf, 0xc7, 0xf5, 0xae, 0xb5, 0xaa, 0x52, 0x59, 0x1d, 0x22, 0x04, 0xc9, 0xc9, 0x03,
	0xba, 0xa1, 0x9c, 0x75, 0x20, 0x06, 0x3a, 0x06, 0x06, 0x8a, 0xe8, 0x12, 0x36, 0x96, 0xca, 0x67,
	0x1b, 0xc7, 0xe8, 0x62, 0x57, 0xb6, 0xdb, 0xea, 0xbe, 0x05, 0x1f, 0x01, 0x36, 0x3e, 0x0a, 0x63,
	0x47, 0xa6, 0x08, 0xe5, 0x16, 0xe6, 0x8c, 0x4c, 0x28, 0x4e, 0x22, 0x85, 0x09, 0x75, 0x7b, 0xf6,
	0xef, 0xa7, 0xf7, 0x9e, 0xfe, 0x7a, 0xf0, 0x89, 0x17, 0xd6, 0x52, 0x72, 0x4b, 0x5d, 0x41, 0x6e,
	0x56, 0x6b, 0xe1, 0xe9, 0x2a, 0x3c, 0x96, 0x57, 0xd6, 0x78, 0x83, 0x50, 0xc0, 0xcb, 0xf0, 0xd3,
	0xe1, 0x47, 0x27, 0xd2, 0x48, 0x13, 0x30, 0x69, 0xaa, 0xd6, 0xc4, 0xdf, 0x01, 0x3c, 0x7a, 0x2f,
	0x24, 0x65, 0xdb, 0xd7, 0x86, 0x8b, 0x73, 0xfd, 0xc9, 0xa0, 0x97, 0x70, 0xc6, 0x0c, 0x17, 0x97,
	0x8a, 0x47, 0x60, 0x0e, 0x16, 0x93, 0xf4, 0x71, 0x55, 0x26, 0xd3, 0x80, 0xdf, 0xd4, 0x65, 0x72,
	0xb4, 0xa5, 0xc5, 0xe6, 0x0c, 0x77, 0x0a, 0xce, 0xa6, 0x4d, 0x75, 0xce, 0xd1, 0x0a, 0x1e, 0x84,
	0xbf, 0x9c, 0xba, 0x3c, 0x1a, 0xcf, 0xc1, 0xe2, 0x30, 0x3d, 0xa9, 0xcb, 0xe4, 0x78, 0xa0, 0x37,
	0x08, 0x67, 0xfb, 0x4d, 0xfd, 0x96, 0xba, 0x1c, 0x9d, 0xc2, 0x19, 0xb3, 0x82, 0x7a, 0x63, 0xa3,
	0x07, 0x73, 0xb0, 0x38, 0x48, 0xd1, 0xa0, 0x7f, 0x0b, 0x70, 0xd6, 0x2b, 0xf8, 0xdb, 0x18, 0xa2,
	0x7e, 0x55, 0xed, 0x2d, 0x65, 0x3e, 0xac, 0x7b, 0x0a, 0x67, 0x94, 0x73, 0x2b, 0x9c, 0x0b, 0xeb,
	0xfe, 0xd3, 0xa4, 0x03, 0x38, 0xeb, 0x95, 0xe1, 0xc8, 0xf1, 0x7f, 0x47, 0xa2, 0xa7, 0x70, 0x8f,
	0xf2, 0x42, 0xe9, 0x6e, 0xbd, 0xe3, 0xba, 0x4c, 0x0e, 0xfb, 0xce, 0x85, 0xd2, 0x38, 0x6b, 0xf1,
	0x30, 0xb2, 0xc9, 0x3d, 0x22, 0x7b, 0x07, 0xf7, 0x95, 0x56, 0xfe, 0xb2, 0x70, 0x32, 0xda, 0x0b,
	0x89, 0x91, 0xba, 0x4c, 0x1e, 0xb6, 0x76, 0x4f, 0xf0, 0x9f, 0x32, 0x89, 0x84, 0x66, 0x86, 0x2b,
	0x2d, 0xc9, 0x67, 0x67, 0xf4, 0x32, 0xa3, 0xb7, 0x17, 0xc2, 0x39, 0x2a, 0x45, 0x36, 0x6b, 0xb4,
	0x0b, 0x27, 0xcf, 0x26, 0xbf, 0xbf, 0x26, 0x20, 0xfd, 0xf0, 0xa3, 0x8a, 0xc1, 0x5d, 0x15, 0x83,
	0x5f, 0x55, 0x0c, 0xbe, 0xec, 0xe2, 0xd1, 0xdd, 0x2e, 0x1e, 0xfd, 0xdc, 0xc5, 0xa3, 0x8f, 0xaf,
	0xa4, 0xf2, 0xf9, 0xf5, 0x7a, 0xc9, 0x4c, 0x41, 0xd8, 0x86, 0x3a, 0xa7, 0xd8, 0xb3, 0xf6, 0x88,
	0x98, 0xb1, 0x82, 0xdc, 0x3c, 0x27, 0xec, 0xda, 0x79, 0x53, 0xb4, 0x37, 0xe5, 0xb7, 0x57, 0xc2,
	0x91, 0x4d, 0x48, 0x7b, 0x3d, 0x0d, 0xa7, 0xf2, 0xe2, 0x6f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xed,
	0x5e, 0x7e, 0xf3, 0x75, 0x02, 0x00, 0x00,
}

func (this *LegacyContractInfo) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*LegacyContractInfo)
	if !ok {
		that2, ok := that.(LegacyContractInfo)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Address != that1.Address {
		return false
	}
	if this.Creator != that1.Creator {
		return false
	}
	if this.Admin != that1.Admin {
		return false
	}
	if this.CodeID != that1.CodeID {
		return false
	}
	if !bytes.Equal(this.InitMsg, that1.InitMsg) {
		return false
	}
	return true
}

func (m *LegacyCodeInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LegacyCodeInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LegacyCodeInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.CodeHash) > 0 {
		i -= len(m.CodeHash)
		copy(dAtA[i:], m.CodeHash)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.CodeHash)))
		i--
		dAtA[i] = 0x12
	}
	if m.CodeID != 0 {
		i = encodeVarintWasm(dAtA, i, uint64(m.CodeID))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *LegacyContractInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LegacyContractInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LegacyContractInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.InitMsg) > 0 {
		i -= len(m.InitMsg)
		copy(dAtA[i:], m.InitMsg)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.InitMsg)))
		i--
		dAtA[i] = 0x2a
	}
	if m.CodeID != 0 {
		i = encodeVarintWasm(dAtA, i, uint64(m.CodeID))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Admin) > 0 {
		i -= len(m.Admin)
		copy(dAtA[i:], m.Admin)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Admin)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintWasm(dAtA []byte, offset int, v uint64) int {
	offset -= sovWasm(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func (m *LegacyCodeInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.CodeID != 0 {
		n += 1 + sovWasm(uint64(m.CodeID))
	}
	l = len(m.CodeHash)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	return n
}

func (m *LegacyContractInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	l = len(m.Admin)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	if m.CodeID != 0 {
		n += 1 + sovWasm(uint64(m.CodeID))
	}
	l = len(m.InitMsg)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	return n
}

func sovWasm(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}

func sozWasm(x uint64) (n int) {
	return sovWasm(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}

func (m *LegacyCodeInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowWasm
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: LegacyCodeInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LegacyCodeInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CodeID", wireType)
			}
			m.CodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CodeID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CodeHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CodeHash = append(m.CodeHash[:0], dAtA[iNdEx:postIndex]...)
			if m.CodeHash == nil {
				m.CodeHash = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipWasm(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthWasm
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (m *LegacyContractInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowWasm
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: LegacyContractInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LegacyContractInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Admin", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Admin = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CodeID", wireType)
			}
			m.CodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CodeID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InitMsg", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InitMsg = append(m.InitMsg[:0], dAtA[iNdEx:postIndex]...)
			if m.InitMsg == nil {
				m.InitMsg = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipWasm(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthWasm
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func skipWasm(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowWasm
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthWasm
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupWasm
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthWasm
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthWasm        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowWasm          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupWasm = fmt.Errorf("proto: unexpected end of group")
)