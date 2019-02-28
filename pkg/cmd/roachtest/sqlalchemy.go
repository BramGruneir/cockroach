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

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var sqlAlchemyResultRegex = regexp.MustCompile(`(?P<result>PASSED|ERROR|FAILED) (?P<name>.*::.*::.*)`)
var sqlAlchemyReleaseTagRegex = regexp.MustCompile(`^rel_(?P<major>\d+)_(?P<minor>\d+)_(?P<point>\d+)$`)

var sqlAlchemyTestsets = map[string]string{
	"sqlalchemy/a": "Aa0",
	"sqlalchemy/b": "Bb1",
	"sqlalchemy/c": "Cc2",
	"sqlalchemy/d": "Dd3",
	"sqlalchemy/e": "Ee4",
	"sqlalchemy/f": "Ff5",
	"sqlalchemy/g": "Gg6",
	"sqlalchemy/h": "Hh7",
	"sqlalchemy/i": "Ii8",
	"sqlalchemy/j": "Jj9",
	"sqlalchemy/k": "Kk",
	"sqlalchemy/l": "Ll",
	"sqlalchemy/m": "Mm",
	"sqlalchemy/n": "Nn",
	"sqlalchemy/o": "Oo",
	"sqlalchemy/p": "Pp",
	"sqlalchemy/q": "Qq",
	"sqlalchemy/r": "Rr",
	"sqlalchemy/s": "Ss",
	"sqlalchemy/t": "Tt",
	"sqlalchemy/u": "Uu",
	"sqlalchemy/v": "Vv",
	"sqlalchemy/w": "Ww",
	"sqlalchemy/x": "Xx",
	"sqlalchemy/y": "Yy",
	"sqlalchemy/z": "Zz",
}

var knownTestCrashes = []string{
	"test_join_cache",
	"test_mapper_reset",
	"test_savepoints",
	"test_unicode_warnings",
	"CorrelatedSubqueryTest", // Panic #35437
	"test_autoflush",
}

// This test runs SQLAlchemy's full core test suite against an single cockroach
// node.

func registerSQLAlchemy(r *registry) {
	runSQLAlchemy := func(
		ctx context.Context,
		t *test,
		c *cluster,
	) {
		if c.isLocal() {
			t.Fatal("cannot be run in local mode")
		}

		node := c.Node(1)
		t.Status("setting up cockroach")
		c.Put(ctx, cockroach, "./cockroach", c.All())
		c.Start(ctx, t, c.All())

		db := c.Conn(ctx, c.nodes)
		defer func() {
			_ = db.Close()
		}()

		// Get the testset based on the test name.
		sqlAlchemyTestset, ok := sqlAlchemyTestsets[t.Name()]
		if !ok {
			t.Fatalf("Could not find test set for %s", t.Name())
		}

		// Set the default ttl seconds to 5 sec since we're droping and createing
		// so many tables.
		if _, err := db.Exec(
			`ALTER RANGE default CONFIGURE ZONE USING gc.ttlseconds = 10`,
		); err != nil {
			t.Fatal(err)
		}

		// Turn off automatic stats. Again, since this is create and drop to many
		// tables, this will just add a lot of noise.
		if _, err := db.Exec(
			`SET CLUSTER SETTING sql.stats.automatic_collection.enabled = false;`,
		); err != nil {
			t.Fatal(err)
		}

		// Create a dbs for better isolation and convert the list of testsets into
		// an array.
		var remainingTestsets []string
		for _, l := range sqlAlchemyTestset {
			letter := string(l)
			remainingTestsets = append(remainingTestsets, letter)
			if unicode.IsUpper(l) {
				letter = strings.ToLower(letter + letter)
			}
			if _, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS db%s", letter)); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE db%s", letter)); err != nil {
				t.Fatal(err)
			}
		}

		t.Status("cloning SQL Alchemy and installing prerequisites")
		latestTag, err := repeatGetLatestTag(
			ctx, c, "sqlalchemy", "sqlalchemy", sqlAlchemyReleaseTagRegex,
		)
		if err != nil {
			t.Fatal(err)
		}
		c.l.Printf("Latest SQLAlchemy release is %s.", latestTag)

		if err := repeatRunE(ctx, c, node, "update apt-get", `sudo apt-get -qq update`); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			c,
			node,
			"install dependencies",
			`sudo apt-get -qq install make python3 libpq-dev python-dev gcc`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			c,
			node,
			"download pip",
			`curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			c,
			node,
			"install pip",
			`sudo python3 get-pip.py`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			c,
			node,
			"install pytest",
			`sudo pip install pytest -q`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			c,
			node,
			"install psycopg2",
			`sudo pip install psycopg2-binary -q`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx, c, node, "remove old SQLAlchemy", `sudo rm -rf /mnt/data1/sqlalchemy`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatGitCloneE(
			ctx,
			c,
			"https://github.com/sqlalchemy/sqlalchemy.git",
			"/mnt/data1/sqlalchemy",
			latestTag,
			node,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx,
			c,
			node,
			"patch base.py",
			sqlAlchemyBasePatchCmd,
		); err != nil {
			t.Fatal(err)
		}

		// Check the version of Cockroach and find the correct blacklist.
		version, err := fetchCockroachVersion(ctx, c, c.Node(1)[0])
		if err != nil {
			t.Fatal(err)
		}
		// Prepend the version with the test name so we can have different
		// blacklists for each.
		prependedVersion := fmt.Sprintf("%s %s", t.Name(), version)
		blacklistName, expectedFailureList, ignoredlistName, ignoredlist := sqlalchemyBlacklists.getLists(prependedVersion)
		if expectedFailureList == nil {
			t.Fatalf("No blacklist defined for cockroach version %s with test set %s -- %s",
				version, t.Name(), prependedVersion,
			)
		}
		if ignoredlist == nil {
			t.Fatalf("No ignorelist defined for cockroach version %s", version)
		}
		c.l.Printf("Running cockroach version %s, using blacklist %s, using ignoredlist %s",
			version, blacklistName, ignoredlistName)

		t.Status("running SQL Alchemy test suite")
		var excludedTests string
		for _, badTest := range knownTestCrashes {
			excludedTests += " and not " + badTest
		}

		runTestCommand := func(testset string) string {
			dbName := testset
			if unicode.IsUpper(rune(testset[0])) {
				dbName = strings.ToLower(dbName + dbName)
			}
			return fmt.Sprintf("cd /mnt/data1/sqlalchemy/ && "+
				"py.test --dburi=\"postgresql://root@localhost:26257/db%s?sslmode=disable\" "+
				"--maxfail=10000 -rfEsxXpP "+
				"-k \"test_%s%s\" ", dbName, testset, excludedTests)
		}

		// Find all the failed and errored tests.
		var failUnexpectedCount, failExpectedCount, ignoredCount int
		var passUnexpectedCount, passExpectedCount, notRunCount int
		// Put all the results in a giant map of [testname]result
		results := make(map[string]string)
		// Current failures are any tests that reported as failed, regardless of if
		// they were expected or not.
		var currentFailures, allTests []string
		runTests := make(map[string]struct{})

		getNextTestset := func() string {
			if t.Failed() || ctx.Err() != nil {
				return ""
			}
			if len(remainingTestsets) == 0 {
				return ""
			}
			testset := remainingTestsets[0]
			remainingTestsets = remainingTestsets[1:]
			return testset
		}

		collateResults := func(testset string, rawResults []byte) {
			tempResults := make(map[string]bool) // true for pass, false for failure
			scanner := bufio.NewScanner(bytes.NewReader(rawResults))
			for scanner.Scan() {
				match := sqlAlchemyResultRegex.FindStringSubmatch(scanner.Text())
				if match != nil {
					groups := map[string]string{}
					for i, name := range match {
						groups[sqlAlchemyResultRegex.SubexpNames()[i]] = name
					}
					// When a test fails the teardown, it show up as both a pass and a
					// fail.  This is confusing, but luckily, the failure result is
					// reported before the pass. So if a duplicate shows up, it's because
					// of a failure to teardown. For now, it makes sense to ignore these
					// teardown failures.
					test := fmt.Sprintf("%s--%s", groups["name"], testset)
					pass := groups["result"] == "PASSED"
					tempResults[test] = pass
				}
			}
			for test, pass := range tempResults {
				runTests[test] = struct{}{}
				allTests = append(allTests, test)

				ignoredIssue, expectedIgnored := ignoredlist[test]
				issue, expectedFailure := expectedFailureList[test]
				switch {
				case expectedIgnored:
					results[test] = fmt.Sprintf("--- SKIP: %s due to %s (expected)", test, ignoredIssue)
					ignoredCount++
				case pass && !expectedFailure:
					results[test] = fmt.Sprintf("--- PASS: %s (expected)", test)
					passExpectedCount++
				case pass && expectedFailure:
					results[test] = fmt.Sprintf("--- PASS: %s - %s (unexpected)",
						test, maybeAddGithubLink(issue),
					)
					passUnexpectedCount++
				case !pass && expectedFailure:
					results[test] = fmt.Sprintf("--- FAIL: %s - %s (expected)",
						test, maybeAddGithubLink(issue),
					)
					failExpectedCount++
					currentFailures = append(currentFailures, test)
				case !pass && !expectedFailure:
					results[test] = fmt.Sprintf("--- FAIL: %s (unexpected)", test)
					failUnexpectedCount++
					currentFailures = append(currentFailures, test)
				}
			}
		}

		for testset := getNextTestset(); len(testset) > 0; testset = getNextTestset() {
			t.Status(fmt.Sprintf("running SQL Alchemy test suite for tests %s", testset))
			c.l.Printf("running SQL Alchemy test suite for tests %s", testset)
			rawResults, _ := c.RunWithBuffer(ctx, t.l, node, runTestCommand(testset))
			c.l.Printf("Test Results for tests %s:\n%s", testset, rawResults)

			if t.Failed() || ctx.Err() != nil {
				return
			}

			c.l.Printf("collating the test results for tests %s", testset)
			collateResults(testset, rawResults)
			c.l.Printf("-- Remaining test sets: %v", remainingTestsets)
		}

		t.Status("collating all test results")

		// Collect all the tests that were not run.
		for test, issue := range expectedFailureList {
			if _, ok := runTests[test]; ok {
				continue
			}
			allTests = append(allTests, test)
			results[test] = fmt.Sprintf("--- FAIL: %s - %s (not run)", test, maybeAddGithubLink(issue))
			notRunCount++
		}

		// Log all the test results. We re-order the tests alphabetically here
		// to ensure that we're doing .
		sort.Strings(allTests)
		for _, test := range allTests {
			result, ok := results[test]
			if !ok {
				t.Fatalf("can't find %s in test result list", test)
			}
			t.l.Printf("%s\n", result)
		}

		t.l.Printf("------------------------\n")

		var bResults strings.Builder
		fmt.Fprintf(&bResults, "Tests run on Cockroach %s\n", version)
		fmt.Fprintf(&bResults, "Tests run against SQL Alchemy %s\n", latestTag)
		fmt.Fprintf(&bResults, "%d Total Tests Run\n",
			passExpectedCount+passUnexpectedCount+failExpectedCount+failUnexpectedCount,
		)

		p := func(msg string, count int) {
			testString := "tests"
			if count == 1 {
				testString = "test"
			}
			fmt.Fprintf(&bResults, "%d %s %s\n", count, testString, msg)
		}
		p("passed", passUnexpectedCount+passExpectedCount)
		p("failed", failUnexpectedCount+failExpectedCount)
		p("ignored", ignoredCount)
		p("passed unexpectedly", passUnexpectedCount)
		p("failed unexpectedly", failUnexpectedCount)
		p("expected failed but not run", notRunCount)

		fmt.Fprintf(&bResults, "For a full summary look at the sqlalchemy artifacts. \n")
		t.l.Printf("%s\n", bResults.String())
		t.l.Printf("------------------------\n")

		if failUnexpectedCount > 0 || passUnexpectedCount > 0 || notRunCount > 0 {
			// Create a new blacklist so we can easily update this test.
			sort.Strings(currentFailures)
			var b strings.Builder
			fmt.Fprintf(&b, "Here is new SQL Alchemy blacklist that can be used to update the test:\n\n")
			fmt.Fprintf(&b, "var %s = blacklist{\n", blacklistName)
			for _, test := range currentFailures {
				issue := expectedFailureList[test]
				if len(issue) == 0 {
					issue = "unknown"
				}
				fmt.Fprintf(&b, "  \"%s\": \"%s\",\n", test, issue)
			}
			fmt.Fprintf(&b, "}\n\n")
			t.l.Printf("\n\n%s\n\n", b.String())
			t.l.Printf("------------------------\n")
			t.Fatalf("\n%s\nAn updated blacklist (%s) is available in the logs\n",
				bResults.String(),
				blacklistName,
			)
		}
	}

	// Sadly, #35480 causes the cluster to be unstable after too many create
	// tables, drop tables. To combat this, but limiting the number of tests
	// per run we can try to minimize the impact of that bug.
	var alphabet = "abcdefghijklmnopqrstuvwxyz"
	for _, letter := range alphabet {
		r.Add(testSpec{
			Name:    fmt.Sprintf("sqlalchemy/%s", string(letter)),
			Cluster: makeClusterSpec(1),
			Run: func(ctx context.Context, t *test, c *cluster) {
				runSQLAlchemy(ctx, t, c)
			},
			MinVersion: "v19.1.0",
		})
	}
}

const sqlAlchemyBasePatchCmd = `cd /mnt/data1/sqlalchemy/ && patch -p1 << EOF
diff --git a/lib/sqlalchemy/dialects/postgresql/base.py b/lib/sqlalchemy/dialects/postgresql/base.py
index ceb624644..06769d254 100644
--- a/lib/sqlalchemy/dialects/postgresql/base.py
+++ b/lib/sqlalchemy/dialects/postgresql/base.py
@@ -2527,7 +2527,9 @@ class PGDialect(default.DefaultDialect):

     def has_schema(self, connection, schema):
         query = (
-            "select nspname from pg_namespace " "where lower(nspname)=:schema"
+            "SELECT schema_name AS nspname "
+            "FROM [SHOW SCHEMAS] "
+            "WHERE lower(schema_name)=:schema"
         )
         cursor = connection.execute(
             sql.text(query).bindparams(
@@ -2546,10 +2548,9 @@ class PGDialect(default.DefaultDialect):
         if schema is None:
             cursor = connection.execute(
                 sql.text(
-                    "select relname from pg_class c join pg_namespace n on "
-                    "n.oid=c.relnamespace where "
-                    "pg_catalog.pg_table_is_visible(c.oid) "
-                    "and relname=:name"
+                    "SELECT table_name AS relname "
+                    "FROM [SHOW TABLES] "
+                    "WHERE table_name=:name"
                 ).bindparams(
                     sql.bindparam(
                         "name",
@@ -2561,9 +2562,9 @@ class PGDialect(default.DefaultDialect):
         else:
             cursor = connection.execute(
                 sql.text(
-                    "select relname from pg_class c join pg_namespace n on "
-                    "n.oid=c.relnamespace where n.nspname=:schema and "
-                    "relname=:name"
+                    "SELECT table_name AS relname "
+                    "FROM [SHOW TABLES FROM :schema] "
+                    "WHERE table_name=:name"
                 ).bindparams(
                     sql.bindparam(
                         "name",
@@ -2583,10 +2584,9 @@ class PGDialect(default.DefaultDialect):
         if schema is None:
             cursor = connection.execute(
                 sql.text(
-                    "SELECT relname FROM pg_class c join pg_namespace n on "
-                    "n.oid=c.relnamespace where relkind='S' and "
-                    "n.nspname=current_schema() "
-                    "and relname=:name"
+                    "SELECT table_name AS relname "
+                    "FROM [SHOW SEQUENCES] "
+                    "WHERE table_name=:name"
                 ).bindparams(
                     sql.bindparam(
                         "name",
@@ -2598,9 +2598,9 @@ class PGDialect(default.DefaultDialect):
         else:
             cursor = connection.execute(
                 sql.text(
-                    "SELECT relname FROM pg_class c join pg_namespace n on "
-                    "n.oid=c.relnamespace where relkind='S' and "
-                    "n.nspname=:schema and relname=:name"
+                    "SELECT table_name AS relname "
+                    "FROM [SHOW SEQUENCES FROM :schema] "
+                    "WHERE table_name=:name"
                 ).bindparams(
                     sql.bindparam(
                         "name",
@@ -2633,7 +2633,6 @@ class PGDialect(default.DefaultDialect):
             SELECT EXISTS (
                 SELECT * FROM pg_catalog.pg_type t
                 WHERE t.typname = :typname
-                AND pg_type_is_visible(t.oid)
                 )
                 """
             query = sql.text(query)
@@ -2653,6 +2652,8 @@ class PGDialect(default.DefaultDialect):

     def _get_server_version_info(self, connection):
         v = connection.execute("select version()").scalar()
+        if v.startswith("CockroachDB"):
+            return 9, 5, None
         m = re.match(
             r".*(?:PostgreSQL|EnterpriseDB) "
             r"(\d+)\.?(\d+)?(?:\.(\d+))?(?:\.\d+)?(?:devel|beta)?",
@@ -2677,7 +2678,7 @@ class PGDialect(default.DefaultDialect):
         if schema is not None:
             schema_where_clause = "n.nspname = :schema"
         else:
-            schema_where_clause = "pg_catalog.pg_table_is_visible(c.oid)"
+            schema_where_clause = "true"
         query = (
             """
             SELECT c.oid
@@ -2708,9 +2709,8 @@ class PGDialect(default.DefaultDialect):
     def get_schema_names(self, connection, **kw):
         result = connection.execute(
             sql.text(
-                "SELECT nspname FROM pg_namespace "
-                "WHERE nspname NOT LIKE 'pg_%' "
-                "ORDER BY nspname"
+                "SELECT schame_name AS nspname "
+                "FROM [SHOW SCHEMAS]"
             ).columns(nspname=sqltypes.Unicode)
         )
         return [name for name, in result]
@@ -2719,9 +2719,8 @@ class PGDialect(default.DefaultDialect):
     def get_table_names(self, connection, schema=None, **kw):
         result = connection.execute(
             sql.text(
-                "SELECT c.relname FROM pg_class c "
-                "JOIN pg_namespace n ON n.oid = c.relnamespace "
-                "WHERE n.nspname = :schema AND c.relkind in ('r', 'p')"
+                "SELECT table_name AS relname"
+                "FROM [SHOW TABLES FROM :schema]"
             ).columns(relname=sqltypes.Unicode),
             schema=schema if schema is not None else self.default_schema_name,
         )
@@ -2731,9 +2730,8 @@ class PGDialect(default.DefaultDialect):
     def _get_foreign_table_names(self, connection, schema=None, **kw):
         result = connection.execute(
             sql.text(
-                "SELECT c.relname FROM pg_class c "
-                "JOIN pg_namespace n ON n.oid = c.relnamespace "
-                "WHERE n.nspname = :schema AND c.relkind = 'f'"
+                "SELECT '' AS relname "
+                "WHERE false"
             ).columns(relname=sqltypes.Unicode),
             schema=schema if schema is not None else self.default_schema_name,
         )
@@ -3432,7 +3430,7 @@ class PGDialect(default.DefaultDialect):
             SELECT t.typname as "name",
                -- no enum defaults in 8.4 at least
                -- t.typdefault as "default",
-               pg_catalog.pg_type_is_visible(t.oid) as "visible",
+               true as "visible",
                n.nspname as "schema",
                e.enumlabel as "label"
             FROM pg_catalog.pg_type t
@@ -3479,7 +3477,7 @@ class PGDialect(default.DefaultDialect):
                pg_catalog.format_type(t.typbasetype, t.typtypmod) as "attype",
                not t.typnotnull as "nullable",
                t.typdefault as "default",
-               pg_catalog.pg_type_is_visible(t.oid) as "visible",
+               true as "visible",
                n.nspname as "schema"
             FROM pg_catalog.pg_type t
                LEFT JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
EOF
`
