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

package sqlbase

import (
	"github.com/cockroachdb/cockroach/pkg/util/mon"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/util"
	"github.com/cockroachdb/cockroach/pkg/util/log"
)

// cascader is used to handle all referential integrity cascading actions.
type cascader struct {
	txn                *client.Txn
	tablesByID         TableLookupsByID                   // TablesDescriptors by Table ID
	indexRowFetchers   map[ID]map[IndexID]MultiRowFetcher // RowFetchers by Table ID and Index ID
	rowDeleters        map[ID]RowDeleter                  // RowDeleters by Table ID
	deleterRowFetchers map[ID]MultiRowFetcher             // RowFetchers for rowDeleters by Table ID
	rowsToCheck        map[ID]*RowContainer               // Rows that have been deleted by Table ID
	alloc              *DatumAlloc
}

// The row container here is reused, so store the range of index values that
// are queued in this particular case.
type cascadeQueueElement struct {
	table           *TableDescriptor
	values          *RowContainer
	colIDtoRowIndex map[ColumnID]int
	startIndex      int // Index to the first row that's queued.
	totalRows       int // Total number of rows queued.
}

// cascadeQueue is used for a breadth first walk of the referential integrity
// graph.
type cascadeQueue struct {
	// ********* fix this, remove queue
	queue []cascadeQueueElement
}

func (q *cascadeQueue) enqueue(
	ctx context.Context,
	table *TableDescriptor,
	values *RowContainer,
	colIDtoRowIndex map[ColumnID]int,
	startIndex int,
) {
	(*q).queue = append((*q).queue, cascadeQueueElement{
		table:           table,
		values:          values,
		colIDtoRowIndex: colIDtoRowIndex,
		startIndex:      startIndex,
		totalRows:       values.Len() - startIndex,
	})
}

func (q *cascadeQueue) dequeue() (cascadeQueueElement, bool) {
	if len((*q).queue) == 0 {
		return cascadeQueueElement{}, false
	}
	elem := ((*q).queue)[0]
	(*q).queue = ((*q).queue)[1:]
	return elem, true
}

func makeCascader(txn *client.Txn, tablesByID TableLookupsByID, alloc *DatumAlloc) *cascader {
	return &cascader{
		txn:                txn,
		tablesByID:         tablesByID,
		indexRowFetchers:   make(map[ID]map[IndexID]MultiRowFetcher),
		rowDeleters:        make(map[ID]RowDeleter),
		deleterRowFetchers: make(map[ID]MultiRowFetcher),
		rowsToCheck:        make(map[ID]*RowContainer),
		alloc:              alloc,
	}
}

// spanForIndexValues creates a span against an index to extract the primary
// keys needed for cascading.
func spanForIndexValues(
	table *TableDescriptor,
	index *IndexDescriptor,
	prefixLen int,
	indexColIDs map[ColumnID]int,
	values []tree.Datum,
	keyPrefix []byte,
) (roachpb.Span, error) {
	keyBytes, _, err := EncodePartialIndexKey(
		table, index, prefixLen, indexColIDs, values, keyPrefix)
	if err != nil {
		return roachpb.Span{}, err
	}
	key := roachpb.Key(keyBytes)
	return roachpb.Span{Key: key, EndKey: key.PrefixEnd()}, nil
}

// batchRequestForIndexValues creates a batch request against an index to
// extract the primary keys needed for cascading.
func batchRequestForIndexValues(
	ctx context.Context,
	referencedIndex *IndexDescriptor,
	referencingTable *TableDescriptor,
	referencingIndex *IndexDescriptor,
	values *RowContainer,
	colIDtoRowIndex map[ColumnID]int,
) (roachpb.BatchRequest, error) {

	//TODO(bram): consider caching some of these values
	keyPrefix := MakeIndexKeyPrefix(referencingTable, referencingIndex.ID)
	prefixLen := len(referencingIndex.ColumnIDs)
	if len(referencedIndex.ColumnIDs) < prefixLen {
		prefixLen = len(referencedIndex.ColumnIDs)
	}
	indexColIDs := make(map[ColumnID]int, len(referencedIndex.ColumnIDs))
	for i, referencedColID := range referencedIndex.ColumnIDs[:prefixLen] {
		if found, ok := colIDtoRowIndex[referencedColID]; ok {
			indexColIDs[referencingIndex.ColumnIDs[i]] = found
		} else {
			return roachpb.BatchRequest{}, errors.Errorf(
				"missing value for column %q in multi-part foreign key", referencedIndex.ColumnNames[i],
			)
		}
	}

	var req roachpb.BatchRequest
	for i := 0; i < values.Len(); i++ {
		log.Warningf(ctx, "********* values at 0:%s", values.At(i))
		span, err := spanForIndexValues(
			referencingTable, referencingIndex, prefixLen, indexColIDs, values.At(i), keyPrefix,
		)
		if err != nil {
			return roachpb.BatchRequest{}, err
		}
		req.Add(&roachpb.ScanRequest{Span: span})
	}
	return req, nil
}

// spanForPKValues creates a span against the primary index of a table and is
// used to fetch rows for cascading.
func spanForPKValues(
	table *TableDescriptor, fetchColIDtoRowIndex map[ColumnID]int, values tree.Datums,
) (roachpb.Span, error) {
	keyBytes, _, err := EncodePartialIndexKey(
		table,
		&table.PrimaryIndex,
		len(table.PrimaryIndex.ColumnIDs),
		fetchColIDtoRowIndex,
		values,
		MakeIndexKeyPrefix(table, table.PrimaryIndex.ID),
	)
	if err != nil {
		return roachpb.Span{}, err
	}
	key := roachpb.Key(keyBytes)
	return roachpb.Span{Key: key, EndKey: key.PrefixEnd()}, nil
}

// batchRequestForPKValues creates a batch request against the primary index of
// a table and is used to fetch rows for cascading.
func batchRequestForPKValues(
	table *TableDescriptor, fetchColIDtoRowIndex map[ColumnID]int, values []tree.Datums,
) (roachpb.BatchRequest, error) {
	var req roachpb.BatchRequest
	for _, value := range values {
		span, err := spanForPKValues(table, fetchColIDtoRowIndex, value)
		if err != nil {
			return roachpb.BatchRequest{}, err
		}
		req.Add(&roachpb.ScanRequest{Span: span})
	}
	return req, nil
}

func (c *cascader) close(ctx context.Context) {
	for _, rows := range c.rowsToCheck {
		rows.Close(ctx)
	}
}

// addIndexRowFetch will create or load a cached row fetcher on an index to
// fetch the primary keys of the rows that will be affected by a cascading
// action.
func (c *cascader) addIndexRowFetcher(
	table *TableDescriptor, index *IndexDescriptor,
) (MultiRowFetcher, error) {
	// Is there a cached row fetcher?
	rowFetchersForTable, exists := c.indexRowFetchers[table.ID]
	if exists {
		rowFetcher, exists := rowFetchersForTable[index.ID]
		if exists {
			return rowFetcher, nil
		}
	} else {
		c.indexRowFetchers[table.ID] = make(map[IndexID]MultiRowFetcher)
	}

	// Create a new row fetcher. Only the primary key columns are required.
	var colDesc []ColumnDescriptor
	for _, id := range table.PrimaryIndex.ColumnIDs {
		cDesc, err := table.FindColumnByID(id)
		if err != nil {
			return MultiRowFetcher{}, err
		}
		colDesc = append(colDesc, *cDesc)
	}
	var valNeededForCol util.FastIntSet
	valNeededForCol.AddRange(0, len(colDesc)-1)
	isSecondary := table.PrimaryIndex.ID != index.ID
	var rowFetcher MultiRowFetcher
	if err := rowFetcher.Init(
		false, /* reverse */
		false, /* returnRangeInfo */
		false, /* isCheck */
		c.alloc,
		MultiRowFetcherTableArgs{
			Desc:             table,
			Index:            index,
			ColIdxMap:        ColIDtoRowIndexFromCols(colDesc),
			IsSecondaryIndex: isSecondary,
			Cols:             colDesc,
			ValNeededForCol:  valNeededForCol,
		},
	); err != nil {
		return MultiRowFetcher{}, err
	}
	// Cache the row fetcher.
	c.indexRowFetchers[table.ID][index.ID] = rowFetcher
	return rowFetcher, nil
}

// addRowDeleter creates the row deleter and primary index row fetcher.
func (c *cascader) addRowDeleter(table *TableDescriptor) (RowDeleter, MultiRowFetcher, error) {
	// Is there a cached row fetcher and deleter?
	if rowDeleter, exists := c.rowDeleters[table.ID]; exists {
		return rowDeleter, c.deleterRowFetchers[table.ID], nil
	}

	// Create the row deleter. The row deleter is needed prior to the row fetcher
	// as it will dictate what columns are required in the row fetcher.
	rowDeleter, err := MakeRowDeleter(
		c.txn,
		table,
		c.tablesByID,
		nil,  /* requestedCol */
		true, /* checkFKs */
		c.alloc,
	)
	if err != nil {
		return RowDeleter{}, MultiRowFetcher{}, err
	}

	// Create the row fetcher that will retrive the rows and columns needed for
	// deletion.
	var valNeededForCol util.FastIntSet
	valNeededForCol.AddRange(0, len(rowDeleter.FetchCols)-1)
	tableArgs := MultiRowFetcherTableArgs{
		Desc:             table,
		Index:            &table.PrimaryIndex,
		ColIdxMap:        rowDeleter.FetchColIDtoRowIndex,
		IsSecondaryIndex: false,
		Cols:             rowDeleter.FetchCols,
		ValNeededForCol:  valNeededForCol,
	}
	var rowFetcher MultiRowFetcher
	if err := rowFetcher.Init(
		false, /* reverse */
		false, /* returnRangeInfo */
		false, /* isCheck */
		c.alloc,
		tableArgs,
	); err != nil {
		return RowDeleter{}, MultiRowFetcher{}, err
	}

	// Cache both the fetcher and deleter.
	c.rowDeleters[table.ID] = rowDeleter
	c.deleterRowFetchers[table.ID] = rowFetcher
	return rowDeleter, rowFetcher, nil
}

// deleteRow performs row deletions on a single table for all rows that match
// the values. Returns the row container and the start index into the row
// container to show all the values that were deleted. This deletion happens in
// a single batch.
func (c *cascader) deleteRow(
	ctx context.Context,
	referencedIndex *IndexDescriptor,
	referencingTable *TableDescriptor,
	referencingIndex *IndexDescriptor,
	elem cascadeQueueElement,
	mon *mon.BytesMonitor,
	traceKV bool,
) (*RowContainer, map[ColumnID]int, int, error) {
	// Create the span to search for index values.
	// TODO(bram): This initial index lookup can be skipped if the index is the
	// primary index.
	if traceKV {
		log.VEventf(ctx, 2,
			"cascading delete from redIndex:%s, into table:%s, using index:%s for values:%+v",
			referencedIndex.Name, referencingTable.Name, referencingIndex.Name, values.chunks,
		)
	}
	req, err := batchRequestForIndexValues(
		ctx, referencedIndex, referencingTable, referencingIndex, values, colIDtoRowIndex,
	)
	if err != nil {
		return nil, nil, 0, err
	}
	log.Warningf(ctx, "*********** req:%+v", req)
	br, roachErr := c.txn.Send(ctx, req)
	if roachErr != nil {
		return nil, nil, 0, roachErr.GoError()
	}

	log.Warningf(ctx, "*********** br:%+v", br)

	// Create or retrieve the index row fetcher.
	indexRowFetcher, err := c.addIndexRowFetcher(referencingTable, referencingIndex)
	if err != nil {
		return nil, nil, 0, err
	}

	// Fetch all the primary keys that need to be deleted.
	// TODO(Bram): use a row container here and consider chunking this into n,
	// primary keys, perhaps 100, at time.
	var primaryKeysToDel []tree.Datums
	for _, resp := range br.Responses {
		fetcher := spanKVFetcher{
			kvs: resp.GetInner().(*roachpb.ScanResponse).Rows,
		}
		if err := indexRowFetcher.StartScanFrom(ctx, &fetcher); err != nil {
			return nil, nil, 0, err
		}
		for !indexRowFetcher.kvEnd {
			primaryKey, _, _, err := indexRowFetcher.NextRowDecoded(ctx)
			if err != nil {
				return nil, nil, 0, err
			}
			// Make a copy of the primary key because the datum struct is reused in
			// the row fetcher.
			primaryKey = append(tree.Datums(nil), primaryKey...)
			primaryKeysToDel = append(primaryKeysToDel, primaryKey)
		}
	}
	log.Warningf(ctx, "********** pks:%s", primaryKeysToDel)

	// Early exit if no rows need to be deleted.
	if len(primaryKeysToDel) == 0 {
		return nil, nil, 0, nil
	}

	// Create or retrieve the row deleter and primary index row fetcher.
	rowDeleter, pkRowFetcher, err := c.addRowDeleter(referencingTable)
	if err != nil {
		return nil, nil, 0, err
	}

	// Create a batch request to get all the spans of the primary keys that need
	// to be deleted.
	pkLookupReq, err := batchRequestForPKValues(
		referencingTable, rowDeleter.FetchColIDtoRowIndex, primaryKeysToDel,
	)
	if err != nil {
		return nil, nil, 0, err
	}
	pkResp, roachErr := c.txn.Send(ctx, pkLookupReq)
	if roachErr != nil {
		return nil, nil, 0, roachErr.GoError()
	}

	// Fetch the rows for deletion.
	// Add the values to be checked for consistency after all cascading changes
	// have finished and store how many new rows have been added.
	if _, exists := c.rowsToCheck[referencingTable.ID]; !exists {
		colTypeInfo, err := makeColTypeInfo(referencingTable, rowDeleter.FetchColIDtoRowIndex)
		if err != nil {
			return nil, nil, 0, err
		}
		c.rowsToCheck[referencingTable.ID] = NewRowContainer(mon.MakeBoundAccount(), colTypeInfo, len(primaryKeysToDel))
	}
	deletedRowsContainer := c.rowsToCheck[referencingTable.ID]
	startIndex := deletedRowsContainer.Len()

	// Delete the rows in a new batch.
	deleteBatch := c.txn.NewBatch()
	for _, resp := range pkResp.Responses {
		fetcher := spanKVFetcher{
			kvs: resp.GetInner().(*roachpb.ScanResponse).Rows,
		}
		if err := pkRowFetcher.StartScanFrom(ctx, &fetcher); err != nil {
			return nil, nil, 0, err
		}
		for !pkRowFetcher.kvEnd {
			rowToDelete, _, _, err := pkRowFetcher.NextRowDecoded(ctx)
			if err != nil {
				return nil, nil, 0, err
			}
			if _, err := deletedRowsContainer.AddRow(ctx, rowToDelete); err != nil {
				return nil, nil, 0, err
			}
			if err := rowDeleter.deleteRowNoCascade(ctx, deleteBatch, rowToDelete, traceKV); err != nil {
				return nil, nil, 0, err
			}
		}
	}
	if err := c.txn.Run(ctx, deleteBatch); err != nil {
		return nil, nil, 0, err
	}

	return deletedRowsContainer, rowDeleter.FetchColIDtoRowIndex, startIndex, nil
}

// cascadeAll performs all required cascading operations, then checks all the
// remaining indexes to ensure that no orphans were created.
func (c *cascader) cascadeAll(
	ctx context.Context,
	table *TableDescriptor,
	originalValues tree.Datums,
	colIDtoRowIndex map[ColumnID]int,
	mon *mon.BytesMonitor,
	traceKV bool,
) error {
	defer c.close(ctx)
	// Perform all the required cascading operations.
	cascadeQ := cascadeQueue{}

	// Create a row container for the passed in values.
	colTypeInfo, err := makeColTypeInfo(table, colIDtoRowIndex)
	if err != nil {
		return err
	}
	rowContainer := NewRowContainer(mon.MakeBoundAccount(), colTypeInfo, len(originalValues))
	if _, err := rowContainer.AddRow(ctx, originalValues); err != nil {
		return err
	}

	// Add the new row container to the cascader.
	c.rowsToCheck[table.ID] = rowContainer

	// Enqueue the passed in values to start the queue.
	cascadeQ.enqueue(ctx, table, rowContainer, colIDtoRowIndex, 0)

	for {
		select {
		case <-ctx.Done():
			return NewQueryCanceledError()
		default:
		}
		elem, exists := cascadeQ.dequeue()
		if !exists {
			break
		}
		log.Warningf(ctx, ">>>>>>>>> Dequeue:%s", elem.values.At(0))
		if traceKV {
			log.VEventf(ctx, 2, "cascading into %s for values:%s", elem.table.Name, elem.values)
		}
		for _, referencedIndex := range elem.table.AllNonDropIndexes() {
			for _, ref := range referencedIndex.ReferencedBy {
				log.Warningf(ctx, ">>>>>>>>> ref index:%d", ref.Index)
				referencingTable, ok := c.tablesByID[ref.Table]
				if !ok {
					return errors.Errorf("Could not find table:%d in table descriptor map", ref.Table)
				}
				log.Warningf(ctx, ">>>>>>>>> refing table:%s", referencingTable.Table.Name)
				if referencingTable.IsAdding {
					// We can assume that a table being added but not yet public is empty,
					// and thus does not need to be checked for cascading.
					continue
				}
				referencingIndex, err := referencingTable.Table.FindIndexByID(ref.Index)
				if err != nil {
					return err
				}
				log.Warningf(ctx, ">>>>>>>>> refing index:%d-%s", referencingIndex.ID, referencingIndex.Name)
				if referencingIndex.ForeignKey.OnDelete == ForeignKeyReference_CASCADE {
					log.Warningf(ctx, ">>>>>>>>> cascading on delete")
					returnedRowContainer, colIDtoRowIndex, rowStartIndex, err := c.deleteRow(
						ctx, &referencedIndex, referencingTable.Table, referencingIndex, elem.values, elem.colIDtoRowIndex, mon, traceKV,
					)
					if err != nil {
						return err
					}
					if returnedRowContainer != nil && returnedRowContainer.Len() > rowStartIndex {
						// If a row was deleted, add the table to the queue.
						cascadeQ.enqueue(ctx, referencingTable.Table, returnedRowContainer, colIDtoRowIndex, rowStartIndex)
					}
				}
			}
		}
	}

	// Check all values to ensure there are no orphans.
	for tableID, removedValues := range c.rowsToCheck {
		rowDeleter, exists := c.rowDeleters[tableID]
		if !exists {
			return errors.Errorf("Could not find row deleter for table %d", tableID)
		}
		for i := 0; i < removedValues.Len(); i++ {
			if err := rowDeleter.Fks.checkAll(ctx, removedValues.At(i)); err != nil {
				return err
			}
		}
		removedValues.Clear(ctx)
	}

	return nil
}
