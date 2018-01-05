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

package sqlbase

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgerror"
	"github.com/cockroachdb/cockroach/pkg/sql/privilege"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/util/log"
)

// TableLookupsByID maps table IDs to looked up descriptors or, for tables that
// exist but are not yet public/leasable, entries with just the IsAdding flag.
type TableLookupsByID map[ID]TableLookup

// TableLookup is the value type of TableLookupsByID: An optional table
// descriptor, populated when the table is public/leasable, and an IsAdding
// flag.
type TableLookup struct {
	Table    *TableDescriptor
	IsAdding bool
}

// TableLookupFunction is the function type used by TablesNeededForFKs that will
// perform the actual lookup.
type TableLookupFunction func(context.Context, ID) (TableLookup, error)

// NoLookup can be used to not perform any lookups during a TablesNeededForFKs
// function call.
func NoLookup(_ context.Context, _ ID) (TableLookup, error) {
	return TableLookup{}, nil
}

// CheckPrivilegeFunction is the function type used by TablesNeededForFKs that will
// check the privileges of the current user to access specific tables.
type CheckPrivilegeFunction func(DescriptorProto, privilege.Kind) error

// NoCheckPrivilege can be used to not perform any privilege checks during a
// TablesNeededForFKs function call.
func NoCheckPrivilege(_ DescriptorProto, _ privilege.Kind) error {
	return nil
}

// FKCheck indicates a kind of FK check (delete, insert, or both).
type FKCheck int

const (
	// CheckDeletes checks if rows reference a changed value.
	CheckDeletes = iota
	// CheckInserts checks if a new value references an existing row.
	CheckInserts
	// CheckUpdates checks all references (CheckDeletes+CheckInserts).
	CheckUpdates
)

type tableLookupQueueElement struct {
	tableLookup TableLookup
	usage       FKCheck
}

type tableLookupQueue struct {
	queue          []tableLookupQueueElement
	alreadyChecked map[ID]map[FKCheck]struct{}
	tableLookups   TableLookupsByID
	lookup         TableLookupFunction
	checkPrivilege CheckPrivilegeFunction
}

func (q *tableLookupQueue) getTable(ctx context.Context, tableID ID) (TableLookup, error) {
	if tableLookup, exists := q.tableLookups[tableID]; exists {
		return tableLookup, nil
	}
	tableLookup, err := q.lookup(ctx, tableID)
	if err != nil {
		return TableLookup{}, err
	}
	if !tableLookup.IsAdding && tableLookup.Table != nil {
		if err := q.checkPrivilege(tableLookup.Table, privilege.SELECT); err != nil {
			return TableLookup{}, err
		}
	}
	q.tableLookups[tableID] = tableLookup
	return tableLookup, nil
}

func (q *tableLookupQueue) enqueue(ctx context.Context, tableID ID, usage FKCheck) error {
	// Lookup the table.
	tableLookup, err := q.getTable(ctx, tableID)
	if err != nil {
		return err
	}
	// Don't enqueue if lookup returns an empty tableLookup. This just means that
	// there is no need to walk any further.
	if tableLookup.Table == nil {
		return nil
	}
	// Only enqueue checks that haven't been performed yet.
	if alreadyCheckByTableID, exists := q.alreadyChecked[tableID]; exists {
		if _, existsInner := alreadyCheckByTableID[usage]; existsInner {
			return nil
		}
	} else {
		q.alreadyChecked[tableID] = make(map[FKCheck]struct{})
	}
	q.alreadyChecked[tableID][usage] = struct{}{}
	// If the table is being added, there's no need to check it.
	if tableLookup.IsAdding {
		return nil
	}
	switch usage {
	// Insert has already been checked when the table is fetched.
	case CheckDeletes:
		if err := q.checkPrivilege(tableLookup.Table, privilege.DELETE); err != nil {
			return err
		}
	case CheckUpdates:
		if err := q.checkPrivilege(tableLookup.Table, privilege.UPDATE); err != nil {
			return err
		}
	}
	(*q).queue = append((*q).queue, tableLookupQueueElement{tableLookup: tableLookup, usage: usage})
	return nil
}

func (q *tableLookupQueue) dequeue() (TableLookup, FKCheck, bool) {
	if len((*q).queue) == 0 {
		return TableLookup{}, 0, false
	}
	elem := (*q).queue[0]
	(*q).queue = (*q).queue[1:]
	return elem.tableLookup, elem.usage, true
}

// TablesNeededForFKs populates a map of TableLookupsByID for all the
// TableDescriptors that might be needed when performing FK checking for delete
// and/or insert operations. It uses the passed in lookup function to perform
// the actual lookup.
func TablesNeededForFKs(
	ctx context.Context,
	table TableDescriptor,
	usage FKCheck,
	lookup TableLookupFunction,
	checkPrivilege CheckPrivilegeFunction,
) (TableLookupsByID, error) {
	queue := tableLookupQueue{
		tableLookups:   make(TableLookupsByID),
		alreadyChecked: make(map[ID]map[FKCheck]struct{}),
		lookup:         lookup,
		checkPrivilege: checkPrivilege,
	}
	// Add the passed in table descriptor to the table lookup.
	queue.tableLookups[table.ID] = TableLookup{Table: &table}
	if err := queue.enqueue(ctx, table.ID, usage); err != nil {
		return nil, err
	}
	for {
		tableLookup, curUsage, exists := queue.dequeue()
		if !exists {
			return queue.tableLookups, nil
		}
		// If the table descriptor is nil it means that there was no actual lookup
		// performed. Meaning there is no need to walk any secondary relationships
		// and the table descriptor lookup will happen later.
		if tableLookup.IsAdding || tableLookup.Table == nil {
			continue
		}
		for _, idx := range tableLookup.Table.AllNonDropIndexes() {
			if curUsage == CheckInserts || curUsage == CheckUpdates {
				if idx.ForeignKey.IsSet() {
					if _, err := queue.getTable(ctx, idx.ForeignKey.Table); err != nil {
						return nil, err
					}
				}
			}
			if curUsage == CheckDeletes || curUsage == CheckUpdates {
				for _, ref := range idx.ReferencedBy {
					// The table being referenced is required to know the relationship, so
					// fetch it here.
					referencedTableLookup, err := queue.getTable(ctx, ref.Table)
					if err != nil {
						return nil, err
					}
					// Again here if the table descriptor is nil it means that there was
					// no actual lookup performed. Meaning there is no need to walk any
					// secondary relationships.
					if referencedTableLookup.IsAdding || referencedTableLookup.Table == nil {
						continue
					}
					referencedIdx, err := referencedTableLookup.Table.FindIndexByID(ref.Index)
					if err != nil {
						return nil, err
					}
					if curUsage == CheckDeletes {
						var nextUsage FKCheck
						switch referencedIdx.ForeignKey.OnDelete {
						case ForeignKeyReference_CASCADE:
							nextUsage = CheckDeletes
						case ForeignKeyReference_SET_DEFAULT, ForeignKeyReference_SET_NULL:
							nextUsage = CheckUpdates
						default:
							// There is no need to check any other relationships.
							continue
						}
						if err := queue.enqueue(ctx, referencedTableLookup.Table.ID, nextUsage); err != nil {
							return nil, err
						}
					} else {
						// curUsage == CheckUpdates
						if referencedIdx.ForeignKey.OnUpdate == ForeignKeyReference_CASCADE ||
							referencedIdx.ForeignKey.OnUpdate == ForeignKeyReference_SET_DEFAULT ||
							referencedIdx.ForeignKey.OnUpdate == ForeignKeyReference_SET_NULL {
							if err := queue.enqueue(
								ctx, referencedTableLookup.Table.ID, CheckUpdates,
							); err != nil {
								return nil, err
							}
						}
					}
				}
			}
		}
	}
}

// spanKVFetcher is an kvFetcher that returns a set slice of kvs.
type spanKVFetcher struct {
	kvs []roachpb.KeyValue
}

// nextKV implements the kvFetcher interface.
func (f *spanKVFetcher) nextKV(ctx context.Context) (bool, roachpb.KeyValue, error) {
	if len(f.kvs) == 0 {
		return false, roachpb.KeyValue{}, nil
	}
	kv := f.kvs[0]
	f.kvs = f.kvs[1:]
	return true, kv, nil
}

// getRangesInfo implements the kvFetcher interface.
func (f *spanKVFetcher) getRangesInfo() []roachpb.RangeInfo {
	panic("getRangesInfo() called on spanKVFetcher")
}

// fkBatchChecker accumulates foreign key checks and sends them out as a single
// kv batch on demand. Checks are accumulated in order - the first failing check
// will be the one that produces an error report.
type fkBatchChecker struct {
	batch roachpb.BatchRequest
	// batchIdxToFk maps the index of the check request/response in the kv batch
	// to the baseFKHelper that created it.
	batchIdxToFk []*baseFKHelper
	txn          *client.Txn

	// indexChecks is used to accumulate index checks that will be added when
	// runCheck is called.
	// TODO(bram): Consider turning these into maps to not repeat.
	insertIndexChecks []*baseFKHelper
	deleteIndexChecks []*baseFKHelper
}

func (f *fkBatchChecker) reset() {
	f.batch.Reset()
	f.batchIdxToFk = f.batchIdxToFk[:0]
	f.insertIndexChecks = f.insertIndexChecks[:0]
	f.deleteIndexChecks = f.deleteIndexChecks[:0]
}

func (f *fkBatchChecker) addInsertIndexChecks(
	fksHelpers []baseFKHelper,
) {
	for i := range fksHelpers {
		f.insertIndexChecks = append(f.insertIndexChecks, &fksHelpers[i])
	}
}

func (f *fkBatchChecker) addDeleteIndexChecks(
	fksHelpers []baseFKHelper,
) {
	for i := range fksHelpers {
		f.deleteIndexChecks = append(f.deleteIndexChecks, &fksHelpers[i])
	}
}

// addIndexCheck adds all the indexChecks based off a list baseFKHelpers. This
// should generally be used internally to the checker to add both insert and
// delete index checks.
func (f *fkBatchChecker) addIndexChecks(
	ctx context.Context, row tree.Datums, indexChecks []*baseFKHelper,
) error {
	if len(indexChecks) > 0 && row == nil {
		return pgerror.NewError(pgerror.CodeInternalError,
			"programming error: adding in check with nil data",
		)
	}
	log.Warningf(ctx, "********** %+v %s", indexChecks, row)
	for _, fk := range indexChecks {
		if fk.searchTable.Name == "customer reviews" {
			continue
		}
		log.Warningf(ctx, "********** %s %s %s", fk.searchTable.Name, fk.searchIdx.Name, row)
		nulls := true
		for _, colID := range fk.searchIdx.ColumnIDs[:fk.prefixLen] {
			found, ok := fk.ids[colID]
			if !ok {
				panic(fmt.Sprintf("fk ids (%v) missing column id %d", fk.ids, colID))
			}
			if row[found] != tree.DNull {
				nulls = false
				break
			}
		}
		if nulls {
			continue
		}
		if err := f.addCheck(row, fk); err != nil {
			return err
		}
	}
	return nil
}

// addCheck adds a check for the given row and baseFKHelper to the batch.
func (f *fkBatchChecker) addCheck(row tree.Datums, source *baseFKHelper) error {
	span, err := source.spanForValues(row)
	if err != nil {
		return err
	}
	r := roachpb.RequestUnion{}
	scan := roachpb.ScanRequest{Span: span}
	r.MustSetInner(&scan)
	f.batch.Requests = append(f.batch.Requests, r)
	f.batchIdxToFk = append(f.batchIdxToFk, source)
	log.Warningf(context.TODO(), "*********** %s", scan.Key)
	return nil
}

// runCheck sends the accumulated batch of foreign key checks to kv, given the
// old and new values of the row being modified. Either oldRow or newRow can
// be set to nil in the case of an insert or a delete, respectively.
// A pgerror.CodeForeignKeyViolationError is returned if a foreign key violation
// is detected, corresponding to the first foreign key that was violated in
// order of addition.
func (f *fkBatchChecker) runCheck(
	ctx context.Context, oldRow tree.Datums, newRow tree.Datums,
) error {
	log.Warningf(ctx, "********* running checks %s -> %s", oldRow, newRow)
	// Add in all outstanding index checks.
	if err := f.addIndexChecks(ctx, oldRow, f.deleteIndexChecks); err != nil {
		return err
	}
	if err := f.addIndexChecks(ctx, newRow, f.insertIndexChecks); err != nil {
		return err
	}

	if len(f.batch.Requests) == 0 {
		return nil
	}
	defer f.reset()

	br, err := f.txn.Send(ctx, f.batch)
	if err != nil {
		return err.GoError()
	}

	fetcher := spanKVFetcher{}
	for i, resp := range br.Responses {
		fk := f.batchIdxToFk[i]
		fetcher.kvs = resp.GetInner().(*roachpb.ScanResponse).Rows
		if err := fk.rf.StartScanFrom(ctx, &fetcher); err != nil {
			return err
		}
		switch fk.dir {
		case CheckInserts:
			// If we're inserting, then there's a violation if the scan found nothing.
			if fk.rf.kvEnd {
				fkValues := make(tree.Datums, fk.prefixLen)
				for valueIdx, colID := range fk.searchIdx.ColumnIDs[:fk.prefixLen] {
					fkValues[valueIdx] = newRow[fk.ids[colID]]
				}
				return pgerror.NewErrorf(pgerror.CodeForeignKeyViolationError,
					"foreign key violation: value %s not found in %s@%s %s",
					fkValues, fk.searchTable.Name, fk.searchIdx.Name, fk.searchIdx.ColumnNames[:fk.prefixLen])
			}
		case CheckDeletes:
			// If we're deleting, then there's a violation if the scan found something.
			if !fk.rf.kvEnd {
				if oldRow == nil {
					return pgerror.NewErrorf(pgerror.CodeForeignKeyViolationError,
						"foreign key violation: non-empty columns %s referenced in table %q",
						fk.writeIdx.ColumnNames[:fk.prefixLen], fk.searchTable.Name)
				}
				fkValues := make(tree.Datums, fk.prefixLen)
				for valueIdx, colID := range fk.searchIdx.ColumnIDs[:fk.prefixLen] {
					fkValues[valueIdx] = oldRow[fk.ids[colID]]
				}
				return pgerror.NewErrorf(pgerror.CodeForeignKeyViolationError,
					"foreign key violation: values %v in columns %s referenced in table %q",
					fkValues, fk.writeIdx.ColumnNames[:fk.prefixLen], fk.searchTable.Name)
			}
		default:
			log.Fatalf(ctx, "impossible case: baseFKHelper has dir=%v", fk.dir)
		}
	}
	return nil
}

type fkInsertHelper struct {
	// fks maps index id to slice of baseFKHelper, the outgoing foreign keys for
	// each index. These slices will have at most one entry, since there can be
	// at most one outgoing foreign key per index. We use this data structure
	// instead of a one-to-one map for consistency with the other insert helpers.
	fks map[IndexID][]baseFKHelper

	checker *fkBatchChecker
}

var errSkipUnusedFK = errors.New("no columns involved in FK included in writer")

func makeFKInsertHelper(
	txn *client.Txn,
	table TableDescriptor,
	otherTables TableLookupsByID,
	colMap map[ColumnID]int,
	alloc *DatumAlloc,
) (fkInsertHelper, error) {
	h := fkInsertHelper{
		checker: &fkBatchChecker{
			txn: txn,
		},
	}
	for _, idx := range table.AllNonDropIndexes() {
		if idx.ForeignKey.IsSet() {
			fk, err := makeBaseFKHelper(txn, otherTables, idx, idx.ForeignKey, colMap, alloc, CheckInserts)
			if err == errSkipUnusedFK {
				continue
			}
			if err != nil {
				return h, err
			}
			if h.fks == nil {
				h.fks = make(map[IndexID][]baseFKHelper)
			}
			h.fks[idx.ID] = append(h.fks[idx.ID], fk)
		}
	}
	return h, nil
}

func (h fkInsertHelper) checkAll(ctx context.Context, row tree.Datums) error {
	if len(h.fks) == 0 {
		return nil
	}
	for _, fkHelpers := range h.fks {
		h.checker.addInsertIndexChecks(fkHelpers)
	}
	return h.checker.runCheck(ctx, nil, row)
}

// CollectSpans implements the FkSpanCollector interface.
func (h fkInsertHelper) CollectSpans() roachpb.Spans {
	return collectSpansWithFKMap(h.fks)
}

// CollectSpansForValues implements the FkSpanCollector interface.
func (h fkInsertHelper) CollectSpansForValues(values tree.Datums) (roachpb.Spans, error) {
	return collectSpansForValuesWithFKMap(h.fks, values)
}

type fkDeleteHelper struct {
	fks         map[IndexID][]baseFKHelper
	otherTables TableLookupsByID
	alloc       *DatumAlloc

	checker *fkBatchChecker
}

func makeFKDeleteHelper(
	txn *client.Txn,
	table TableDescriptor,
	otherTables TableLookupsByID,
	colMap map[ColumnID]int,
	alloc *DatumAlloc,
) (fkDeleteHelper, error) {
	h := fkDeleteHelper{
		otherTables: otherTables,
		alloc:       alloc,
		checker: &fkBatchChecker{
			txn: txn,
		},
	}
	for _, idx := range table.AllNonDropIndexes() {
		for _, ref := range idx.ReferencedBy {
			if otherTables[ref.Table].IsAdding {
				// We can assume that a table being added but not yet public is empty,
				// and thus does not need to be checked for FK violations.
				continue
			}
			fk, err := makeBaseFKHelper(txn, otherTables, idx, ref, colMap, alloc, CheckDeletes)
			if err == errSkipUnusedFK {
				continue
			}
			if err != nil {
				return fkDeleteHelper{}, err
			}
			if h.fks == nil {
				h.fks = make(map[IndexID][]baseFKHelper)
			}
			h.fks[idx.ID] = append(h.fks[idx.ID], fk)
		}
	}
	return h, nil
}

func (h fkDeleteHelper) checkAll(ctx context.Context, row tree.Datums) error {
	if len(h.fks) == 0 {
		return nil
	}
	//for idx := range h.fks {
	//	h.checker.addDeleteIndexChecks(h.fks[idx])
	//}
	for _, fkHelpers := range h.fks {
		h.checker.addDeleteIndexChecks(fkHelpers)
	}
	return h.checker.runCheck(ctx, row, nil /* newRow */)
}

// CollectSpans implements the FkSpanCollector interface.
func (h fkDeleteHelper) CollectSpans() roachpb.Spans {
	return collectSpansWithFKMap(h.fks)
}

// CollectSpansForValues implements the FkSpanCollector interface.
func (h fkDeleteHelper) CollectSpansForValues(values tree.Datums) (roachpb.Spans, error) {
	return collectSpansForValuesWithFKMap(h.fks, values)
}

type fkUpdateHelper struct {
	inbound  fkDeleteHelper // Check old values are not referenced.
	outbound fkInsertHelper // Check rows referenced by new values still exist.

	checker *fkBatchChecker
}

func makeFKUpdateHelper(
	txn *client.Txn,
	table TableDescriptor,
	otherTables TableLookupsByID,
	colMap map[ColumnID]int,
	alloc *DatumAlloc,
) (fkUpdateHelper, error) {
	ret := fkUpdateHelper{}
	var err error
	if ret.inbound, err = makeFKDeleteHelper(txn, table, otherTables, colMap, alloc); err != nil {
		return ret, err
	}
	ret.outbound, err = makeFKInsertHelper(txn, table, otherTables, colMap, alloc)
	ret.outbound.checker = ret.inbound.checker
	ret.checker = ret.inbound.checker
	return ret, err
}

func (fks fkUpdateHelper) checkIdx(
	ctx context.Context, idx IndexID, oldValues, newValues tree.Datums,
) error {
	fks.checker.addDeleteIndexChecks(fks.inbound.fks[idx])
	fks.checker.addInsertIndexChecks(fks.outbound.fks[idx])
	return nil
}

// CollectSpans implements the FkSpanCollector interface.
func (fks fkUpdateHelper) CollectSpans() roachpb.Spans {
	inboundReads := fks.inbound.CollectSpans()
	outboundReads := fks.outbound.CollectSpans()
	return append(inboundReads, outboundReads...)
}

// CollectSpansForValues implements the FkSpanCollector interface.
func (fks fkUpdateHelper) CollectSpansForValues(values tree.Datums) (roachpb.Spans, error) {
	inboundReads, err := fks.inbound.CollectSpansForValues(values)
	if err != nil {
		return nil, err
	}
	outboundReads, err := fks.outbound.CollectSpansForValues(values)
	if err != nil {
		return nil, err
	}
	return append(inboundReads, outboundReads...), nil
}

type baseFKHelper struct {
	txn          *client.Txn
	rf           MultiRowFetcher
	searchTable  *TableDescriptor // the table being searched (for err msg)
	searchIdx    *IndexDescriptor // the index that must (not) contain a value
	prefixLen    int
	writeIdx     IndexDescriptor  // the index we want to modify
	searchPrefix []byte           // prefix of keys in searchIdx
	ids          map[ColumnID]int // col IDs
	dir          FKCheck          // direction of check
}

func makeBaseFKHelper(
	txn *client.Txn,
	otherTables TableLookupsByID,
	writeIdx IndexDescriptor,
	ref ForeignKeyReference,
	colMap map[ColumnID]int,
	alloc *DatumAlloc,
	dir FKCheck,
) (baseFKHelper, error) {
	b := baseFKHelper{txn: txn, writeIdx: writeIdx, searchTable: otherTables[ref.Table].Table, dir: dir}
	if b.searchTable == nil {
		return b, errors.Errorf("referenced table %d not in provided table map %+v", ref.Table, otherTables)
	}
	b.searchPrefix = MakeIndexKeyPrefix(b.searchTable, ref.Index)
	searchIdx, err := b.searchTable.FindIndexByID(ref.Index)
	if err != nil {
		return b, err
	}
	b.prefixLen = len(searchIdx.ColumnIDs)
	if len(writeIdx.ColumnIDs) < b.prefixLen {
		b.prefixLen = len(writeIdx.ColumnIDs)
	}
	b.searchIdx = searchIdx
	tableArgs := MultiRowFetcherTableArgs{
		Desc:             b.searchTable,
		Index:            b.searchIdx,
		ColIdxMap:        ColIDtoRowIndexFromCols(b.searchTable.Columns),
		IsSecondaryIndex: b.searchIdx.ID != b.searchTable.PrimaryIndex.ID,
		Cols:             b.searchTable.Columns,
	}
	err = b.rf.Init(false /* reverse */, false /* returnRangeInfo */, false /* isCheck */, alloc, tableArgs)
	if err != nil {
		return b, err
	}

	b.ids = make(map[ColumnID]int, len(writeIdx.ColumnIDs))
	nulls := true
	for i, writeColID := range writeIdx.ColumnIDs[:b.prefixLen] {
		if found, ok := colMap[writeColID]; ok {
			b.ids[searchIdx.ColumnIDs[i]] = found
			nulls = false
		} else if !nulls {
			return b, errors.Errorf("missing value for column %q in multi-part foreign key", writeIdx.ColumnNames[i])
		}
	}
	if nulls {
		return b, errSkipUnusedFK
	}
	return b, nil
}

func (f baseFKHelper) spanForValues(values tree.Datums) (roachpb.Span, error) {
	var key roachpb.Key
	if values != nil {
		keyBytes, _, err := EncodePartialIndexKey(
			f.searchTable, f.searchIdx, f.prefixLen, f.ids, values, f.searchPrefix)
		if err != nil {
			return roachpb.Span{}, err
		}
		key = roachpb.Key(keyBytes)
	} else {
		key = roachpb.Key(f.searchPrefix)
	}
	return roachpb.Span{Key: key, EndKey: key.PrefixEnd()}, nil
}

func (f baseFKHelper) span() roachpb.Span {
	key := roachpb.Key(f.searchPrefix)
	return roachpb.Span{Key: key, EndKey: key.PrefixEnd()}
}

// FkSpanCollector can collect the spans that foreign key validation will touch.
type FkSpanCollector interface {
	CollectSpans() roachpb.Spans
	CollectSpansForValues(values tree.Datums) (roachpb.Spans, error)
}

var _ FkSpanCollector = fkInsertHelper{}
var _ FkSpanCollector = fkDeleteHelper{}
var _ FkSpanCollector = fkUpdateHelper{}

func collectSpansWithFKMap(fks map[IndexID][]baseFKHelper) roachpb.Spans {
	var reads roachpb.Spans
	for idx := range fks {
		for _, fk := range fks[idx] {
			reads = append(reads, fk.span())
		}
	}
	return reads
}

func collectSpansForValuesWithFKMap(
	fks map[IndexID][]baseFKHelper, values tree.Datums,
) (roachpb.Spans, error) {
	var reads roachpb.Spans
	for idx := range fks {
		for _, fk := range fks[idx] {
			read, err := fk.spanForValues(values)
			if err != nil {
				return nil, err
			}
			reads = append(reads, read)
		}
	}
	return reads, nil
}
