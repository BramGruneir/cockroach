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
// permissions and limitations under the License.

package schemachange

import (
	"context"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/sqlutils"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
)

// TestStressDropDB ...
func TestStressDropDB(t *testing.T) {
	defer leaktest.AfterTest(t)()

	s, db, _ := serverutils.StartServer(t, base.TestServerArgs{})
	defer s.Stopper().Stop(context.Background())
	sqlDB := sqlutils.MakeSQLRunner(db)

	for i := 0; i < 1000; i++ {
		result := sqlDB.Exec(t, `CREATE DATABASE a`)
		t.Log(result)
		result = sqlDB.Exec(t, `CREATE TABLE a.b (id INT PRIMARY KEY)`)
		t.Log(result)
		result = sqlDB.Exec(t, `INSERT INTO a.b VALUES (1), (2), (3), (4), (5), (6), (7), (8), (9), (10)`)
		t.Log(result)
		result = sqlDB.Exec(t, `DROP TABLE a.b`)
		t.Log(result)
		result = sqlDB.Exec(t, `DROP DATABASE a`)
		t.Log(result)
	}
}
