// Code generated by protoc-gen-gogo.
// source: cockroach/pkg/util/hlc/timestamp.proto
// DO NOT EDIT!

/*
	Package hlc is a generated protocol buffer package.

	It is generated from these files:
		cockroach/pkg/util/hlc/timestamp.proto

	It has these top-level messages:
		Timestamp
*/
package hlc

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// Timestamp represents a state of the hybrid logical clock.
type Timestamp struct {
	// Holds a wall time, typically a unix epoch time
	// expressed in nanoseconds.
	WallTime int64 `protobuf:"varint,1,opt,name=wall_time,json=wallTime" json:"wall_time"`
	// The logical component captures causality for events whose wall
	// times are equal. It is effectively bounded by (maximum clock
	// skew)/(minimal ns between events) and nearly impossible to
	// overflow.
	Logical int32 `protobuf:"varint,2,opt,name=logical" json:"logical"`
}

func (m *Timestamp) Reset()                    { *m = Timestamp{} }
func (*Timestamp) ProtoMessage()               {}
func (*Timestamp) Descriptor() ([]byte, []int) { return fileDescriptorTimestamp, []int{0} }

func init() {
	proto.RegisterType((*Timestamp)(nil), "cockroach.util.hlc.Timestamp")
}
func (m *Timestamp) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Timestamp) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	dAtA[i] = 0x8
	i++
	i = encodeVarintTimestamp(dAtA, i, uint64(m.WallTime))
	dAtA[i] = 0x10
	i++
	i = encodeVarintTimestamp(dAtA, i, uint64(m.Logical))
	return i, nil
}

func encodeFixed64Timestamp(dAtA []byte, offset int, v uint64) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	dAtA[offset+4] = uint8(v >> 32)
	dAtA[offset+5] = uint8(v >> 40)
	dAtA[offset+6] = uint8(v >> 48)
	dAtA[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Timestamp(dAtA []byte, offset int, v uint32) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintTimestamp(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func NewPopulatedTimestamp(r randyTimestamp, easy bool) *Timestamp {
	this := &Timestamp{}
	this.WallTime = int64(r.Int63())
	if r.Intn(2) == 0 {
		this.WallTime *= -1
	}
	this.Logical = int32(r.Int31())
	if r.Intn(2) == 0 {
		this.Logical *= -1
	}
	if !easy && r.Intn(10) != 0 {
	}
	return this
}

type randyTimestamp interface {
	Float32() float32
	Float64() float64
	Int63() int64
	Int31() int32
	Uint32() uint32
	Intn(n int) int
}

func randUTF8RuneTimestamp(r randyTimestamp) rune {
	ru := r.Intn(62)
	if ru < 10 {
		return rune(ru + 48)
	} else if ru < 36 {
		return rune(ru + 55)
	}
	return rune(ru + 61)
}
func randStringTimestamp(r randyTimestamp) string {
	v1 := r.Intn(100)
	tmps := make([]rune, v1)
	for i := 0; i < v1; i++ {
		tmps[i] = randUTF8RuneTimestamp(r)
	}
	return string(tmps)
}
func randUnrecognizedTimestamp(r randyTimestamp, maxFieldNumber int) (dAtA []byte) {
	l := r.Intn(5)
	for i := 0; i < l; i++ {
		wire := r.Intn(4)
		if wire == 3 {
			wire = 5
		}
		fieldNumber := maxFieldNumber + r.Intn(100)
		dAtA = randFieldTimestamp(dAtA, r, fieldNumber, wire)
	}
	return dAtA
}
func randFieldTimestamp(dAtA []byte, r randyTimestamp, fieldNumber int, wire int) []byte {
	key := uint32(fieldNumber)<<3 | uint32(wire)
	switch wire {
	case 0:
		dAtA = encodeVarintPopulateTimestamp(dAtA, uint64(key))
		v2 := r.Int63()
		if r.Intn(2) == 0 {
			v2 *= -1
		}
		dAtA = encodeVarintPopulateTimestamp(dAtA, uint64(v2))
	case 1:
		dAtA = encodeVarintPopulateTimestamp(dAtA, uint64(key))
		dAtA = append(dAtA, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	case 2:
		dAtA = encodeVarintPopulateTimestamp(dAtA, uint64(key))
		ll := r.Intn(100)
		dAtA = encodeVarintPopulateTimestamp(dAtA, uint64(ll))
		for j := 0; j < ll; j++ {
			dAtA = append(dAtA, byte(r.Intn(256)))
		}
	default:
		dAtA = encodeVarintPopulateTimestamp(dAtA, uint64(key))
		dAtA = append(dAtA, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	}
	return dAtA
}
func encodeVarintPopulateTimestamp(dAtA []byte, v uint64) []byte {
	for v >= 1<<7 {
		dAtA = append(dAtA, uint8(uint64(v)&0x7f|0x80))
		v >>= 7
	}
	dAtA = append(dAtA, uint8(v))
	return dAtA
}
func (m *Timestamp) Size() (n int) {
	var l int
	_ = l
	n += 1 + sovTimestamp(uint64(m.WallTime))
	n += 1 + sovTimestamp(uint64(m.Logical))
	return n
}

func sovTimestamp(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozTimestamp(x uint64) (n int) {
	return sovTimestamp(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Timestamp) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTimestamp
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Timestamp: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Timestamp: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field WallTime", wireType)
			}
			m.WallTime = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTimestamp
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.WallTime |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Logical", wireType)
			}
			m.Logical = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTimestamp
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Logical |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipTimestamp(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthTimestamp
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
func skipTimestamp(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTimestamp
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
					return 0, ErrIntOverflowTimestamp
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTimestamp
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
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthTimestamp
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowTimestamp
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipTimestamp(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthTimestamp = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTimestamp   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("cockroach/pkg/util/hlc/timestamp.proto", fileDescriptorTimestamp) }

var fileDescriptorTimestamp = []byte{
	// 188 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x52, 0x4b, 0xce, 0x4f, 0xce,
	0x2e, 0xca, 0x4f, 0x4c, 0xce, 0xd0, 0x2f, 0xc8, 0x4e, 0xd7, 0x2f, 0x2d, 0xc9, 0xcc, 0xd1, 0xcf,
	0xc8, 0x49, 0xd6, 0x2f, 0xc9, 0xcc, 0x4d, 0x2d, 0x2e, 0x49, 0xcc, 0x2d, 0xd0, 0x2b, 0x28, 0xca,
	0x2f, 0xc9, 0x17, 0x12, 0x82, 0xab, 0xd3, 0x03, 0xa9, 0xd1, 0xcb, 0xc8, 0x49, 0x96, 0x12, 0x49,
	0xcf, 0x4f, 0xcf, 0x07, 0x4b, 0xeb, 0x83, 0x58, 0x10, 0x95, 0x4a, 0x11, 0x5c, 0x9c, 0x21, 0x30,
	0xcd, 0x42, 0x8a, 0x5c, 0x9c, 0xe5, 0x89, 0x39, 0x39, 0xf1, 0x20, 0xe3, 0x24, 0x18, 0x15, 0x18,
	0x35, 0x98, 0x9d, 0x58, 0x4e, 0xdc, 0x93, 0x67, 0x08, 0xe2, 0x00, 0x09, 0x83, 0xd4, 0x09, 0xc9,
	0x71, 0xb1, 0xe7, 0xe4, 0xa7, 0x67, 0x26, 0x27, 0xe6, 0x48, 0x30, 0x29, 0x30, 0x6a, 0xb0, 0x42,
	0x15, 0xc0, 0x04, 0xad, 0x38, 0x66, 0x2c, 0x90, 0x67, 0xd8, 0xb1, 0x40, 0x9e, 0xd1, 0x49, 0xf6,
	0xc4, 0x43, 0x39, 0x86, 0x13, 0x8f, 0xe4, 0x18, 0x2f, 0x3c, 0x92, 0x63, 0xbc, 0xf1, 0x48, 0x8e,
	0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x09, 0x8f, 0xe5, 0x18, 0xa2, 0x98, 0x33, 0x72, 0x92, 0x01, 0x01,
	0x00, 0x00, 0xff, 0xff, 0x4d, 0x10, 0x25, 0xee, 0xcb, 0x00, 0x00, 0x00,
}
