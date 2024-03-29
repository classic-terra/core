// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: terra/dyncomm/v1beta1/dyncomm.proto

package types

import (
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
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

// Params defines the parameters for the dyncomm module.
type Params struct {
	MaxZero       github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,1,opt,name=max_zero,json=maxZero,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"max_zero" yaml:"max_zero"`
	SlopeBase     github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,2,opt,name=slope_base,json=slopeBase,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"slope_base" yaml:"slope_base"`
	SlopeVpImpact github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,3,opt,name=slope_vp_impact,json=slopeVpImpact,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"slope_vp_impact" yaml:"slope_vp_impact"`
	Cap           github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,4,opt,name=cap,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"cap" yaml:"cap"`
}

func (m *Params) Reset()      { *m = Params{} }
func (*Params) ProtoMessage() {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_960758a428b59bad, []int{0}
}

func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}

func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}

func (m *Params) XXX_Size() int {
	return m.Size()
}

func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Params)(nil), "terra.dyncomm.v1beta1.Params")
}

func init() {
	proto.RegisterFile("terra/dyncomm/v1beta1/dyncomm.proto", fileDescriptor_960758a428b59bad)
}

var fileDescriptor_960758a428b59bad = []byte{
	// 338 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0x31, 0x4f, 0xc2, 0x40,
	0x14, 0xc7, 0x5b, 0x31, 0x08, 0x97, 0x18, 0x62, 0xa3, 0xa6, 0x71, 0x68, 0x4d, 0x4d, 0x8c, 0x0b,
	0x3d, 0xd1, 0x8d, 0xb8, 0x88, 0x0e, 0xea, 0x60, 0x4c, 0x07, 0x07, 0x62, 0x42, 0x5e, 0x8f, 0x0b,
	0x12, 0x39, 0xdf, 0xe5, 0xae, 0x12, 0xf0, 0x53, 0x38, 0x19, 0x47, 0x3e, 0x0e, 0x23, 0xa3, 0x71,
	0x20, 0x06, 0x16, 0x67, 0x3f, 0x81, 0xe1, 0x0a, 0xc8, 0xda, 0xe9, 0xee, 0xff, 0xee, 0x9f, 0xdf,
	0x6f, 0xb8, 0x47, 0x0e, 0x12, 0xae, 0x14, 0xd0, 0x66, 0xff, 0x99, 0xa1, 0x10, 0xb4, 0x5b, 0x89,
	0x79, 0x02, 0x95, 0x45, 0x0e, 0xa5, 0xc2, 0x04, 0x9d, 0x1d, 0x53, 0x0a, 0x17, 0xc3, 0x79, 0x69,
	0x6f, 0xbb, 0x85, 0x2d, 0x34, 0x0d, 0x3a, 0xbb, 0xa5, 0xe5, 0xe0, 0x3d, 0x47, 0xf2, 0x77, 0xa0,
	0x40, 0x68, 0xe7, 0x81, 0x14, 0x04, 0xf4, 0x1a, 0xaf, 0x5c, 0xa1, 0x6b, 0xef, 0xdb, 0x47, 0xc5,
	0xda, 0xf9, 0x70, 0xec, 0x5b, 0x5f, 0x63, 0xff, 0xb0, 0xd5, 0x4e, 0x1e, 0x5f, 0xe2, 0x90, 0xa1,
	0xa0, 0x0c, 0xb5, 0x40, 0x3d, 0x3f, 0xca, 0xba, 0xf9, 0x44, 0x93, 0xbe, 0xe4, 0x3a, 0xbc, 0xe4,
	0xec, 0x77, 0xec, 0x97, 0xfa, 0x20, 0x3a, 0xd5, 0x60, 0xc1, 0x09, 0xa2, 0x0d, 0x01, 0xbd, 0x3a,
	0x57, 0xe8, 0xc4, 0x84, 0xe8, 0x0e, 0x4a, 0xde, 0x88, 0x41, 0x73, 0x77, 0xcd, 0xf0, 0x2f, 0x32,
	0xf3, 0xb7, 0x52, 0xfe, 0x3f, 0x29, 0x88, 0x8a, 0x26, 0xd4, 0x40, 0x73, 0x47, 0x92, 0x52, 0xfa,
	0xd2, 0x95, 0x8d, 0xb6, 0x90, 0xc0, 0x12, 0x37, 0x67, 0x44, 0x57, 0x99, 0x45, 0xbb, 0xab, 0xa2,
	0x25, 0x2e, 0x88, 0x36, 0xcd, 0xe4, 0x5e, 0x5e, 0x9b, 0xec, 0xdc, 0x92, 0x1c, 0x03, 0xe9, 0xae,
	0x1b, 0xcb, 0x59, 0x66, 0x0b, 0x49, 0x2d, 0x0c, 0x64, 0x10, 0xcd, 0x40, 0xd5, 0xc2, 0xc7, 0xc0,
	0xb7, 0x7e, 0x06, 0xbe, 0x5d, 0xbb, 0x19, 0x4e, 0x3c, 0x7b, 0x34, 0xf1, 0xec, 0xef, 0x89, 0x67,
	0xbf, 0x4d, 0x3d, 0x6b, 0x34, 0xf5, 0xac, 0xcf, 0xa9, 0x67, 0xd5, 0x8f, 0x57, 0xf1, 0x1d, 0xd0,
	0xba, 0xcd, 0xca, 0xe9, 0x5e, 0x30, 0x54, 0x9c, 0x76, 0x4f, 0x68, 0x6f, 0xb9, 0x21, 0x46, 0x16,
	0xe7, 0xcd, 0x5f, 0x9f, 0xfe, 0x05, 0x00, 0x00, 0xff, 0xff, 0x5d, 0x3a, 0x1d, 0x57, 0x3f, 0x02,
	0x00, 0x00,
}

func (this *Params) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Params)
	if !ok {
		that2, ok := that.(Params)
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
	if !this.MaxZero.Equal(that1.MaxZero) {
		return false
	}
	if !this.SlopeBase.Equal(that1.SlopeBase) {
		return false
	}
	if !this.SlopeVpImpact.Equal(that1.SlopeVpImpact) {
		return false
	}
	if !this.Cap.Equal(that1.Cap) {
		return false
	}
	return true
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Cap.Size()
		i -= size
		if _, err := m.Cap.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintDyncomm(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	{
		size := m.SlopeVpImpact.Size()
		i -= size
		if _, err := m.SlopeVpImpact.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintDyncomm(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	{
		size := m.SlopeBase.Size()
		i -= size
		if _, err := m.SlopeBase.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintDyncomm(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size := m.MaxZero.Size()
		i -= size
		if _, err := m.MaxZero.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintDyncomm(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintDyncomm(dAtA []byte, offset int, v uint64) int {
	offset -= sovDyncomm(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.MaxZero.Size()
	n += 1 + l + sovDyncomm(uint64(l))
	l = m.SlopeBase.Size()
	n += 1 + l + sovDyncomm(uint64(l))
	l = m.SlopeVpImpact.Size()
	n += 1 + l + sovDyncomm(uint64(l))
	l = m.Cap.Size()
	n += 1 + l + sovDyncomm(uint64(l))
	return n
}

func sovDyncomm(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}

func sozDyncomm(x uint64) (n int) {
	return sovDyncomm(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}

func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDyncomm
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxZero", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDyncomm
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
				return ErrInvalidLengthDyncomm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDyncomm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MaxZero.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SlopeBase", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDyncomm
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
				return ErrInvalidLengthDyncomm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDyncomm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.SlopeBase.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SlopeVpImpact", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDyncomm
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
				return ErrInvalidLengthDyncomm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDyncomm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.SlopeVpImpact.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Cap", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDyncomm
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
				return ErrInvalidLengthDyncomm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDyncomm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Cap.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDyncomm(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDyncomm
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

func skipDyncomm(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDyncomm
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
					return 0, ErrIntOverflowDyncomm
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
					return 0, ErrIntOverflowDyncomm
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
				return 0, ErrInvalidLengthDyncomm
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDyncomm
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDyncomm
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDyncomm        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDyncomm          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDyncomm = fmt.Errorf("proto: unexpected end of group")
)
