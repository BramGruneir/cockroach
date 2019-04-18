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

	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgerror"
	"github.com/cockroachdb/cockroach/pkg/sql/rowcontainer"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
)

type checkerDetails struct {
	checker      fkExistenceCheckForDelete
	updateColMap map[sqlbase.ColumnID]int
	originalRows *rowcontainer.RowContainer
	updatedRows  *rowcontainer.RowContainer
}

// FKChecker stores all foreign key checks so they can be executed on a per
// statement instead of per row basis.
// TODO(bram): Should this move under fkTables or vice versa?
type FKChecker struct {
	deleteChekers 
	deleteCheckers map[TableID]fkExistenceCheckForDelete
	insertColMap   map[TableID]map[sqlbase.ColumnID]int
	deletedRows    map[TableID]*rowcontainer.RowContainer // Rows that have been deleted by Table ID

	insertCheckers map[TableID]fkExistenceCheckForInsert
	insertColMap   map[TableID]map[sqlbase.ColumnID]int
	insertedRows   map[TableID]*rowcontainer.RowContainer // Rows that have been inserted by Table ID

	updateCheckers map[TableID]fkExistenceCheckForUpdate
	updateColMap   map[sqlbase.ColumnID]int
	originalRows   map[TableID]*rowcontainer.RowContainer // Original values for rows that have been updated by Table ID
	updatedRows    map[TableID]*rowcontainer.RowContainer // New values for rows that have been updated by Table ID

	txn      *client.Txn
	fkTables FkTableMetadata
	alloc    *sqlbase.DatumAlloc
	evalCtx  *tree.EvalContext
}

// MakeFKChecker creates a Foreign Key Checker for use by a writer.
func MakeFKChecker(
	txn *client.Txn, evalCtx *tree.EvalContext, fkTables FkTableMetadata, alloc *sqlbase.DatumAlloc,
) *FKChecker {
	return &FKChecker{
		deleteCheckers: make(map[TableID]fkExistenceCheckForDelete),
		deletedRows:    make(map[TableID]*rowcontainer.RowContainer),
		insertCheckers: make(map[TableID]fkExistenceCheckForInsert),
		insertedRows:   make(map[TableID]*rowcontainer.RowContainer),
		updateCheckers: make(map[TableID]fkExistenceCheckForUpdate),
		originalRows:   make(map[TableID]*rowcontainer.RowContainer),
		updatedRows:    make(map[TableID]*rowcontainer.RowContainer),
		txn:            txn,
		fkTables:       fkTables,
		alloc:          alloc,
		evalCtx:        evalCtx,
	}
}

func colMapsEqual(a, b map[sqlbase.ColumnID]int) bool {
	if len(a) != len(b) {
		return false
	}
	for aColumnID, aIndex := range a {
		if bIndex, exists := b[aColumnID]; !exists || bIndex != aIndex {
			return false
		}
	}
	return true
}

func (fk *FKChecker) addDeleteChecker(
	table *sqlbase.ImmutableTableDescriptor, colMap map[sqlbase.ColumnID]int,
) (fkExistenceCheckForDelete, error) {
	if deleteChecker, exists := fk.deleteCheckers[table.ID]; exists {
		if !colMapsEqual(fk.deleteColMap, colMap) {
			panic("WHAAA?")
		}
		return deleteChecker, nil
	}
	deleteChecker, err := makeFkExistenceCheckHelperForDelete(
		fk.txn, table, fk.fkTables, colMap, fk.alloc,
	)
	if err != nil {
		return fkExistenceCheckForDelete{}, err
	}
	fk.deleteCheckers[table.ID] = deleteChecker
	fk.deleteColMap = colMap
	return deleteChecker, nil
}

func (fk *FKChecker) addInsertChecker(
	table *sqlbase.ImmutableTableDescriptor, colMap map[sqlbase.ColumnID]int,
) (fkExistenceCheckForInsert, error) {
	if insertChecker, exists := fk.insertCheckers[table.ID]; exists {
		if !colMapsEqual(fk.insertColMap, colMap) {
			panic("WHAAA?")
		}
		return insertChecker, nil
	}
	insertChecker, err := makeFkExistenceCheckHelperForInsert(
		fk.txn, table, fk.fkTables, colMap, fk.alloc,
	)
	if err != nil {
		return fkExistenceCheckForInsert{}, err
	}
	fk.insertCheckers[table.ID] = insertChecker
	fk.insertColMap = colMap
	return insertChecker, nil
}

func (fk *FKChecker) addUpdateChecker(
	table *sqlbase.ImmutableTableDescriptor, colMap map[sqlbase.ColumnID]int, primaryKeyUpdate bool,
) (fkExistenceCheckForUpdate, error) {
	if updateChecker, exists := fk.updateCheckers[table.ID]; exists {
		if !colMapsEqual(fk.updateColMap, colMap) {
			panic("WHAAA?")
		}
		return updateChecker, nil
	}
	updateChecker, err := makeFkExistenceCheckHelperForUpdate(
		fk.txn, table, fk.fkTables, colMap, fk.alloc,
	)
	if err != nil {
		return fkExistenceCheckForUpdate{}, err
	}
	fk.updateCheckers[table.ID] = updateChecker
	fk.updateColMap = colMap
	return updateChecker, nil
}

func (fk *FKChecker) addDeletedRow(ctx context.Context, tableID TableID, row tree.Datums) error {
	deletedRowContainer, exists := fk.deletedRows[tableID]
	if !exists {
		colTypeInfo, err := sqlbase.MakeColTypeInfo(table, colIDtoRowIndex)
		if err != nil {
			return err
		}
		deletedRowContainer = rowcontainer.NewRowContainer(
			fk.evalCtx.Mon.MakeBoundAccount(), colTypeInfo, 1,
		)
		fk.deletedRows[tableID] = deletedRowContainer
	}
	_, err := deletedRowContainer.AddRow(ctx, row)
	return err
}

func (fk *FKChecker) addInsertedRow(
	ctx context.Context, tableID TableID, colTypeInfo sqlbase.ColTypeInfo, row tree.Datums,
) error {
	insertedRowContainer, exists := fk.insertedRows[tableID]
	if !exists {
		insertedRowContainer = rowcontainer.NewRowContainer(
			fk.evalCtx.Mon.MakeBoundAccount(), colTypeInfo, 1,
		)
		fk.insertedRows[tableID] = insertedRowContainer
	}
	_, err := insertedRowContainer.AddRow(ctx, row)
	return err
}

func (fk *FKChecker) addUpdatedRow(
	ctx context.Context,
	tableID TableID,
	colTypeInfo sqlbase.ColTypeInfo,
	originalRow tree.Datums,
	updatedRow tree.Datums,
) error {
	originalRowContainer, exists := fk.originalRows[tableID]
	if !exists {
		originalRowContainer = rowcontainer.NewRowContainer(
			fk.evalCtx.Mon.MakeBoundAccount(), colTypeInfo, 1,
		)
		fk.originalRows[tableID] = originalRowContainer
	}
	updatedRowContainer, exists := fk.updatedRows[tableID]
	if !exists {
		updatedRowContainer = rowcontainer.NewRowContainer(
			fk.evalCtx.Mon.MakeBoundAccount(), colTypeInfo, 1,
		)
		fk.updatedRows[tableID] = updatedRowContainer
	}
	if _, err := originalRowContainer.AddRow(ctx, originalRow); err != nil {
		return err
	}
	_, err := updatedRowContainer.AddRow(ctx, updatedRow)
	return err
}

// RunChecks executes all the checks in the checker.
func (fk *FKChecker) RunChecks(ctx context.Context, traceKV bool) error {
	for tableID, deletedRows := range fk.deletedRows {
		if deletedRows.Len() == 0 {
			continue
		}
		deleteChecker, exists := fk.deleteCheckers[tableID]
		if !exists {
			return pgerror.NewAssertionErrorf("could not find delete checker for table %d", tableID)
		}
		for deletedRows.Len() > 0 {
			if err := deleteChecker.addAllIdxChecks(ctx, deletedRows.At(0), traceKV); err != nil {
				return err
			}
			if err := deleteChecker.checker.runCheck(ctx, deletedRows.At(0), nil); err != nil {
				return err
			}
			deletedRows.PopFirst()
		}
	}
	return nil
}
