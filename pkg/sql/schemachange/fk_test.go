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
	"fmt"
	"sync"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/sqlutils"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
)

// TestStressFKTables ...
func TestStressFKTables(t *testing.T) {
	defer leaktest.AfterTest(t)()

	s, db, _ := serverutils.StartServer(t, base.TestServerArgs{})
	defer s.Stopper().Stop(context.Background())
	sqlDB := sqlutils.MakeSQLRunner(db)

	_ = sqlDB.Exec(t, `DROP TABLE IF EXISTS a`)
	_ = sqlDB.Exec(t, `CREATE TABLE a ( id INT PRIMARY KEY )`)

	fmt.Println("*********** wooo")

	var wg sync.WaitGroup
	createReferencingTable := func(n int) {
		defer wg.Done()
		res := sqlDB.Exec(t, fmt.Sprintf("CREATE TABLE b%d ( id INT PRIMARY KEY REFERENCES a )", n))
		t.Log(res)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go createReferencingTable(i)
	}

	wg.Wait()
}
