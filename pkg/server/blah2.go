package server

import (
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
)

const a = sqlbase.TableDescriptor{Name: "_",
	ID:        0xf9,
	ParentID:  0x32,
	Version:   0x2,
	UpVersion: false,
	ModificationTime: hlc.Timestamp{WallTime: 1515744481037025800,
		Logical: 0},
	Columns:      []sqlbase.ColumnDescriptor(nil),
	NextColumnID: 0x1,
	Families:     []sqlbase.ColumnFamilyDescriptor(nil),
	NextFamilyID: 0x0,
	PrimaryIndex: sqlbase.IndexDescriptor{Name: "",
		ID:                 0x0,
		Unique:             false,
		ColumnNames:        []string(nil),
		ColumnDirections:   []sqlbase.IndexDescriptor_Direction(nil),
		StoreColumnNames:   []string(nil),
		ColumnIDs:          []sqlbase.ColumnID(nil),
		ExtraColumnIDs:     []sqlbase.ColumnID(nil),
		StoreColumnIDs:     []sqlbase.ColumnID(nil),
		CompositeColumnIDs: []sqlbase.ColumnID(nil),
		ForeignKey: sqlbase.ForeignKeyReference{Table: 0x0,
			Index:           0x0,
			Name:            "",
			Validity:        0,
			SharedPrefixLen: 0,
			OnDelete:        0,
			OnUpdate:        0},
		ReferencedBy:  []sqlbase.ForeignKeyReference(nil),
		Interleave:    sqlbase.InterleaveDescriptor{Ancestors: []sqlbase.InterleaveDescriptor_Ancestor(nil)},
		InterleavedBy: []sqlbase.ForeignKeyReference(nil),
		Partitioning: sqlbase.PartitioningDescriptor{NumColumns: 0x0,
			List:  []sqlbase.PartitioningDescriptor_List(nil),
			Range: []sqlbase.PartitioningDescriptor_Range(nil)}},
	Indexes:        []sqlbase.IndexDescriptor(nil),
	NextIndexID:    0x0,
	Privileges:     (*sqlbase.PrivilegeDescriptor)(0xc4204e61c0),
	Mutations:      []sqlbase.DescriptorMutation(nil),
	Lease:          (*sqlbase.TableDescriptor_SchemaChangeLease)(nil),
	NextMutationID: 0x1,
	FormatVersion:  0x3,
	State:          2,
	Checks:         []*sqlbase.TableDescriptor_CheckConstraint(nil),
	Renames:        []sqlbase.TableDescriptor_RenameInfo(nil),
	ViewQuery:      "",
	DependsOn:      []sqlbase.ID(nil),
	DependedOnBy:   []sqlbase.TableDescriptor_Reference(nil),
	MutationJobs:   []sqlbase.TableDescriptor_MutationJob(nil),
	SequenceOpts:   (*sqlbase.TableDescriptor_SequenceOpts)(nil)}
