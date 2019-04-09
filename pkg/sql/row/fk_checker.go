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
	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/sql/rowcontainer"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
)

// FKChecker stores all foreign key checks so they can be executed on a per
// statement instead of per row basis.
type FKChecker struct {
	deleteCheckers map[TableID]fkExistenceCheckForDelete
	deletedRows    map[TableID]*rowcontainer.RowContainer // Rows that have been deleted by Table ID

	insertCheckers map[TableID]fkExistenceCheckForInsert
	insertedRows   map[TableID]*rowcontainer.RowContainer // Rows that have been inserted by Table ID

	updateCheckers map[TableID]fkExistenceCheckForUpdate
	originalRows   map[TableID]*rowcontainer.RowContainer // Original values for rows that have been updated by Table ID
	updatedRows    map[TableID]*rowcontainer.RowContainer // New values for rows that have been updated by Table ID

	txn      *client.Txn
	fkTables FkTableMetadata
	alloc    *sqlbase.DatumAlloc
}

func MakeFKChecker(
	txn *client.Txn, fkTables FkTableMetadata, alloc *sqlbase.DatumAlloc,
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
	}
}

func (fk *FKChecker) addDeleteChecker(
	txn *client.Txn,
	table *sqlbase.ImmutableTableDescriptor,
	fkTables FkTableMetadata,
	colMap map[sqlbase.ColumnID]int,
	alloc *sqlbase.DatumAlloc,
) (fkExistenceCheckForDelete, error) {
	if deleteChecker, exists := fk.deleteCheckers[table.ID]; exists {
		return deleteChecker, nil
	}
	deleteChecker, err := makeFkExistenceCheckHelperForDelete(txn, table, fkTables, colMap, alloc)
	fk.deleteCheckers[table.ID] = deleteChecker
	if err != nil {
		return fkExistenceCheckForDelete{}, err
	}
	return deleteChecker, nil
}

func (fk *FKChecker) addInsertChecker(
	txn *client.Txn,
	table *sqlbase.ImmutableTableDescriptor,
	fkTables FkTableMetadata,
	colMap map[sqlbase.ColumnID]int,
	alloc *sqlbase.DatumAlloc,
) (fkExistenceCheckForInsert, error) {
	if insertChecker, exists := fk.insertCheckers[table.ID]; exists {
		return insertChecker, nil
	}
	insertChecker, err := makeFkExistenceCheckHelperForInsert(txn, table, fkTables, colMap, alloc)
	if err != nil {
		return fkExistenceCheckForInsert{}, err
	}
	fk.insertCheckers[table.ID] = insertChecker
	return insertChecker, nil
}

func (fk *FKChecker) addUpdateChecker(
	txn *client.Txn,
	table *sqlbase.ImmutableTableDescriptor,
	fkTables FkTableMetadata,
	colMap map[sqlbase.ColumnID]int,
	alloc *sqlbase.DatumAlloc,
) (fkExistenceCheckForUpdate, error) {
	if updateChecker, exists := fk.updateCheckers[table.ID]; exists {
		return updateChecker, nil
	}
	updateChecker, err := makeFkExistenceCheckHelperForUpdate(txn, table, fkTables, colMap, alloc)
	if err != nil {
		return fkExistenceCheckForUpdate{}, err
	}
	fk.updateCheckers[table.ID] = updateChecker
	return updateChecker, nil
}
