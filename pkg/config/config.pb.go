// Code generated by protoc-gen-gogo.
// source: cockroach/pkg/config/config.proto
// DO NOT EDIT!

/*
	Package config is a generated protocol buffer package.

	It is generated from these files:
		cockroach/pkg/config/config.proto

	It has these top-level messages:
		GCPolicy
		Constraint
		Constraints
		ZoneConfig
		SystemConfig
*/
package config

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import cockroach_roachpb1 "github.com/cockroachdb/cockroach/pkg/roachpb"
import cockroach_roachpb "github.com/cockroachdb/cockroach/pkg/roachpb"

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

type Constraint_Type int32

const (
	// POSITIVE will attempt to ensure all stores the replicas are on has this
	// constraint.
	Constraint_POSITIVE Constraint_Type = 0
	// REQUIRED is like POSITIVE except replication will fail if not satisfied.
	Constraint_REQUIRED Constraint_Type = 1
	// PROHIBITED will prevent replicas from having this key, value.
	Constraint_PROHIBITED Constraint_Type = 2
)

var Constraint_Type_name = map[int32]string{
	0: "POSITIVE",
	1: "REQUIRED",
	2: "PROHIBITED",
}
var Constraint_Type_value = map[string]int32{
	"POSITIVE":   0,
	"REQUIRED":   1,
	"PROHIBITED": 2,
}

func (x Constraint_Type) Enum() *Constraint_Type {
	p := new(Constraint_Type)
	*p = x
	return p
}
func (x Constraint_Type) String() string {
	return proto.EnumName(Constraint_Type_name, int32(x))
}
func (x *Constraint_Type) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Constraint_Type_value, data, "Constraint_Type")
	if err != nil {
		return err
	}
	*x = Constraint_Type(value)
	return nil
}
func (Constraint_Type) EnumDescriptor() ([]byte, []int) { return fileDescriptorConfig, []int{1, 0} }

// GCPolicy defines garbage collection policies which apply to MVCC
// values within a zone.
//
// TODO(spencer): flesh this out to include maximum number of values
//   as well as whether there's an intersection between max values
//   and TTL or a union.
type GCPolicy struct {
	// TTLSeconds specifies the maximum age of a value before it's
	// garbage collected. Only older versions of values are garbage
	// collected. Specifying <=0 mean older versions are never GC'd.
	TTLSeconds int32 `protobuf:"varint,1,opt,name=ttl_seconds,json=ttlSeconds" json:"ttl_seconds"`
}

func (m *GCPolicy) Reset()                    { *m = GCPolicy{} }
func (m *GCPolicy) String() string            { return proto.CompactTextString(m) }
func (*GCPolicy) ProtoMessage()               {}
func (*GCPolicy) Descriptor() ([]byte, []int) { return fileDescriptorConfig, []int{0} }

// Constraint constrains the stores a replica can be stored on.
type Constraint struct {
	Type Constraint_Type `protobuf:"varint,1,opt,name=type,enum=cockroach.config.Constraint_Type" json:"type"`
	// Key is only set if this is a constraint on locality.
	Key string `protobuf:"bytes,2,opt,name=key" json:"key"`
	// Value to constrain to.
	Value string `protobuf:"bytes,3,opt,name=value" json:"value"`
}

func (m *Constraint) Reset()                    { *m = Constraint{} }
func (*Constraint) ProtoMessage()               {}
func (*Constraint) Descriptor() ([]byte, []int) { return fileDescriptorConfig, []int{1} }

// Constraints is a collection of constraints.
type Constraints struct {
	Constraints []Constraint `protobuf:"bytes,6,rep,name=constraints" json:"constraints"`
}

func (m *Constraints) Reset()                    { *m = Constraints{} }
func (m *Constraints) String() string            { return proto.CompactTextString(m) }
func (*Constraints) ProtoMessage()               {}
func (*Constraints) Descriptor() ([]byte, []int) { return fileDescriptorConfig, []int{2} }

// ZoneConfig holds configuration that is needed for a range of KV pairs. This
// and the conversion methods must stay in sync with ZoneConfigHuman.
type ZoneConfig struct {
	// TODO(d4l3k): Remove replica_attrs after a sufficient amount of time has passed.
	ReplicaAttrs  []cockroach_roachpb.Attributes `protobuf:"bytes,1,rep,name=replica_attrs,json=replicaAttrs" json:"replica_attrs" yaml:",omitempty"`
	RangeMinBytes int64                          `protobuf:"varint,2,opt,name=range_min_bytes,json=rangeMinBytes" json:"range_min_bytes" yaml:"range_min_bytes"`
	RangeMaxBytes int64                          `protobuf:"varint,3,opt,name=range_max_bytes,json=rangeMaxBytes" json:"range_max_bytes" yaml:"range_max_bytes"`
	// If GC policy is not set, uses the next highest, non-null policy
	// in the zone config hierarchy, up to the default policy if necessary.
	GC GCPolicy `protobuf:"bytes,4,opt,name=gc" json:"gc"`
	// NumReplicas specifies the desired number of replicas
	NumReplicas int32 `protobuf:"varint,5,opt,name=num_replicas,json=numReplicas" json:"num_replicas" yaml:"num_replicas"`
	// Constraints constrains which stores the replicas can be stored on. The
	// order in which the constraints are stored is arbitrary and may change.
	// https://github.com/cockroachdb/cockroach/blob/master/docs/RFCS/expressive_zone_config.md#constraint-system
	Constraints Constraints `protobuf:"bytes,6,opt,name=constraints" json:"constraints" yaml:"constraints,flow"`
}

func (m *ZoneConfig) Reset()                    { *m = ZoneConfig{} }
func (m *ZoneConfig) String() string            { return proto.CompactTextString(m) }
func (*ZoneConfig) ProtoMessage()               {}
func (*ZoneConfig) Descriptor() ([]byte, []int) { return fileDescriptorConfig, []int{3} }

type SystemConfig struct {
	Values []cockroach_roachpb1.KeyValue `protobuf:"bytes,1,rep,name=values" json:"values"`
}

func (m *SystemConfig) Reset()                    { *m = SystemConfig{} }
func (m *SystemConfig) String() string            { return proto.CompactTextString(m) }
func (*SystemConfig) ProtoMessage()               {}
func (*SystemConfig) Descriptor() ([]byte, []int) { return fileDescriptorConfig, []int{4} }

func init() {
	proto.RegisterType((*GCPolicy)(nil), "cockroach.config.GCPolicy")
	proto.RegisterType((*Constraint)(nil), "cockroach.config.Constraint")
	proto.RegisterType((*Constraints)(nil), "cockroach.config.Constraints")
	proto.RegisterType((*ZoneConfig)(nil), "cockroach.config.ZoneConfig")
	proto.RegisterType((*SystemConfig)(nil), "cockroach.config.SystemConfig")
	proto.RegisterEnum("cockroach.config.Constraint_Type", Constraint_Type_name, Constraint_Type_value)
}
func (m *GCPolicy) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GCPolicy) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	dAtA[i] = 0x8
	i++
	i = encodeVarintConfig(dAtA, i, uint64(m.TTLSeconds))
	return i, nil
}

func (m *Constraint) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Constraint) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	dAtA[i] = 0x8
	i++
	i = encodeVarintConfig(dAtA, i, uint64(m.Type))
	dAtA[i] = 0x12
	i++
	i = encodeVarintConfig(dAtA, i, uint64(len(m.Key)))
	i += copy(dAtA[i:], m.Key)
	dAtA[i] = 0x1a
	i++
	i = encodeVarintConfig(dAtA, i, uint64(len(m.Value)))
	i += copy(dAtA[i:], m.Value)
	return i, nil
}

func (m *Constraints) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Constraints) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Constraints) > 0 {
		for _, msg := range m.Constraints {
			dAtA[i] = 0x32
			i++
			i = encodeVarintConfig(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func (m *ZoneConfig) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ZoneConfig) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.ReplicaAttrs) > 0 {
		for _, msg := range m.ReplicaAttrs {
			dAtA[i] = 0xa
			i++
			i = encodeVarintConfig(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	dAtA[i] = 0x10
	i++
	i = encodeVarintConfig(dAtA, i, uint64(m.RangeMinBytes))
	dAtA[i] = 0x18
	i++
	i = encodeVarintConfig(dAtA, i, uint64(m.RangeMaxBytes))
	dAtA[i] = 0x22
	i++
	i = encodeVarintConfig(dAtA, i, uint64(m.GC.Size()))
	n1, err := m.GC.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n1
	dAtA[i] = 0x28
	i++
	i = encodeVarintConfig(dAtA, i, uint64(m.NumReplicas))
	dAtA[i] = 0x32
	i++
	i = encodeVarintConfig(dAtA, i, uint64(m.Constraints.Size()))
	n2, err := m.Constraints.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n2
	return i, nil
}

func (m *SystemConfig) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SystemConfig) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Values) > 0 {
		for _, msg := range m.Values {
			dAtA[i] = 0xa
			i++
			i = encodeVarintConfig(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func encodeFixed64Config(dAtA []byte, offset int, v uint64) int {
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
func encodeFixed32Config(dAtA []byte, offset int, v uint32) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintConfig(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *GCPolicy) Size() (n int) {
	var l int
	_ = l
	n += 1 + sovConfig(uint64(m.TTLSeconds))
	return n
}

func (m *Constraint) Size() (n int) {
	var l int
	_ = l
	n += 1 + sovConfig(uint64(m.Type))
	l = len(m.Key)
	n += 1 + l + sovConfig(uint64(l))
	l = len(m.Value)
	n += 1 + l + sovConfig(uint64(l))
	return n
}

func (m *Constraints) Size() (n int) {
	var l int
	_ = l
	if len(m.Constraints) > 0 {
		for _, e := range m.Constraints {
			l = e.Size()
			n += 1 + l + sovConfig(uint64(l))
		}
	}
	return n
}

func (m *ZoneConfig) Size() (n int) {
	var l int
	_ = l
	if len(m.ReplicaAttrs) > 0 {
		for _, e := range m.ReplicaAttrs {
			l = e.Size()
			n += 1 + l + sovConfig(uint64(l))
		}
	}
	n += 1 + sovConfig(uint64(m.RangeMinBytes))
	n += 1 + sovConfig(uint64(m.RangeMaxBytes))
	l = m.GC.Size()
	n += 1 + l + sovConfig(uint64(l))
	n += 1 + sovConfig(uint64(m.NumReplicas))
	l = m.Constraints.Size()
	n += 1 + l + sovConfig(uint64(l))
	return n
}

func (m *SystemConfig) Size() (n int) {
	var l int
	_ = l
	if len(m.Values) > 0 {
		for _, e := range m.Values {
			l = e.Size()
			n += 1 + l + sovConfig(uint64(l))
		}
	}
	return n
}

func sovConfig(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozConfig(x uint64) (n int) {
	return sovConfig(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GCPolicy) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowConfig
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
			return fmt.Errorf("proto: GCPolicy: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GCPolicy: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TTLSeconds", wireType)
			}
			m.TTLSeconds = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TTLSeconds |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipConfig(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthConfig
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
func (m *Constraint) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowConfig
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
			return fmt.Errorf("proto: Constraint: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Constraint: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= (Constraint_Type(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Key", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthConfig
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Key = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthConfig
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipConfig(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthConfig
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
func (m *Constraints) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowConfig
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
			return fmt.Errorf("proto: Constraints: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Constraints: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Constraints", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthConfig
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Constraints = append(m.Constraints, Constraint{})
			if err := m.Constraints[len(m.Constraints)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipConfig(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthConfig
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
func (m *ZoneConfig) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowConfig
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
			return fmt.Errorf("proto: ZoneConfig: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ZoneConfig: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReplicaAttrs", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthConfig
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ReplicaAttrs = append(m.ReplicaAttrs, cockroach_roachpb.Attributes{})
			if err := m.ReplicaAttrs[len(m.ReplicaAttrs)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RangeMinBytes", wireType)
			}
			m.RangeMinBytes = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RangeMinBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RangeMaxBytes", wireType)
			}
			m.RangeMaxBytes = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RangeMaxBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field GC", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthConfig
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.GC.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NumReplicas", wireType)
			}
			m.NumReplicas = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NumReplicas |= (int32(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Constraints", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthConfig
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Constraints.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipConfig(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthConfig
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
func (m *SystemConfig) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowConfig
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
			return fmt.Errorf("proto: SystemConfig: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SystemConfig: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Values", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowConfig
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthConfig
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Values = append(m.Values, cockroach_roachpb1.KeyValue{})
			if err := m.Values[len(m.Values)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipConfig(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthConfig
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
func skipConfig(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowConfig
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
					return 0, ErrIntOverflowConfig
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
					return 0, ErrIntOverflowConfig
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
				return 0, ErrInvalidLengthConfig
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowConfig
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
				next, err := skipConfig(dAtA[start:])
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
	ErrInvalidLengthConfig = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowConfig   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("cockroach/pkg/config/config.proto", fileDescriptorConfig) }

var fileDescriptorConfig = []byte{
	// 585 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x92, 0xc1, 0x6e, 0xd3, 0x4c,
	0x10, 0xc7, 0xb3, 0x71, 0x5a, 0xf5, 0x1b, 0xa7, 0xfd, 0xc2, 0x82, 0x8a, 0x95, 0x82, 0xe3, 0x5a,
	0x1c, 0x72, 0xa8, 0x1c, 0x29, 0x48, 0x48, 0x14, 0x09, 0x84, 0xd3, 0x50, 0x2c, 0x40, 0x2d, 0x4e,
	0xe8, 0xa1, 0x17, 0xb3, 0x75, 0xb7, 0xc6, 0xaa, 0xed, 0xb5, 0xec, 0x0d, 0xd4, 0x6f, 0xc1, 0x91,
	0x23, 0x6f, 0xc1, 0x2b, 0xf4, 0xc8, 0x0d, 0x4e, 0x15, 0x84, 0x37, 0xe0, 0x09, 0x90, 0xed, 0x4d,
	0xeb, 0xa6, 0x55, 0x4f, 0x9e, 0x99, 0xfd, 0xcf, 0xcf, 0xbb, 0xf3, 0x1f, 0x58, 0x77, 0x99, 0x7b,
	0x9c, 0x30, 0xe2, 0x7e, 0xe8, 0xc5, 0xc7, 0x5e, 0xcf, 0x65, 0xd1, 0x91, 0x3f, 0xfb, 0x18, 0x71,
	0xc2, 0x38, 0xc3, 0xad, 0x73, 0x89, 0x51, 0xd6, 0xdb, 0xda, 0xe5, 0xa6, 0x22, 0x8a, 0x0f, 0x7a,
	0x87, 0x84, 0x93, 0xb2, 0xa7, 0xfd, 0xe0, 0x7a, 0x45, 0x48, 0x39, 0xa9, 0xa8, 0xee, 0x78, 0xcc,
	0x63, 0x45, 0xd8, 0xcb, 0xa3, 0xb2, 0xaa, 0x3f, 0x83, 0xa5, 0xed, 0xc1, 0x2e, 0x0b, 0x7c, 0x37,
	0xc3, 0x0f, 0x41, 0xe6, 0x3c, 0x70, 0x52, 0xea, 0xb2, 0xe8, 0x30, 0x55, 0x90, 0x86, 0xba, 0x0b,
	0x26, 0x3e, 0x3d, 0xeb, 0xd4, 0xa6, 0x67, 0x1d, 0x18, 0x8f, 0x5f, 0x8f, 0xca, 0x13, 0x1b, 0x38,
	0x0f, 0x44, 0xac, 0x7f, 0x43, 0x00, 0x03, 0x16, 0xa5, 0x3c, 0x21, 0x7e, 0xc4, 0xf1, 0x13, 0x68,
	0xf0, 0x2c, 0xa6, 0x45, 0xf3, 0x4a, 0x7f, 0xdd, 0x98, 0x7f, 0x8e, 0x71, 0xa1, 0x35, 0xc6, 0x59,
	0x4c, 0xcd, 0x46, 0xce, 0xb7, 0x8b, 0x26, 0xbc, 0x0a, 0xd2, 0x31, 0xcd, 0x94, 0xba, 0x86, 0xba,
	0xff, 0x89, 0x83, 0xbc, 0x80, 0xdb, 0xb0, 0xf0, 0x91, 0x04, 0x13, 0xaa, 0x48, 0x95, 0x93, 0xb2,
	0xa4, 0xf7, 0xa1, 0x91, 0x73, 0x70, 0x13, 0x96, 0x76, 0x77, 0x46, 0xd6, 0xd8, 0xda, 0x1b, 0xb6,
	0x6a, 0x79, 0x66, 0x0f, 0xdf, 0xbe, 0xb3, 0xec, 0xe1, 0x56, 0x0b, 0xe1, 0x15, 0x80, 0x5d, 0x7b,
	0xe7, 0xa5, 0x65, 0x5a, 0xe3, 0xe1, 0x56, 0xab, 0xbe, 0xd9, 0xf8, 0xf2, 0xb5, 0x53, 0xd3, 0x47,
	0x20, 0x5f, 0x5c, 0x26, 0xc5, 0x5b, 0x20, 0xbb, 0x17, 0xa9, 0xb2, 0xa8, 0x49, 0x5d, 0xb9, 0x7f,
	0xef, 0xa6, 0x07, 0x88, 0x8b, 0x54, 0xdb, 0xf4, 0x1f, 0x12, 0xc0, 0x3e, 0x8b, 0xe8, 0xa0, 0x10,
	0x63, 0x07, 0x96, 0x13, 0x1a, 0x07, 0xbe, 0x4b, 0x1c, 0xc2, 0x79, 0x92, 0x0f, 0x35, 0xc7, 0xde,
	0xaf, 0x60, 0x85, 0x5d, 0xc6, 0x73, 0xce, 0x13, 0xff, 0x60, 0xc2, 0x69, 0x6a, 0xae, 0xe5, 0xdc,
	0xbf, 0x67, 0x9d, 0x5b, 0x19, 0x09, 0x83, 0x4d, 0x7d, 0x83, 0x85, 0x3e, 0xa7, 0x61, 0xcc, 0x33,
	0x5d, 0x41, 0x76, 0x53, 0x00, 0x73, 0x7d, 0x8a, 0x5f, 0xc0, 0xff, 0x09, 0x89, 0x3c, 0xea, 0x84,
	0x7e, 0xe4, 0x1c, 0x64, 0x9c, 0xa6, 0xc5, 0xf8, 0x24, 0x53, 0x15, 0x8c, 0xd5, 0x92, 0x31, 0x27,
	0xd2, 0xed, 0xe5, 0xa2, 0xf2, 0xc6, 0x8f, 0xcc, 0x3c, 0xaf, 0x70, 0xc8, 0x89, 0xe0, 0x48, 0x37,
	0x70, 0x66, 0xa2, 0x73, 0x0e, 0x39, 0x29, 0x39, 0x8f, 0xa0, 0xee, 0xb9, 0x4a, 0x43, 0x43, 0x5d,
	0xb9, 0xdf, 0xbe, 0x3a, 0xbc, 0xd9, 0xae, 0x99, 0x20, 0xd6, 0xaa, 0xbe, 0x3d, 0xb0, 0xeb, 0x9e,
	0x8b, 0x9f, 0x42, 0x33, 0x9a, 0x84, 0x8e, 0x78, 0x5b, 0xaa, 0x2c, 0x14, 0xcb, 0x37, 0x1b, 0xc4,
	0xed, 0xf2, 0xe7, 0x55, 0x85, 0x6e, 0xcb, 0xd1, 0x24, 0xb4, 0x45, 0x86, 0xdf, 0xcf, 0xbb, 0x87,
	0xe6, 0xc6, 0x7c, 0xc5, 0xbd, 0xd4, 0xec, 0x08, 0xfa, 0xdd, 0x92, 0x5e, 0xe9, 0xdf, 0x38, 0x0a,
	0xd8, 0x27, 0xfd, 0xb2, 0xb3, 0x16, 0x34, 0x47, 0x59, 0xca, 0x69, 0x28, 0xac, 0x7d, 0x0c, 0x8b,
	0xc5, 0x06, 0xce, 0x3c, 0x5d, 0xbb, 0xc6, 0xd3, 0x57, 0x34, 0xdb, 0xcb, 0x35, 0x62, 0x53, 0x44,
	0x83, 0xa9, 0x9d, 0xfe, 0x56, 0x6b, 0xa7, 0x53, 0x15, 0x7d, 0x9f, 0xaa, 0xe8, 0xe7, 0x54, 0x45,
	0xbf, 0xa6, 0x2a, 0xfa, 0xfc, 0x47, 0xad, 0xed, 0x2f, 0x96, 0xd7, 0xfc, 0x17, 0x00, 0x00, 0xff,
	0xff, 0x2d, 0x87, 0x71, 0x8f, 0x2a, 0x04, 0x00, 0x00,
}
