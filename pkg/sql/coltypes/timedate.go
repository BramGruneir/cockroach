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

package coltypes

import (
	"strings"

	"github.com/cockroachdb/cockroach/pkg/sql/lex"
)

// TDate represents a DATE type.
type TDate struct{}

// Format implements the ColTypeFormatter interface.
func (node *TDate) Format(sb *strings.Builder, f lex.EncodeFlags) {
	sb.WriteString("DATE")
}

// TTime represents a TIME type.
type TTime struct{}

// Format implements the ColTypeFormatter interface.
func (node *TTime) Format(sb *strings.Builder, f lex.EncodeFlags) {
	sb.WriteString("TIME")
}

// TTimeTZ represents a TIMETZ type.
type TTimeTZ struct{}

// Format implements the ColTypeFormatter interface.
func (node *TTimeTZ) Format(sb *strings.Builder, f lex.EncodeFlags) {
	sb.WriteString("TIME WITH TIME ZONE")
}

// TTimestamp represents a TIMESTAMP type.
type TTimestamp struct{}

// Format implements the ColTypeFormatter interface.
func (node *TTimestamp) Format(sb *strings.Builder, f lex.EncodeFlags) {
	sb.WriteString("TIMESTAMP")
}

// TTimestampTZ represents a TIMESTAMP type.
type TTimestampTZ struct{}

// Format implements the ColTypeFormatter interface.
func (node *TTimestampTZ) Format(sb *strings.Builder, f lex.EncodeFlags) {
	sb.WriteString("TIMESTAMP WITH TIME ZONE")
}

// TInterval represents an INTERVAL type
type TInterval struct{}

// Format implements the ColTypeFormatter interface.
func (node *TInterval) Format(sb *strings.Builder, f lex.EncodeFlags) {
	sb.WriteString("INTERVAL")
}
