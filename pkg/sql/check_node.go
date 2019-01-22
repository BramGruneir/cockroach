// Copyright 2016 The Cockroach Authors.
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

package sql

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
)

// checkNode implements a constraint check stage.
type checkNode struct {
	source          planDataSource
	colIDToRowIndex map[sqlbase.ColumnID]int
	checkHelper     *sqlbase.CheckHelper
}

func (p *planner) Check() (planNode, error) {
	//**** DO THIS?
}

// Next implements the planNode interface.
func (n *checkNode) Next(params runParams) (bool, error) {
	if next, err := n.source.plan.Next(params); !next || err != nil {
		return false, err
	}
	if err := n.checkHelper.LoadRow(n.colIDToRowIndex, n.Values(), false /* merge */); err != nil {
		return false, err
	}
	err := n.checkHelper.Check(params.EvalContext())
	return err == nil, err
}

func (n *checkNode) Values() tree.Datums       { return n.source.plan.Values() }
func (n *checkNode) Close(ctx context.Context) { n.source.plan.Close(ctx) }
