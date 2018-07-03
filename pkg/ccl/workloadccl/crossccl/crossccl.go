// Copyright 2018 The Cockroach Authors.
//
// Licensed as a CockroachDB Enterprise file under the Cockroach Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/cockroachdb/cockroach/blob/master/licenses/CCL.txt

package geoccl

import (
	"context"
	gosql "database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/cockroachdb/cockroach/pkg/util/randutil"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
	"github.com/cockroachdb/cockroach/pkg/util/uint128"
	"github.com/cockroachdb/cockroach/pkg/util/uuid"
	"github.com/cockroachdb/cockroach/pkg/workload"
)

// These need to be kept in sync with the zones used when --geo is passed
// to roachprod.
var zones = []string{"east", "west"}

const (
	sessionsSchema = `
(
  id STRING PRIMARY KEY
 ,a STRING NOT NULL
 ,b STRING NOT NULL
 ,c STRING NOT NULL
 ,d STRING NOT NULL
 ,e STRING NULL
 ,f STRING NULL
 ,created TIMESTAMP NOT NULL
 ,updated TIMESTAMP NOT NULL
) PARTITION BY RANGE (id) (
  PARTITION east VALUES FROM ('E-') TO ('F')
 ,PARTITION west VALUES FROM ('W-') TO ('X')
)
`

	customersSchema = `
(
  session_id STRING NOT NULL REFERENCES sessions
 ,id STRING NOT NULL
 ,a STRING NOT NULL
 ,created TIMESTAMP NOT NULL
 ,updated TIMESTAMP NOT NULL
 ,PRIMARY KEY (session_id, id)
) INTERLEAVE IN PARENT sessions (session_id)
`

	devicesSchema = `
(
  session_id STRING NOT NULL REFERENCES sessions
 ,id STRING NOT NULL
 ,a STRING NOT NULL
 ,b STRING NOT NULL
 ,c STRING NOT NULL
 ,d STRING NOT NULL
 ,e STRING NOT NULL
 ,f STRING NOT NULL
 ,created TIMESTAMP NOT NULL
 ,updated TIMESTAMP NOT NULL
 ,PRIMARY KEY (session_id, id)
 ) INTERLEAVE IN PARENT sessions (session_id)
`

	variantsSchema = `
(
  session_id STRING NOT NULL REFERENCES sessions
 ,id STRING NOT NULL
 ,a STRING NOT NULL
 ,created TIMESTAMP NOT NULL
 ,updated TIMESTAMP NOT NULL
 ,PRIMARY KEY (session_id, id)
) INTERLEAVE IN PARENT sessions (session_id)
`

	parametersSchema = `
(
  session_id STRING NOT NULL REFERENCES sessions
 ,id STRING NOT NULL
 ,a STRING NOT NULL
 ,created TIMESTAMP NOT NULL
 ,updated TIMESTAMP NOT NULL
 ,PRIMARY KEY (session_id, id)
) INTERLEAVE IN PARENT sessions (session_id)
`

	defaultSessions             = 10000
	defaultCustomersPerSession  = 2
	defaultDevicesPerSession    = 1
	defaultVariantsPerSession   = 0
	defaultParametersPerSession = 0
	defaultEastPercent          = 50

	// Default setup is a ration of 2:5:5 I:R:U (ish)
	defaultInsertPercent   = 16
	defaultRetrievePercent = 41

	defaultInsertLocalPercent   = 100
	defaultRetrieveLocalPercent = 100
	defaultUpdateLocalPercent   = 100

	zoneLocationsStmt = `
UPSERT INTO system.locations VALUES
	('zone', 'us-east1-b', 33.0641249, -80.0433347)
 ,('zone', 'us-west1-b', 45.6319052, -121.2010282)
`
)

type crossccl struct {
	flags     workload.Flags
	connFlags *workload.ConnFlags

	seed int64

	east bool

	sessions             int
	customersPerSession  int
	devicesPerSession    int
	variantsPerSession   int
	parametersPerSession int
	eastPercent          int

	insertPercent   int
	retrievePercent int

	insertLocalPercent   int
	retrieveLocalPercent int
	updateLocalPercent   int
}

func init() {
	workload.Register(crosscclMeta)
}

var crosscclMeta = workload.Meta{
	Name:        `crossccl`,
	Description: `crossccl models an example workload with customers locked in either east or west`,
	Version:     `1.0.0`,
	New: func() workload.Generator {
		c := &crossccl{}
		c.flags.FlagSet = pflag.NewFlagSet(`crossccl`, pflag.ContinueOnError)
		c.flags.Int64Var(&c.seed, `seed`, 0, `Key hash seed.`)
		c.flags.IntVar(&c.sessions, `sessions`, defaultSessions, `Initial number of sessions in users table.`)
		c.flags.IntVar(&c.customersPerSession, `customersPerSession`, defaultCustomersPerSession, `Number of customers per session.`)
		c.flags.IntVar(&c.devicesPerSession, `devicesPerSession`, defaultDevicesPerSession, `Number of devices per session.`)
		c.flags.IntVar(&c.variantsPerSession, `variantsPerSession`, defaultVariantsPerSession, `Number of variants per session.`)
		c.flags.IntVar(&c.parametersPerSession, `parametersPerSession`, defaultParametersPerSession, `Number of parameters per session.`)
		c.flags.IntVar(&c.eastPercent, `eastPercent`, defaultEastPercent, `Percent (0-100) of initial sessions created in the east (as opposed to west).`)
		c.flags.IntVar(&c.insertPercent, `insertPercent`, defaultInsertPercent, `Percent (0-100) of operations that are inserts. inserts + retrieves + update = 100`)
		c.flags.IntVar(&c.retrievePercent, `retrievePercent`, defaultRetrievePercent, `Percent (0-100) of operations that are retrieves. inserts + retrieves + update = 100`)
		c.flags.IntVar(&c.insertLocalPercent, `insertLocalPercent`, defaultInsertLocalPercent, `Percent (0-100) of Insert operations that should be local.`)
		c.flags.IntVar(&c.retrieveLocalPercent, `retrieveLocalPercent`, defaultRetrieveLocalPercent, `Percent (0-100) of Retrive operations that should be local.`)
		c.flags.IntVar(&c.updateLocalPercent, `updateLocalPercent`, defaultUpdateLocalPercent, `Percent (0-100) of Update operations that should be local.`)
		c.flags.BoolVar(&c.east, `east`, true, `Is this location in the east (true) or west (false)`)
		c.connFlags = workload.NewConnFlags(&c.flags)
		return c
	},
}

// Meta implements the Generator interface.
func (*crossccl) Meta() workload.Meta { return crosscclMeta }

// Flags implements the Flagser interface.
func (c *crossccl) Flags() workload.Flags { return c.flags }

// Hooks implements the Hookser interface.
func (c *crossccl) Hooks() workload.Hooks {
	return workload.Hooks{
		PreLoad: func(db *gosql.DB) error {
			if _, err := db.Exec(zoneLocationsStmt); err != nil {
				return err
			}
			if _, err := db.Exec(
				"ALTER PARTITION west OF TABLE sessions EXPERIMENTAL CONFIGURE ZONE 'experimental_lease_preferences: [[+zone=us-west1-b]]'",
			); err != nil {
				return errors.Wrapf(err, "could not set zone for partition east")
			}
			if _, err := db.Exec(
				"ALTER PARTITION east OF TABLE sessions EXPERIMENTAL CONFIGURE ZONE 'experimental_lease_preferences: [[+zone=us-east1-b]]'",
			); err != nil {
				return errors.Wrapf(err, "could not set zone for partition west")
			}
			return nil
		},
	}
}

func (c *crossccl) generateSessionID(rowID int) string {
	largeID := uint128.Uint128{Lo: uint64(rowID)}
	id := uuid.FromUint128(largeID)
	if (rowID % 100) > c.eastPercent {
		return fmt.Sprintf("E-%s", id)
	}
	return fmt.Sprintf("W-%s", id)
}

// Tables implements the Generator interface.
func (c *crossccl) Tables() []workload.Table {
	rng := rand.New(rand.NewSource(c.seed))
	sessions := workload.Table{
		Name:            `sessions`,
		Schema:          sessionsSchema,
		InitialRowCount: c.sessions,
		InitialRowFn: func(rowIdx int) []interface{} {
			return []interface{}{
				c.generateSessionID(rowIdx),         // id
				string(randutil.RandBytes(rng, 50)), // a
				string(randutil.RandBytes(rng, 50)), // b
				string(randutil.RandBytes(rng, 20)), // c
				string(randutil.RandBytes(rng, 20)), // d
				string(randutil.RandBytes(rng, 50)), // e
				string(randutil.RandBytes(rng, 10)), // f
				timeutil.Now(),                      // date_created
				timeutil.Now(),                      // date_updated
			}
		},
	}
	customers := workload.Table{
		Name:            `customers`,
		Schema:          customersSchema,
		InitialRowCount: c.sessions * c.customersPerSession,
		InitialRowFn: func(rowIdx int) []interface{} {
			return []interface{}{
				c.generateSessionID(rowIdx / c.customersPerSession), // session_id
				fmt.Sprint(rowIdx),                                  // id
				string(randutil.RandBytes(rng, 50)),                 // a
				timeutil.Now(),                                      // date_created
				timeutil.Now(),                                      // date_updated
			}
		},
	}
	devices := workload.Table{
		Name:            `devices`,
		Schema:          devicesSchema,
		InitialRowCount: c.sessions * c.devicesPerSession,
		InitialRowFn: func(rowIdx int) []interface{} {
			return []interface{}{
				c.generateSessionID(rowIdx / c.devicesPerSession), // sessions_id
				fmt.Sprint(rowIdx),                                // id
				string(randutil.RandBytes(rng, 50)),               // a
				string(randutil.RandBytes(rng, 50)),               // b
				string(randutil.RandBytes(rng, 50)),               // c
				string(randutil.RandBytes(rng, 50)),               // d
				string(randutil.RandBytes(rng, 50)),               // e
				string(randutil.RandBytes(rng, 50)),               // f
				timeutil.Now(),                                    // date_created
				timeutil.Now(),                                    // date_updated
			}
		},
	}
	variants := workload.Table{
		Name:            `variants`,
		Schema:          variantsSchema,
		InitialRowCount: c.sessions * c.variantsPerSession,
		InitialRowFn: func(rowIdx int) []interface{} {
			return []interface{}{
				c.generateSessionID(rowIdx / c.variantsPerSession), // session_id
				fmt.Sprint(rowIdx),                                 // id
				string(randutil.RandBytes(rng, 50)),                // a
				timeutil.Now(),                                     // date_created
				timeutil.Now(),                                     // date_updated
			}
		},
	}
	parameters := workload.Table{
		Name:            `parameters`,
		Schema:          parametersSchema,
		InitialRowCount: c.sessions * c.parametersPerSession,
		InitialRowFn: func(rowIdx int) []interface{} {
			return []interface{}{
				c.generateSessionID(rowIdx / c.parametersPerSession), // session_id
				fmt.Sprint(rowIdx),                                   // id
				string(randutil.RandBytes(rng, 50)),                  // a
				timeutil.Now(),                                       // date_created
				timeutil.Now(),                                       // date_updated
			}
		},
	}
	return []workload.Table{sessions, customers, devices, variants, parameters}
}

func (c *crossccl) createRandomSessionID(rng *rand.Rand, east bool) string {
	id, err := uuid.FromBytes(randutil.RandBytes(rng, 16))
	if err != nil {
		panic(err)
	}
	if east {
		return fmt.Sprintf("E-%s", id)
	}
	return fmt.Sprintf("W-%s", id)
}

const insertQuerySessions = `INSERT INTO sessions VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())`
const insertQueryCustomers = `INSERT INTO customers VALUES ($1, $2, $3, now(), now())`
const insertQueryDevices = `INSERT INTO devices VALUES ($1, $2, $3, $4, $5, $6, now(), now())`
const insertQueryVariants = `INSERT INTO variants VALUES ($1, $2, now(), now())`
const insertQueryParameters = `INSERT INTO parameters VALUES ($1, $2, now(), now())`

const retrieveQuery0 = `SELECT id FROM sessions WHERE id > $1 LIMIT 1`
const retrieveQuery1 = `
SELECT id, a, b, created, c, e, f, d, updated
FROM sessions
WHERE id = $1
`

const retrieveQuery2 = `
SELECT
  device.id
 ,device.session_id
 ,device.created
 ,device.a
 ,device.d
 ,device.c
 ,device.e
 ,device.b
 ,device.f
 ,device.updated
 ,session.id
 ,session.a
 ,session.b
 ,session.created
 ,session.c
 ,session.e
 ,session.f
 ,session.d
 ,session.updated
FROM sessions AS session
LEFT OUTER JOIN devices AS device
ON session.id = device.session_id
WHERE session.id = $1
`

const retrieveQuery3 = `
UPDATE sessions
SET updated = now()
WHERE id = $1
`

const retrieveQuery4 = `
SELECT session_id, id, id, session_id, created, a, updated
FROM customers
WHERE session_id = $1
`

const retrieveQuery5 = `
SELECT session_id, id, id, session_id created, a, updated
FROM parameters
WHERE session_id = $1
`

const retrieveQuery6 = `
SELECT session_id, id, id, session_id created, a, updated
FROM variants
WHERE session_id = $1
`

const updateQuery0 = retrieveQuery0
const updateQuery1 = retrieveQuery1
const updateQuery2 = retrieveQuery2
const updateQuery3 = retrieveQuery4
const updateQuery4 = retrieveQuery5
const updateQuery5 = retrieveQuery6
const updateQuery6 = `
UPDATE sessions
SET f = $1, updated = now()
WHERE id = $2
`

func (c *crossccl) pickLocality(rng *rand.Rand, percent int) bool {
	localRand := rng.Intn(100)
	if localRand < percent {
		return c.east
	}
	return !c.east
}

// Ops implements the Opser interface.
func (c *crossccl) Ops(
	urls []string, reg *workload.HistogramRegistry,
) (workload.QueryLoad, error) {
	sqlDatabase, err := workload.SanitizeUrls(c, c.connFlags.DBOverride, urls)
	if err != nil {
		return workload.QueryLoad{}, err
	}
	db, err := gosql.Open(`cockroach`, strings.Join(urls, ` `))
	if err != nil {
		return workload.QueryLoad{}, err
	}
	// Allow a maximum of concurrency+1 connections to the database.
	db.SetMaxOpenConns(c.connFlags.Concurrency + 1)
	db.SetMaxIdleConns(c.connFlags.Concurrency + 1)

	ql := workload.QueryLoad{SQLDatabase: sqlDatabase}
	hists := reg.GetHandle()

	for i := 0; i < c.connFlags.Concurrency; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		workerFn := func(ctx context.Context) error {
			opRand := rng.Intn(100)
			if opRand < c.insertPercent {
				// Insert

				sessionID := c.createRandomSessionID(rng, c.pickLocality(rng, c.insertLocalPercent))
				start1 := timeutil.Now()
				if _, err := db.ExecContext(ctx, insertQuerySessions,
					sessionID,                           // ID
					string(randutil.RandBytes(rng, 50)), // a
					string(randutil.RandBytes(rng, 50)), // b
					string(randutil.RandBytes(rng, 20)), // c
					string(randutil.RandBytes(rng, 20)), // d
					string(randutil.RandBytes(rng, 50)), // e
					string(randutil.RandBytes(rng, 10)), // f
				); err != nil {
					return errors.Wrapf(err, "error running %s", insertQuerySessions)
				}
				hists.Get(`insertQuerySession`).Record(timeutil.Since(start1))

				// Devices
				for i = 0; i < c.devicesPerSession; i++ {
					start := timeutil.Now()
					if _, err := db.ExecContext(ctx, insertQueryDevices,
						sessionID,                           // session_id
						fmt.Sprint(i),                       // id
						string(randutil.RandBytes(rng, 50)), // a
						string(randutil.RandBytes(rng, 50)), // b
						string(randutil.RandBytes(rng, 20)), // c
						string(randutil.RandBytes(rng, 20)), // d
						string(randutil.RandBytes(rng, 50)), // e
						string(randutil.RandBytes(rng, 10)), // f
					); err != nil {
						return errors.Wrapf(err, "error running %s", insertQueryDevices)
					}
					hists.Get(`insertQueryDevices`).Record(timeutil.Since(start))
				}

				// Customers
				for i = 0; i < c.customersPerSession; i++ {
					start := timeutil.Now()
					if _, err := db.ExecContext(ctx, insertQueryCustomers,
						sessionID,                           // session_id
						fmt.Sprint(i),                       // id
						string(randutil.RandBytes(rng, 50)), // a
					); err != nil {
						return errors.Wrapf(err, "error running %s", insertQueryCustomers)
					}
					hists.Get(`insertQueryCustomers`).Record(timeutil.Since(start))
				}

				// Variants
				for i = 0; i < c.variantsPerSession; i++ {
					start := timeutil.Now()
					if _, err := db.ExecContext(ctx, insertQueryVariants,
						sessionID,                           // session_id
						fmt.Sprint(i),                       // id
						string(randutil.RandBytes(rng, 50)), // a
					); err != nil {
						return errors.Wrapf(err, "error running %s", insertQueryVariants)
					}
					hists.Get(`insertQueryVariants`).Record(timeutil.Since(start))
				}

				// Parameters
				for i = 0; i < c.parametersPerSession; i++ {
					start := timeutil.Now()
					if _, err := db.ExecContext(ctx, insertQueryParameters,
						sessionID,                           // session_id
						fmt.Sprint(i),                       // id
						string(randutil.RandBytes(rng, 50)), // a
					); err != nil {
						return errors.Wrapf(err, "error running %s", insertQueryParameters)
					}
					hists.Get(`insertQueryParameters`).Record(timeutil.Since(start))
				}

			} else if opRand < c.insertPercent+c.retrievePercent {
				// Retrieve
				// First we have to find a random id to retrieve.
				start0 := timeutil.Now()
				var sessionID string
				for {
					randomSessionID := c.createRandomSessionID(rng, c.pickLocality(rng, c.retrieveLocalPercent))
					err := db.QueryRowContext(ctx, retrieveQuery0, randomSessionID).Scan(&sessionID)
					if err == nil {
						break
					}
					if err == gosql.ErrNoRows {
						continue
					}
					return errors.Wrapf(err, "error running %s", retrieveQuery0)
				}
				hists.Get(`retrieveQuery0`).Record(timeutil.Since(start0))

				start1 := timeutil.Now()
				if _, err := db.ExecContext(ctx, retrieveQuery1, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", retrieveQuery1)
				}
				hists.Get(`retrieveQuery1`).Record(timeutil.Since(start1))

				start2 := timeutil.Now()
				if _, err := db.ExecContext(ctx, retrieveQuery2, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", retrieveQuery2)
				}
				hists.Get(`retrieveQuery2`).Record(timeutil.Since(start2))

				start3 := timeutil.Now()
				if _, err := db.ExecContext(ctx, retrieveQuery3, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", retrieveQuery3)
				}
				hists.Get(`retrieveQuery3`).Record(timeutil.Since(start3))

				start4 := timeutil.Now()
				if _, err := db.ExecContext(ctx, retrieveQuery4, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", retrieveQuery4)
				}
				hists.Get(`retrieveQuery4`).Record(timeutil.Since(start4))

				start5 := timeutil.Now()
				if _, err := db.ExecContext(ctx, retrieveQuery5, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", retrieveQuery5)
				}
				hists.Get(`retrieveQuery5`).Record(timeutil.Since(start5))

				start6 := timeutil.Now()
				if _, err := db.ExecContext(ctx, retrieveQuery6, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", retrieveQuery6)
				}
				hists.Get(`retrieveQuery6`).Record(timeutil.Since(start6))

				hists.Get(`retrieve`).Record(timeutil.Since(start1))
			} else {
				// Update

				// First we have to find a random id to retrieve.
				start0 := timeutil.Now()
				var sessionID string
				for {
					randomSessionID := c.createRandomSessionID(rng, c.pickLocality(rng, c.updateLocalPercent))
					err := db.QueryRowContext(ctx, updateQuery0, randomSessionID).Scan(&sessionID)
					if err == nil {
						break
					}
					if err == gosql.ErrNoRows {
						continue
					}
					return errors.Wrapf(err, "error running %s", updateQuery0)
				}
				hists.Get(`updateQuery0`).Record(timeutil.Since(start0))

				start1 := timeutil.Now()
				if _, err := db.ExecContext(ctx, updateQuery1, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", updateQuery1)
				}
				hists.Get(`updateQuery1`).Record(timeutil.Since(start1))

				start2 := timeutil.Now()
				if _, err := db.ExecContext(ctx, updateQuery2, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", updateQuery2)
				}
				hists.Get(`updateQuery2`).Record(timeutil.Since(start2))

				start3 := timeutil.Now()
				if _, err := db.ExecContext(ctx, updateQuery3, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", updateQuery3)
				}
				hists.Get(`updateQuery3`).Record(timeutil.Since(start3))

				start4 := timeutil.Now()
				if _, err := db.ExecContext(ctx, updateQuery4, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", updateQuery4)
				}
				hists.Get(`updateQuery4`).Record(timeutil.Since(start4))

				start5 := timeutil.Now()
				if _, err := db.ExecContext(ctx, updateQuery5, sessionID); err != nil {
					return errors.Wrapf(err, "error running %s", updateQuery5)
				}
				hists.Get(`updateQuery5`).Record(timeutil.Since(start5))

				start6 := timeutil.Now()
				if _, err := db.ExecContext(ctx, updateQuery6,
					string(randutil.RandBytes(rng, 10)),
					sessionID,
				); err != nil {
					return errors.Wrapf(err, "error running %s", updateQuery6)
				}
				hists.Get(`updateQuery6`).Record(timeutil.Since(start6))

				hists.Get(`update`).Record(timeutil.Since(start1))
			}
			return nil
		}
		ql.WorkerFns = append(ql.WorkerFns, workerFn)
	}
	return ql, nil
}
