// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in licenses/BSD-vitess.txt.

// Portions of this file are additionally subject to the following
// license and copyright.
//
// Copyright 2015 The Cockroach Authors.
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

// This code was derived from https://github.com/youtube/vitess.

package lex

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cockroachdb/cockroach/pkg/util/stringencoding"
)

var mustQuoteMap = map[byte]bool{
	' ': true,
	',': true,
	'{': true,
	'}': true,
}

// EncodeFlags influence the formatting of strings and identifiers.
type EncodeFlags int

// HasFlags tests whether the given flags are set.
func (f EncodeFlags) HasFlags(subset EncodeFlags) bool {
	return f&subset == subset
}

const (
	// EncNoFlags indicates nothing special should happen while encoding.
	EncNoFlags EncodeFlags = 0

	// EncBareStrings indicates that strings will be rendered without
	// wrapping quotes if they contain no special characters.
	EncBareStrings EncodeFlags = 1 << iota

	// EncBareIdentifiers indicates that identifiers will be rendered
	// without wrapping quotes.
	EncBareIdentifiers

	// EncFirstFreeFlagBit needs to remain unused; it is used as base
	// bit offset for tree.FmtFlags.
	EncFirstFreeFlagBit
)

// EncodeSQLString writes a string literal to the builder. All unicode and
// non-printable characters are escaped.
func EncodeSQLString(sb *strings.Builder, in string) {
	EncodeSQLStringWithFlags(sb, in, EncNoFlags)
}

// EscapeSQLString returns an escaped SQL representation of the given
// string. This is suitable for safely producing a SQL string valid
// for input to the parser.
func EscapeSQLString(in string) string {
	var sb strings.Builder
	EncodeSQLString(&sb, in)
	return sb.String()
}

// HexEncodeString writes a hexadecimal representation of the string
// to the builder.
func HexEncodeString(sb *strings.Builder, in string) {
	for i := 0; i < len(in); i++ {
		sb.Write(stringencoding.RawHexMap[in[i]])
	}
}

// EncodeSQLStringWithFlags writes a string literal to the builder. All
// unicode and non-printable characters are escaped. flags controls
// the output format: if encodeBareString is set, the output string
// will not be wrapped in quotes if the strings contains no special
// characters.
func EncodeSQLStringWithFlags(sb *strings.Builder, in string, flags EncodeFlags) {
	// See http://www.postgresql.org/docs/9.4/static/sql-syntax-lexical.html
	start := 0
	escapedString := false
	bareStrings := flags.HasFlags(EncBareStrings)
	// Loop through each unicode code point.
	for i, r := range in {
		ch := byte(r)
		if r >= 0x20 && r < 0x7F {
			if mustQuoteMap[ch] {
				// We have to quote this string - ignore bareStrings setting
				bareStrings = false
			}
			if !stringencoding.NeedEscape(ch) && ch != '\'' {
				continue
			}
		}

		if !escapedString {
			sb.WriteString("e'") // begin e'xxx' string
			escapedString = true
		}
		sb.WriteString(in[start:i])
		ln := utf8.RuneLen(r)
		if ln < 0 {
			start = i + 1
		} else {
			start = i + ln
		}
		stringencoding.EncodeEscapedChar(sb, in, r, ch, i, '\'')
	}

	quote := !escapedString && !bareStrings
	if quote {
		sb.WriteByte('\'') // begin 'xxx' string if nothing was escaped
	}
	sb.WriteString(in[start:])
	if escapedString || quote {
		sb.WriteByte('\'')
	}
}

// EncodeSQLStringInsideArray writes a string literal to sb using the "string
// within array" formatting.
func EncodeSQLStringInsideArray(sb *strings.Builder, in string) {
	sb.WriteByte('"')
	// Loop through each unicode code point.
	for i, r := range in {
		ch := byte(r)
		if unicode.IsPrint(r) && !stringencoding.NeedEscape(ch) && ch != '"' {
			// Character is printable doesn't need escaping - just print it out.
			sb.WriteRune(r)
		} else {
			stringencoding.EncodeEscapedChar(sb, in, r, ch, i, '"')
		}
	}

	sb.WriteByte('"')
}

// EncodeUnrestrictedSQLIdent writes the identifier in s to sb.
// The identifier is only quoted if the flags don't tell otherwise and
// the identifier contains special characters.
func EncodeUnrestrictedSQLIdent(sb *strings.Builder, s string, flags EncodeFlags) {
	if flags.HasFlags(EncBareIdentifiers) || isBareIdentifier(s) {
		sb.WriteString(s)
		return
	}
	encodeEscapedSQLIdent(sb, s)
}

// EncodeRestrictedSQLIdent writes the identifier in s to the builder. The
// identifier is quoted if either the flags ask for it, the identifier
// contains special characters, or the identifier is a reserved SQL
// keyword.
func EncodeRestrictedSQLIdent(sb *strings.Builder, s string, flags EncodeFlags) {
	if flags.HasFlags(EncBareIdentifiers) || (!isReservedKeyword(s) && isBareIdentifier(s)) {
		sb.WriteString(s)
		return
	}
	encodeEscapedSQLIdent(sb, s)
}

// encodeEscapedSQLIdent writes the identifier in s to the builder. The
// identifier is always quoted. Double quotes inside the identifier
// are escaped.
func encodeEscapedSQLIdent(sb *strings.Builder, s string) {
	sb.WriteByte('"')
	start := 0
	for i, n := 0, len(s); i < n; i++ {
		ch := s[i]
		// The only character that requires escaping is a double quote.
		if ch == '"' {
			if start != i {
				sb.WriteString(s[start:i])
			}
			start = i + 1
			sb.WriteByte(ch)
			sb.WriteByte(ch) // add extra copy of ch
		}
	}
	if start < len(s) {
		sb.WriteString(s[start:])
	}
	sb.WriteByte('"')
}

// EncodeSQLBytes encodes the SQL byte array in 'in' to the builder.
func EncodeSQLBytes(sb *strings.Builder, in string) {
	start := 0
	sb.WriteString("b'")
	// Loop over the bytes of the string (i.e., don't use range over unicode
	// code points).
	for i, n := 0, len(in); i < n; i++ {
		ch := in[i]
		if encodedChar := stringencoding.EncodeMap[ch]; encodedChar != stringencoding.DontEscape {
			sb.WriteString(in[start:i])
			sb.WriteByte('\\')
			sb.WriteByte(encodedChar)
			start = i + 1
		} else if ch == '\'' {
			// We can't just fold this into stringencoding.EncodeMap because stringencoding.EncodeMap is also used for strings which aren't quoted with single-quotes
			sb.WriteString(in[start:i])
			sb.WriteByte('\\')
			sb.WriteByte(ch)
			start = i + 1
		} else if ch < 0x20 || ch >= 0x7F {
			sb.WriteString(in[start:i])
			// Escape non-printable characters.
			sb.Write(stringencoding.HexMap[ch])
			start = i + 1
		}
	}
	sb.WriteString(in[start:])
	sb.WriteByte('\'')
}
