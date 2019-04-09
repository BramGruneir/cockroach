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
	"github.com/cockroachdb/cockroach/pkg/keys"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
	"github.com/cockroachdb/cockroach/pkg/util/log"
)

// Deleter abstracts the key/value operations for deleting table rows.
type Deleter struct {
	Helper               rowHelper
	FetchCols            []sqlbase.ColumnDescriptor
	FetchColIDtoRowIndex map[sqlbase.ColumnID]int
	FKChecker            *FKChecker
	Fks                  fkExistenceCheckForDelete
	cascader             *cascader
	// For allocation avoidance.
	key roachpb.Key
}

// MakeDeleter creates a Deleter for the given table.
//
// The returned Deleter contains a FetchCols field that defines the
// expectation of which values are passed as values to DeleteRow. Any column
// passed in requestedCols will be included in FetchCols.
func MakeDeleter(
	txn *client.Txn,
	tableDesc *sqlbase.ImmutableTableDescriptor,
	fkTables FkTableMetadata,
	requestedCols []sqlbase.ColumnDescriptor,
	checkFKs checkFKConstraints,
	evalCtx *tree.EvalContext,
	alloc *sqlbase.DatumAlloc,
	fkChecker *FKChecker,
) (Deleter, error) {
	rowDeleter, err := makeRowDeleterWithoutCascader(
		txn, tableDesc, fkTables, requestedCols, checkFKs, alloc, fkChecker,
	)
	if err != nil {
		return Deleter{}, err
	}
	if checkFKs == CheckFKs {
		var err error
		rowDeleter.cascader, err = makeDeleteCascader(
			txn, tableDesc, fkTables, evalCtx, alloc, fkChecker,
		)
		if err != nil {
			return Deleter{}, err
		}
	}
	return rowDeleter, nil
}

// makeRowDeleterWithoutCascader creates a rowDeleter but does not create an
// additional cascader.
func makeRowDeleterWithoutCascader(
	txn *client.Txn,
	tableDesc *sqlbase.ImmutableTableDescriptor,
	fkTables FkTableMetadata,
	requestedCols []sqlbase.ColumnDescriptor,
	checkFKs checkFKConstraints,
	alloc *sqlbase.DatumAlloc,
	fkChecker *FKChecker,
) (Deleter, error) {
	indexes := tableDesc.DeletableIndexes()

	fetchCols := requestedCols[:len(requestedCols):len(requestedCols)]
	fetchColIDtoRowIndex := ColIDtoRowIndexFromCols(fetchCols)

	maybeAddCol := func(colID sqlbase.ColumnID) error {
		if _, ok := fetchColIDtoRowIndex[colID]; !ok {
			col, err := tableDesc.FindColumnByID(colID)
			if err != nil {
				return err
			}
			fetchColIDtoRowIndex[col.ID] = len(fetchCols)
			fetchCols = append(fetchCols, *col)
		}
		return nil
	}
	for _, colID := range tableDesc.PrimaryIndex.ColumnIDs {
		if err := maybeAddCol(colID); err != nil {
			return Deleter{}, err
		}
	}
	for _, index := range indexes {
		for _, colID := range index.ColumnIDs {
			if err := maybeAddCol(colID); err != nil {
				return Deleter{}, err
			}
		}
		// The extra columns are needed to fix #14601.
		for _, colID := range index.ExtraColumnIDs {
			if err := maybeAddCol(colID); err != nil {
				return Deleter{}, err
			}
		}
	}

	rd := Deleter{
		Helper:               newRowHelper(tableDesc, indexes),
		FetchCols:            fetchCols,
		FetchColIDtoRowIndex: fetchColIDtoRowIndex,
		FKChecker:            fkChecker,
	}
	if checkFKs == CheckFKs {
		var err error
		if rd.Fks, err = rd.FKChecker.addDeleteChecker(tableDesc, fetchColIDtoRowIndex); err != nil {
			return Deleter{}, err
		}
	}

	return rd, nil
}

// DeleteRow adds to the batch the kv operations necessary to delete a table row
// with the given values. It also will cascade as required and check for
// orphaned rows. The bytesMonitor is only used if cascading/fk checking and can
// be nil if not.
func (rd *Deleter) DeleteRow(
	ctx context.Context,
	b *client.Batch,
	values []tree.Datum,
	checkFKs checkFKConstraints,
	traceKV bool,
) error {
	primaryIndexKey, secondaryIndexEntries, err := rd.Helper.encodeIndexes(rd.FetchColIDtoRowIndex, values)
	if err != nil {
		return err
	}

	// Delete the row from any secondary indices.
	for i, secondaryIndexEntry := range secondaryIndexEntries {
		if traceKV {
			log.VEventf(ctx, 2, "Del %s", keys.PrettyPrint(rd.Helper.secIndexValDirs[i], secondaryIndexEntry.Key))
		}
		b.Del(&secondaryIndexEntry.Key)
	}

	// Delete the row.
	for i, family := range rd.Helper.TableDesc.Families {
		if i > 0 {
			// HACK: MakeFamilyKey appends to its argument, so on every loop iteration
			// after the first, trim primaryIndexKey so nothing gets overwritten.
			// TODO(dan): Instead of this, use something like engine.ChunkAllocator.
			primaryIndexKey = primaryIndexKey[:len(primaryIndexKey):len(primaryIndexKey)]
		}
		rd.key = keys.MakeFamilyKey(primaryIndexKey, uint32(family.ID))
		if traceKV {
			log.VEventf(ctx, 2, "Del %s", keys.PrettyPrint(rd.Helper.primIndexValDirs, rd.key))
		}
		b.Del(&rd.key)
		rd.key = nil
	}

	if rd.cascader != nil {
		if err := rd.cascader.cascadeAll(
			ctx,
			rd.Helper.TableDesc,
			tree.Datums(values),
			nil, /* updatedValues */
			rd.FetchColIDtoRowIndex,
			traceKV,
		); err != nil {
			return err
		}
	}
	if rd.Fks.checker != nil && checkFKs == CheckFKs {
		if err := rd.Fks.addAllIdxChecks(ctx, values, traceKV); err != nil {
			return err
		}
		return rd.Fks.checker.runCheck(ctx, values, nil)
	}
	return nil
}

// DeleteIndexRow adds to the batch the kv operations necessary to delete a
// table row from the given index.
func (rd *Deleter) DeleteIndexRow(
	ctx context.Context,
	b *client.Batch,
	idx *sqlbase.IndexDescriptor,
	values []tree.Datum,
	traceKV bool,
) error {
	if rd.Fks.checker != nil {
		if err := rd.Fks.addAllIdxChecks(ctx, values, traceKV); err != nil {
			return err
		}
		if err := rd.Fks.checker.runCheck(ctx, values, nil); err != nil {
			return err
		}
	}
	secondaryIndexEntry, err := sqlbase.EncodeSecondaryIndex(
		rd.Helper.TableDesc.TableDesc(), idx, rd.FetchColIDtoRowIndex, values)
	if err != nil {
		return err
	}

	for _, entry := range secondaryIndexEntry {
		if traceKV {
			log.VEventf(ctx, 2, "Del %s", entry.Key)
		}
		b.Del(entry.Key)
	}
	return nil
}
