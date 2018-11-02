// Copyright 2017 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package types

import (
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgerror"
	"github.com/lib/pq/oid"
)

var (
	// Oid is the type of an OID.
	Oid T = makeTOid(oid.T_oid)
	// RegClass is the type of an regclass OID variant.
	RegClass T = makeTOid(oid.T_regclass)
	// RegNamespace is the type of an regnamespace OID variant.
	RegNamespace T = makeTOid(oid.T_regnamespace)
	// RegProc is the type of an regproc OID variant.
	RegProc T = makeTOid(oid.T_regproc)
	// RegProcedure is the type of an regprocedure OID variant.
	RegProcedure T = makeTOid(oid.T_regprocedure)
	// RegType is the type of an regtype OID variant.
	RegType T = makeTOid(oid.T_regtype)

	// Name is a type-alias for String with a different OID.
	Name = WrapTypeWithOid(String, oid.T_name)
	// IntVector is a type-alias for an IntArray with a different OID.
	IntVector = WrapTypeWithOid(makeTArray(Int), oid.T_int2vector)
	// OidVector is a type-alias for an OidArray with a different OID.
	OidVector = WrapTypeWithOid(makeTArray(Oid), oid.T_oidvector)
	// NameArray is the type family of a DArray containing the Name alias type.
	NameArray T = makeTArray(Name)
)

var (
	// Unexported wrapper types. These exist for Postgres type compatibility.
	typeInt2    = WrapTypeWithOid(Int, oid.T_int2)
	typeInt4    = WrapTypeWithOid(Int, oid.T_int4)
	typeFloat4  = WrapTypeWithOid(Float, oid.T_float4)
	typeVarChar = WrapTypeWithOid(String, oid.T_varchar)
	typeBpChar  = WrapTypeWithOid(String, oid.T_bpchar)
	typeQChar   = WrapTypeWithOid(String, oid.T_char)
	typeBit     = WrapTypeWithOid(BitArray, oid.T_bit)
)

// OidToType maps Postgres object IDs to CockroachDB types.  We export
// the map instead of a method so that other packages can iterate over
// the map directly.
var OidToType = map[oid.Oid]T{
	oid.T_anyelement:   Any,
	oid.T_anyarray:     makeTArray(Any),
	oid.T_bool:         Bool,
	oid.T__bool:        makeTArray(Bool),
	oid.T_bytea:        Bytes,
	oid.T__bytea:       makeTArray(Bytes),
	oid.T_date:         Date,
	oid.T__date:        makeTArray(Date),
	oid.T_time:         Time,
	oid.T__time:        makeTArray(Time),
	oid.T_float4:       typeFloat4,
	oid.T__float4:      makeTArray(typeFloat4),
	oid.T_float8:       Float,
	oid.T__float8:      makeTArray(Float),
	oid.T_int2:         typeInt2,
	oid.T__int2:        makeTArray(typeInt2),
	oid.T_int4:         typeInt4,
	oid.T__int4:        makeTArray(typeInt4),
	oid.T_int8:         Int,
	oid.T__int8:        makeTArray(Int),
	oid.T_interval:     Interval,
	oid.T__interval:    makeTArray(Interval),
	oid.T_name:         Name,
	oid.T__name:        makeTArray(Name),
	oid.T_numeric:      Decimal,
	oid.T__numeric:     makeTArray(Decimal),
	oid.T_oid:          Oid,
	oid.T__oid:         makeTArray(Oid),
	oid.T_text:         String,
	oid.T__text:        makeTArray(String),
	oid.T_timestamp:    Timestamp,
	oid.T__timestamp:   makeTArray(Timestamp),
	oid.T_timestamptz:  TimestampTZ,
	oid.T__timestamptz: makeTArray(TimestampTZ),
	oid.T_uuid:         UUID,
	oid.T__uuid:        makeTArray(UUID),
	oid.T_inet:         INet,
	oid.T__inet:        makeTArray(INet),
	oid.T_varchar:      typeVarChar,
	oid.T__varchar:     makeTArray(typeVarChar),
	oid.T_bpchar:       typeBpChar,
	oid.T__bpchar:      makeTArray(typeBpChar),
	oid.T_char:         typeQChar,
	oid.T__char:        makeTArray(typeQChar),
	oid.T_varbit:       BitArray,
	oid.T__varbit:      makeTArray(BitArray),
	oid.T_bit:          typeBit,
	oid.T__bit:         makeTArray(typeBit),
	oid.T_jsonb:        JSON,
	oid.T_int2vector:   IntVector,
	oid.T_oidvector:    OidVector,
	oid.T_regclass:     RegClass,
	oid.T_regnamespace: RegNamespace,
	oid.T_regproc:      RegProc,
	oid.T_regprocedure: RegProcedure,
	oid.T_regtype:      RegType,
	// TODO(jordan): I think this entry for T_record is out of place.
	oid.T_record: FamTuple,
}

// oidToArrayOid maps scalar type Oids to their corresponding array type Oid.
var oidToArrayOid = map[oid.Oid]oid.Oid{
	oid.T_anyelement:  oid.T_anyarray,
	oid.T_bit:         oid.T__bit,
	oid.T_bool:        oid.T__bool,
	oid.T_bpchar:      oid.T__bpchar,
	oid.T_bytea:       oid.T__bytea,
	oid.T_char:        oid.T__char,
	oid.T_date:        oid.T__date,
	oid.T_float4:      oid.T__float4,
	oid.T_float8:      oid.T__float8,
	oid.T_inet:        oid.T__inet,
	oid.T_int2:        oid.T__int2,
	oid.T_int4:        oid.T__int4,
	oid.T_int8:        oid.T__int8,
	oid.T_interval:    oid.T__interval,
	oid.T_name:        oid.T__name,
	oid.T_numeric:     oid.T__numeric,
	oid.T_oid:         oid.T__oid,
	oid.T_text:        oid.T__text,
	oid.T_time:        oid.T__time,
	oid.T_timestamp:   oid.T__timestamp,
	oid.T_timestamptz: oid.T__timestamptz,
	oid.T_varbit:      oid.T__varbit,
	oid.T_varchar:     oid.T__varchar,
	oid.T_uuid:        oid.T__uuid,
}

// TOid represents an alias to the Int type with a different Postgres OID.
type TOid struct {
	oidType oid.Oid
	_       [0][]byte // Prevents use of the == operator.
}

func makeTOid(oidType oid.Oid) TOid {
	return TOid{
		oidType: oidType,
	}
}

func (t TOid) String() string { return t.SQLName() }

// Identical implements the T interface.
func (TOid) Identical(other T) bool { _, ok := UnwrapType(other).(TOid); return ok }

// Equivalent implements the T interface.
func (t TOid) Equivalent(other T) bool { return t.Identical(other) || Any.Identical(other) }

// FamilyEqual implements the T interface.
func (t TOid) FamilyEqual(other T) bool { return t.Identical(other) }

// Oid implements the T interface.
func (t TOid) Oid() oid.Oid { return t.oidType }

// SQLName implements the T interface.
func (t TOid) SQLName() string {
	switch t.oidType {
	case oid.T_oid:
		return "oid"
	case oid.T_regclass:
		return "regclass"
	case oid.T_regnamespace:
		return "regnamespace"
	case oid.T_regproc:
		return "regproc"
	case oid.T_regprocedure:
		return "regprocedure"
	case oid.T_regtype:
		return "regtype"
	default:
		panic(fmt.Sprintf("unexpected oidType: %v", t.oidType))
	}
}

// IsAmbiguous implements the T interface.
func (TOid) IsAmbiguous() bool { return false }

// TOidWrapper is a T implementation which is a wrapper around a T, allowing
// custom Oid values to be attached to the T. The T is used by DOidWrapper
// to permit type aliasing with custom Oids without needing to create new typing
// rules or define new Datum types.
type TOidWrapper struct {
	T
	oid oid.Oid
}

var customOidNames = map[oid.Oid]string{
	oid.T_name: "name",
}

func (t TOidWrapper) String() string {
	// Allow custom type names for specific Oids, but default to wrapped String.
	if s, ok := customOidNames[t.oid]; ok {
		return s
	}
	return t.T.String()
}

// Oid implements the T interface.
func (t TOidWrapper) Oid() oid.Oid { return t.oid }

// WrapTypeWithOid wraps a T with a custom Oid.
func WrapTypeWithOid(t T, oid oid.Oid) T {
	switch v := t.(type) {
	case tUnknown, tAny, TOidWrapper:
		panic(pgerror.NewAssertionErrorf("cannot wrap %T with an Oid", v))
	}
	return TOidWrapper{
		T:   t,
		oid: oid,
	}
}

// UnwrapType returns the base T type for a provided type, stripping
// a *TOidWrapper if present. This is useful for cases like type switches,
// where type aliases should be ignored.
func UnwrapType(t T) T {
	if w, ok := t.(TOidWrapper); ok {
		return w.T
	}
	return t
}
