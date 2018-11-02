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

package tree

import (
	"fmt"
	"time"

	"github.com/cockroachdb/cockroach/pkg/sql/sem/types"
	"github.com/cockroachdb/cockroach/pkg/util/timeofday"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
)

// presetTypesForTesting is a mapping of qualified names to types that can be mocked out
// for tests to allow the qualified names to be type checked without throwing an error.
var presetTypesForTesting map[string]types.T

// MockNameTypes populates presetTypesForTesting for a test.
func MockNameTypes(types map[string]types.T) func() {
	presetTypesForTesting = types
	return func() {
		presetTypesForTesting = nil
	}
}

// SampleDatum is intended to be a more lightweight version of RandDatum for
// when you just need one consistent example of a datum.
func SampleDatum(t types.T) Datum {
	// TODO(bram): speed this up.
	switch {
	case types.BitArray.Identical(t):
		a, _ := NewDBitArrayFromInt(123, 40)
		return a
	case types.Bool.Identical(t):
		return MakeDBool(true)
	case types.Int.Identical(t):
		return NewDInt(123)
	case types.Float.Identical(t):
		f := DFloat(123.456)
		return &f
	case types.Decimal.Identical(t):
		d := &DDecimal{}
		// int64(rng.Uint64()) to get negative numbers, too
		d.Decimal.SetFinite(3, 6)
		return d
	case types.String.Identical(t):
		return NewDString("Carl")
	case types.Bytes.Identical(t):
		return NewDBytes("Princess")
	case types.Date.Identical(t):
		return NewDDate(123123)
	case types.Time.Identical(t):
		return MakeDTime(timeofday.FromInt(789))
	case types.Timestamp.Identical(t):
		return MakeDTimestamp(timeutil.Unix(123, 123), time.Second)
	case types.TimestampTZ.Identical(t):
		return MakeDTimestampTZ(timeutil.Unix(123, 123), time.Second)
	case types.Interval.Identical(t):
		i, _ := ParseDInterval("1h1m1s")
		return i
	case types.UUID.Identical(t):
		u, _ := ParseDUuidFromString("3189ad07-52f2-4d60-83e8-4a8347fef718")
		return u
	case types.INet.Identical(t):
		i, _ := ParseDIPAddrFromINetString("127.0.0.1")
		return i
	case types.JSON.Identical(t):
		j, _ := ParseDJSON(`{"a": "b"}`)
		return j
	case types.Oid.Identical(t):
		return NewDOid(DInt(1009))
	default:
		panic(fmt.Sprintf("SampleDatum not implemented for %s", t))
	}
}
