// Copyright 2019 The Cockroach Authors.
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

package row

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
	"github.com/pkg/errors"
)

// FKChecker ...      *************
type FKChecker struct {
	referenceingIndex sqlbase.IndexDescriptor
	colIDToRowIndex   map[sqlbase.ColumnID]int // for the referencing index
	tableID           ID
	ref               sqlbase.ForeignKeyReference
	tables            TableLookupsByID
	deletedRows       *sqlbase.RowContainer
	addedRows         *sqlbase.RowContainer
}

func makeFKChecker(
	evalCtx *tree.EvalContext,
	referencingIndex sqlbase.IndexDescriptor,
	tableID ID,
	ref sqlbase.ForeignKeyReference,
	tables TableLookupsByID,
) (FKChecker, error) {

	referencingTableLookup, ok := tables[tableID]
	if !ok || referencingTableLookup.Table == nil {
		return FKChecker{}, errors.Errorf("referencing table %d not in provided table map %+v", tableID, tables)
	}
	//TODO(bram): Check is adding?  It's already been checked in TablesNeededForFKs().
	referencingTable := referencingTableLookup.Table

	colIDToRowIndex := make(map[sqlbase.ColumnID]int)
	for i, columnID := range referencingIndex.ColumnIDs {
		colIDToRowIndex[columnID] = i
	}

	colTypeInfo, err := sqlbase.MakeColTypeInfo(referencingTable, colIDToRowIndex)
	if err != nil {
		return FKChecker{}, err
	}

	return FKChecker{
		ref:         ref,
		tables:      tables,
		deletedRows: sqlbase.NewRowContainer(evalCtx.Mon.MakeBoundAccount(), colTypeInfo, 0),
		addedRows:   sqlbase.NewRowContainer(evalCtx.Mon.MakeBoundAccount(), colTypeInfo, 0),
	}, nil
}

// Close must be called when the FKChecker is no longer needed.
func (fk FKChecker) Close(ctx context.Context) {
	fk.deletedRows.Close(ctx)
	fk.addedRows.Close(ctx)
}

func (fk FKChecker) DeleteRow(ctx context.Context, row tree.Datums) error {
	_, err := fk.deletedRows.AddRow(ctx, row)
	return err
}

func (fk FKChecker) InsertRow(ctx context.Context, row tree.Datums) error {
	_, err := fk.addedRows.AddRow(ctx, row)
}

func (fk FKChecker) UpdateRow(
	ctx context.Context, original tree.Datums, updated tree.Datums,
) error {
	if _, err := fk.deletedRows.AddRow(ctx, original); err != nil {
		return err
	}
	_, err := fk.addedRows.AddRow(ctx, updated)
	return err
}
