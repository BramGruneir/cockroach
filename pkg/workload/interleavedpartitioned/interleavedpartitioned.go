// Copyright 2018 The Cockroach Authors.
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
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.

package interleavedpartitioned

import (
	"context"
	gosql "database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
	"github.com/cockroachdb/cockroach/pkg/workload"
)

const (
	alphas        = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
	sessionSchema = `
(
	session_id STRING(100) PRIMARY KEY,
	affiliate STRING(100) NOT NULL,
	channel STRING(50) NOT NULL,
	language STRING(20) NOT NULL,
	created TIMESTAMP NOT NULL,
	updated TIMESTAMP NOT NULL,
	status STRING(20) NOT NULL,
	platform STRING(50) NOT NULL,
	query_id STRING(100) NOT NULL,
	INDEX con_session_created_idx(created),
	FAMILY "primary" (session_id, affiliate, channel, language, created, updated, status, platform, query_id)
) PARTITION BY RANGE (session_id) (
	PARTITION east VALUES FROM ('E-') TO ('F-'),
	PARTITION west VALUES FROM ('W-') TO ('X-'),
	PARTITION central VALUES FROM ('C-') TO ('D-')
)`
	genericChildSchema = `
(
	session_id STRING(100) NOT NULL,
	key STRING(50) NOT NULL,
	value STRING(50) NOT NULL,
	created TIMESTAMP NOT NULL,
	updated TIMESTAMP NOT NULL,
	PRIMARY KEY (session_id, key),
	FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE,
	FAMILY "primary" (session_id, key, value, created, updated)
) INTERLEAVE IN PARENT sessions(session_id)`
	deviceSchema = `
(
	id STRING(100) NOT NULL,
	session_id STRING(100) NOT NULL REFERENCES sessions ON DELETE CASCADE,
	device_id STRING(50),
	name STRING(50),
	make STRING(50),
	macaddress STRING(50),
	model STRING(50),
	serialno STRING(50),
	created TIMESTAMP NOT NULL,
	updated TIMESTAMP NOT NULL,
	PRIMARY KEY (session_id, id),
	FAMILY "primary" (id, session_id, device_id, name, make, macaddress, model, serialno, created, updated)
) INTERLEAVE IN PARENT sessions(session_id)
`
	querySchema = `
(
	session_id STRING(100) NOT NULL REFERENCES sessions ON DELETE CASCADE,
	id STRING(50) NOT NULL,
	created TIMESTAMP NOT NULL,
	updated TIMESTAMP NOT NULL,
	PRIMARY KEY (session_id, id),
	FAMILY "primary" (session_id, id, created, updated)
) INTERLEAVE IN PARENT sessions(session_id)
`
)

func init() {
	workload.Register(interleavedPartitionedMeta)
}

type interleavedPartitioned struct {
	flags     workload.Flags
	connFlags *workload.ConnFlags

	sessions             int
	customersPerSession  int
	devicesPerSession    int
	variantsPerSession   int
	parametersPerSession int
	queriesPerSession    int
	eastPercent          int

	sessionIDs []string
}

var interleavedPartitionedMeta = workload.Meta{
	Name:        `interleavedpartitioned`,
	Description: `Tests the performance of tables that are both interleaved and partitioned`,
	Version:     `1.0.0`,
	New: func() workload.Generator {
		g := &interleavedPartitioned{}
		g.flags.FlagSet = pflag.NewFlagSet(`interleavedpartitioned`, pflag.ContinueOnError)
		g.flags.Meta = map[string]workload.FlagMeta{
			`batch`: {RuntimeOnly: true},
		}
		g.flags.IntVar(&g.sessions, `sessions`, 1, `Number of sessions (rows in the parent table)`)
		g.flags.IntVar(&g.customersPerSession, `customers-per-session`, 1, `Number of customers associated with each session`)
		g.flags.IntVar(&g.devicesPerSession, `devices-per-session`, 1, `Number of devices associated with each session`)
		g.flags.IntVar(&g.variantsPerSession, `variants-per-session`, 1, `Number of variants associated with each session`)
		g.flags.IntVar(&g.parametersPerSession, `parameters-per-session`, 1, `Number of parameters associated with each session`)
		g.flags.IntVar(&g.queriesPerSession, `queries-per-session`, 1, `Number of queries associated with each session`)
		g.flags.IntVar(&g.eastPercent, `east-percent`, 50, `Percentage of sessions that are in us-east`)
		g.connFlags = workload.NewConnFlags(&g.flags)
		return g
	},
}

// Meta implements the Generator interface.
func (w *interleavedPartitioned) Meta() workload.Meta { return interleavedPartitionedMeta }

// Flags implements the Flagser interface.
func (w *interleavedPartitioned) Flags() workload.Flags { return w.flags }

// Tables implements the Generator interface.
func (w *interleavedPartitioned) Tables() []workload.Table {
	sessionsTable := workload.Table{
		Name:   `sessions`,
		Schema: sessionSchema,
		InitialRows: workload.Tuples(
			w.sessions,
			w.sessionsInitialRow,
		),
	}
	customerTable := workload.Table{
		Name:   `customers`,
		Schema: genericChildSchema,
		InitialRows: workload.BatchedTuples{
			NumBatches: w.sessions,
			Batch:      w.childInitialRowBatchFunc(2, w.customersPerSession),
		},
	}
	devicesTable := workload.Table{
		Name:   `devices`,
		Schema: deviceSchema,
		InitialRows: workload.BatchedTuples{
			NumBatches: w.sessions,
			Batch:      w.deviceInitialRowBatch,
		},
	}
	variantsTable := workload.Table{
		Name:   `variants`,
		Schema: genericChildSchema,
		InitialRows: workload.BatchedTuples{
			NumBatches: w.sessions,
			Batch:      w.childInitialRowBatchFunc(3, w.variantsPerSession),
		},
	}
	parametersTable := workload.Table{
		Name:   `parameters`,
		Schema: genericChildSchema,
		InitialRows: workload.BatchedTuples{
			NumBatches: w.sessions,
			Batch:      w.childInitialRowBatchFunc(4, w.parametersPerSession),
		},
	}
	queriesTable := workload.Table{
		Name:   `queries`,
		Schema: querySchema,
		InitialRows: workload.BatchedTuples{
			NumBatches: w.sessions,
			Batch:      w.queryInitialRowBatch,
		},
	}
	return []workload.Table{sessionsTable, customerTable, devicesTable, variantsTable, parametersTable, queriesTable}
}

func (w *interleavedPartitioned) Ops(
	urls []string, reg *workload.HistogramRegistry,
) (workload.QueryLoad, error) {
	sqlDatabase, err := workload.SanitizeUrls(w, ``, urls)
	if err != nil {
		return workload.QueryLoad{}, err
	}

	db, err := gosql.Open(`cockroach`, strings.Join(urls, ` `))
	if err != nil {
		return workload.QueryLoad{}, err
	}

	db.SetMaxOpenConns(w.connFlags.Concurrency + 1)
	db.SetMaxIdleConns(w.connFlags.Concurrency + 1)

	if err != nil {
		return workload.QueryLoad{}, err
	}

	ql := workload.QueryLoad{
		SQLDatabase: sqlDatabase,
	}

	statement, err := db.Prepare(`DELETE from sessions WHERE 1=1`)
	if err != nil {
		return ql, err
	}
	ql.WorkerFns = append(ql.WorkerFns, func(ctx context.Context) error {
		args := make([]interface{}, 0)
		rows, err := statement.Query(args...)
		if err != nil {
			return err
		}
		return rows.Err()
	})

	return ql, nil
}

func (w *interleavedPartitioned) sessionsInitialRow(rowIdx int) []interface{} {
	rng := rand.New(rand.NewSource(int64(rowIdx)))
	nowString := timeutil.Now().UTC().Format(time.RFC3339)
	sessionID := w.generateSessionID(rng, rowIdx)
	w.sessionIDs = append(w.sessionIDs, sessionID)
	log.Warningf(context.TODO(), "inserting into parent row %d, len of sessionIDs is now %d", rowIdx, len(w.sessionIDs))
	return []interface{}{
		sessionID,            // session_id
		randString(rng, 100), // affiliate
		randString(rng, 50),  // channel
		randString(rng, 20),  // language
		nowString,            // created
		nowString,            // updated
		randString(rng, 20),  // status
		randString(rng, 50),  // platform
		randString(rng, 100), // query_id
	}
}

func (w *interleavedPartitioned) childInitialRowBatchFunc(
	rngFactor int64, nPerBatch int,
) func(int) [][]interface{} {
	return func(sessionRowIdx int) [][]interface{} {
		log.Warningf(context.TODO(), "inserting into child %d, sessionRowIdx %d. len(w.sessionIDs) = %d", rngFactor, sessionRowIdx, len(w.sessionIDs))
		rng := rand.New(rand.NewSource(int64(sessionRowIdx) + rngFactor))
		if sessionRowIdx >= len(w.sessionIDs) {
			sessionRowIdx = randInt(rng, 0, len(w.sessionIDs)-1)
			// log.Warningf(context.TODO(), "len(w.sessionIDs): %d, w.sessionIDs[len] = %s", len(w.sessionIDs), w.sessionIDs[sessionRowIdx])
		}
		sessionID := w.sessionIDs[sessionRowIdx]
		nowString := timeutil.Now().UTC().Format(time.RFC3339)
		var rows [][]interface{}
		for i := 0; i < nPerBatch; i++ {
			rows = append(rows, []interface{}{
				sessionID,
				randString(rng, 50), // key
				randString(rng, 50), // value
				nowString,           // created
				nowString,           // updated
			})
		}
		return rows
	}
}

func (w *interleavedPartitioned) deviceInitialRowBatch(sessionRowIdx int) [][]interface{} {
	log.Warningf(context.TODO(), "inserting into devices, sessionRowIdx %d", sessionRowIdx)
	rng := rand.New(rand.NewSource(int64(sessionRowIdx) * 64))
	if sessionRowIdx >= len(w.sessionIDs) {
		sessionRowIdx = randInt(rng, 0, len(w.sessionIDs)-1)
	}
	sessionID := w.sessionIDs[sessionRowIdx]
	nowString := timeutil.Now().UTC().Format(time.RFC3339)
	var rows [][]interface{}
	for i := 0; i < w.devicesPerSession; i++ {
		rows = append(rows, []interface{}{
			randString(rng, 100), // id
			sessionID,
			randString(rng, 50), // device_id
			randString(rng, 50), // name
			randString(rng, 50), // make
			randString(rng, 50), // macaddress
			randString(rng, 50), // model
			randString(rng, 50), // serialno
			nowString,           // created
			nowString,           // updated
		})
	}
	return rows
}

func (w *interleavedPartitioned) queryInitialRowBatch(sessionRowIdx int) [][]interface{} {
	log.Warningf(context.TODO(), "inserting into queries, sessionRowIdx %d", sessionRowIdx)
	var rows [][]interface{}
	rng := rand.New(rand.NewSource(int64(sessionRowIdx) * 64))
	if sessionRowIdx >= len(w.sessionIDs) {
		sessionRowIdx = randInt(rng, 0, len(w.sessionIDs)-1)
	}
	sessionID := w.sessionIDs[sessionRowIdx]
	nowString := timeutil.Now().UTC().Format(time.RFC3339)
	for i := 0; i < w.queriesPerSession; i++ {
		rows = append(rows, []interface{}{
			sessionID,
			randString(rng, 50), // id
			nowString,           // created
			nowString,           // updated
		})
	}
	return rows
}

func (w *interleavedPartitioned) generateSessionID(rng *rand.Rand, rowIdx int) string {
	sessionIDBase := randString(rng, 98)
	if (rowIdx % 100) >= w.eastPercent {
		return fmt.Sprintf("E-%s", sessionIDBase)
	}
	return fmt.Sprintf("W-%s", sessionIDBase)
}

func randString(rng *rand.Rand, length int) string {
	var s string
	for i := 0; i < length; i++ {
		idx := randInt(rng, 0, len(alphas)-1)
		s = s + string(alphas[idx])
	}
	return s
}

func randInt(rng *rand.Rand, min, max int) int {
	return rng.Intn(max-min+1) + min
}
