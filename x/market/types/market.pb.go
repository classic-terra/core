// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: terra/market/v1beta1/market.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Params defines the parameters for the market module.
type Params struct {
	BasePool           github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,1,opt,name=base_pool,json=basePool,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"base_pool" yaml:"base_pool"`
	PoolRecoveryPeriod uint64                                 `protobuf:"varint,2,opt,name=pool_recovery_period,json=poolRecoveryPeriod,proto3" json:"pool_recovery_period,omitempty" yaml:"pool_recovery_period"`
	MinStabilitySpread github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,3,opt,name=min_stability_spread,json=minStabilitySpread,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"min_stability_spread" yaml:"min_stability_spread"`
}

func (m *Params) Reset()      { *m = Params{} }
func (*Params) ProtoMessage() {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_114ea92c5ae3e66f, []int{0}
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

func (m *Params) GetPoolRecoveryPeriod() uint64 {
	if m != nil {
		return m.PoolRecoveryPeriod
	}
	return 0
}

func init() {
	proto.RegisterType((*Params)(nil), "terra.market.v1beta1.Params")
}

func init() { proto.RegisterFile("terra/market/v1beta1/market.proto", fileDescriptor_114ea92c5ae3e66f) }

var fileDescriptor_114ea92c5ae3e66f = []byte{
	// 358 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0xb1, 0x6e, 0xea, 0x30,
	0x18, 0x85, 0x63, 0xee, 0x15, 0xe2, 0x46, 0x77, 0xb8, 0x8a, 0x32, 0x70, 0xa9, 0x94, 0xd0, 0x0c,
	0x15, 0x0b, 0x89, 0x10, 0x1b, 0x23, 0x62, 0xe9, 0x96, 0xc2, 0xd6, 0x0e, 0x91, 0x13, 0x2c, 0x6a,
	0x11, 0xf3, 0x47, 0xb6, 0x8b, 0x9a, 0x87, 0xa8, 0xd4, 0xb1, 0x23, 0x0f, 0xd1, 0x87, 0x60, 0x44,
	0x9d, 0xaa, 0x0e, 0x51, 0x05, 0x1d, 0x3a, 0xf3, 0x04, 0x55, 0x1c, 0x53, 0x55, 0x15, 0x4b, 0xa7,
	0xe4, 0x3f, 0xfe, 0x7c, 0x74, 0x8e, 0x7e, 0x9b, 0xa7, 0x92, 0x70, 0x8e, 0x03, 0x86, 0xf9, 0x9c,
	0xc8, 0x60, 0xd9, 0x8b, 0x89, 0xc4, 0x3d, 0x3d, 0xfa, 0x19, 0x07, 0x09, 0x96, 0xad, 0x10, 0x5f,
	0x6b, 0x1a, 0x69, 0xfd, 0x4f, 0x40, 0x30, 0x10, 0x91, 0x62, 0x82, 0x6a, 0xa8, 0x2e, 0xb4, 0xec,
	0x19, 0xcc, 0xa0, 0xd2, 0xcb, 0xbf, 0x4a, 0xf5, 0xde, 0x6a, 0x66, 0x3d, 0xc4, 0x1c, 0x33, 0x61,
	0x31, 0xf3, 0x4f, 0x8c, 0x05, 0x89, 0x32, 0x80, 0xb4, 0x89, 0xda, 0xa8, 0xf3, 0x77, 0x18, 0xae,
	0x0b, 0xd7, 0x78, 0x29, 0xdc, 0xb3, 0x19, 0x95, 0xd7, 0x37, 0xb1, 0x9f, 0x00, 0xd3, 0xa6, 0xfa,
	0xd3, 0x15, 0xd3, 0x79, 0x20, 0xf3, 0x8c, 0x08, 0x7f, 0x44, 0x92, 0x7d, 0xe1, 0xfe, 0xcb, 0x31,
	0x4b, 0x07, 0xde, 0xa7, 0x91, 0xf7, 0xf4, 0xd8, 0x35, 0x75, 0x8e, 0x11, 0x49, 0xc6, 0x8d, 0xf2,
	0x24, 0x04, 0x48, 0xad, 0x0b, 0xd3, 0x2e, 0x81, 0x88, 0x93, 0x04, 0x96, 0x84, 0xe7, 0x51, 0x46,
	0x38, 0x85, 0x69, 0xb3, 0xd6, 0x46, 0x9d, 0xdf, 0x43, 0x77, 0x5f, 0xb8, 0x27, 0x95, 0xd7, 0x31,
	0xca, 0x1b, 0x5b, 0xa5, 0x3c, 0xd6, 0x6a, 0xa8, 0x44, 0xeb, 0x0e, 0x99, 0x36, 0xa3, 0x8b, 0x48,
	0x48, 0x1c, 0xd3, 0x94, 0xca, 0x3c, 0x12, 0x19, 0x27, 0x78, 0xda, 0xfc, 0xa5, 0xda, 0x5c, 0xfd,
	0xb8, 0x8d, 0x4e, 0x70, 0xcc, 0xf3, 0x7b, 0x31, 0x8b, 0xd1, 0xc5, 0xe4, 0xc0, 0x4c, 0x14, 0x32,
	0x68, 0x3c, 0xac, 0x5c, 0xe3, 0x7d, 0xe5, 0xa2, 0xe1, 0xf9, 0x7a, 0xeb, 0xa0, 0xcd, 0xd6, 0x41,
	0xaf, 0x5b, 0x07, 0xdd, 0xef, 0x1c, 0x63, 0xb3, 0x73, 0x8c, 0xe7, 0x9d, 0x63, 0x5c, 0x06, 0x5f,
	0xc3, 0xa4, 0x58, 0x08, 0x9a, 0x74, 0xab, 0xed, 0x27, 0xc0, 0x49, 0xb0, 0xec, 0x07, 0xb7, 0x87,
	0x77, 0xa0, 0x92, 0xc5, 0x75, 0xb5, 0xb8, 0xfe, 0x47, 0x00, 0x00, 0x00, 0xff, 0xff, 0x3b, 0xcb,
	0xc4, 0xb5, 0x24, 0x02, 0x00, 0x00,
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
	if !this.BasePool.Equal(that1.BasePool) {
		return false
	}
	if this.PoolRecoveryPeriod != that1.PoolRecoveryPeriod {
		return false
	}
	if !this.MinStabilitySpread.Equal(that1.MinStabilitySpread) {
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
		size := m.MinStabilitySpread.Size()
		i -= size
		if _, err := m.MinStabilitySpread.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintMarket(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if m.PoolRecoveryPeriod != 0 {
		i = encodeVarintMarket(dAtA, i, uint64(m.PoolRecoveryPeriod))
		i--
		dAtA[i] = 0x10
	}
	{
		size := m.BasePool.Size()
		i -= size
		if _, err := m.BasePool.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintMarket(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintMarket(dAtA []byte, offset int, v uint64) int {
	offset -= sovMarket(v)
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
	l = m.BasePool.Size()
	n += 1 + l + sovMarket(uint64(l))
	if m.PoolRecoveryPeriod != 0 {
		n += 1 + sovMarket(uint64(m.PoolRecoveryPeriod))
	}
	l = m.MinStabilitySpread.Size()
	n += 1 + l + sovMarket(uint64(l))
	return n
}

func sovMarket(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMarket(x uint64) (n int) {
	return sovMarket(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMarket
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
				return fmt.Errorf("proto: wrong wireType = %d for field BasePool", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMarket
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
				return ErrInvalidLengthMarket
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMarket
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.BasePool.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PoolRecoveryPeriod", wireType)
			}
			m.PoolRecoveryPeriod = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMarket
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PoolRecoveryPeriod |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinStabilitySpread", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMarket
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
				return ErrInvalidLengthMarket
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthMarket
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MinStabilitySpread.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMarket(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMarket
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
func skipMarket(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMarket
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
					return 0, ErrIntOverflowMarket
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
					return 0, ErrIntOverflowMarket
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
				return 0, ErrInvalidLengthMarket
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMarket
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMarket
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMarket        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMarket          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMarket = fmt.Errorf("proto: unexpected end of group")
)
