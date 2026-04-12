/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package tests

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/orm/plugins/postgres"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8reflect/go/tests/utils"
	"github.com/saichler/l8srlz/go/serialize/object"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
)

// liveColumns returns the set of column names (lowercased) that Postgres
// reports for the given table via information_schema. Returns nil on error.
func liveColumns(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query(
		"SELECT column_name FROM information_schema.columns WHERE table_name = $1",
		strings.ToLower(tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := make(map[string]bool)
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols[strings.ToLower(c)] = true
	}
	return cols, nil
}

// indexExists returns true if an index with the given name exists on tableName.
func indexExists(db *sql.DB, tableName, indexName string) (bool, error) {
	row := db.QueryRow(
		"SELECT COUNT(*) FROM pg_indexes WHERE tablename = $1 AND indexname = $2",
		strings.ToLower(tableName), strings.ToLower(indexName))
	var n int
	if err := row.Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// writeOneRecord creates a single TestProto instance and writes it via a
// fresh Postgres plugin. Returns the plugin used for the write so callers
// can drive follow-up operations on the same connection.
func writeOneRecord(t *testing.T, db *sql.DB, res ifs.IResources, index int) *postgres.Postgres {
	rec := utils.CreateTestModelInstance(index)
	resp := convert.ConvertTo(ifs.POST, object.New(nil, []*testtypes.TestProto{rec}), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return nil
	}
	relData := resp.Element().(*l8orms.L8OrmRData)
	p := postgres.NewPostgres(db, res)
	if err := p.WriteRelational(ifs.POST, relData); err != nil {
		Log.Fail(t, "Error writing relationship", err)
		return nil
	}
	return p
}

// TestPostgresSchemaMigration_AddSingleColumn simulates schema drift by
// dropping one column from the live testproto table and verifies that the
// next WriteRelational call re-adds it.
func TestPostgresSchemaMigration_AddSingleColumn(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	clean(db)
	defer cleanup(db)

	res, _ := CreateResources(25010, 1, ifs.Info_Level)

	// Step 1: create the table by writing a record.
	if writeOneRecord(t, db, res, 1) == nil {
		return
	}

	// Step 2: drop one scalar column to simulate an older proto schema.
	if _, err := db.Exec("ALTER TABLE testproto DROP COLUMN myInt32;"); err != nil {
		Log.Fail(t, "Failed to drop column: ", err)
		return
	}
	cols, err := liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if cols["myint32"] {
		Log.Fail(t, "Drop column did not take effect")
		return
	}

	// Step 3: a fresh Postgres instance has an empty verifyed cache, so
	// WriteRelational will run verifyTable → migrateTable and re-add the column.
	if writeOneRecord(t, db, res, 2) == nil {
		return
	}

	cols, err = liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if !cols["myint32"] {
		Log.Fail(t, "Migration did not re-add column myInt32. Live columns: ", cols)
		return
	}
}

// TestPostgresSchemaMigration_AddMultipleColumns verifies that multiple
// missing columns of different Postgres types are all added in a single
// migration pass.
func TestPostgresSchemaMigration_AddMultipleColumns(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	clean(db)
	defer cleanup(db)

	res, _ := CreateResources(25011, 1, ifs.Info_Level)

	if writeOneRecord(t, db, res, 1) == nil {
		return
	}

	dropStmts := []string{
		"ALTER TABLE testproto DROP COLUMN myInt32;",
		"ALTER TABLE testproto DROP COLUMN myInt64;",
		"ALTER TABLE testproto DROP COLUMN myFloat32;",
	}
	for _, s := range dropStmts {
		if _, err := db.Exec(s); err != nil {
			Log.Fail(t, "Drop failed: ", err)
			return
		}
	}

	if writeOneRecord(t, db, res, 2) == nil {
		return
	}

	cols, err := liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}
	for _, expected := range []string{"myint32", "myint64", "myfloat32"} {
		if !cols[expected] {
			Log.Fail(t, "Migration did not re-add column ", expected, ". Live columns: ", cols)
			return
		}
	}
}

// TestPostgresSchemaMigration_IdempotentRerun verifies that calling
// verifyTables repeatedly on an up-to-date table is a no-op and never
// returns an error.
func TestPostgresSchemaMigration_IdempotentRerun(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	clean(db)
	defer cleanup(db)

	res, _ := CreateResources(25012, 1, ifs.Info_Level)

	if writeOneRecord(t, db, res, 1) == nil {
		return
	}
	colsBefore, err := liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}

	// Second write with a fresh plugin — triggers verify+migrate on an
	// already-correct table. Must not alter the schema.
	if writeOneRecord(t, db, res, 2) == nil {
		return
	}
	// Third write with yet another fresh plugin — same assertion.
	if writeOneRecord(t, db, res, 3) == nil {
		return
	}

	colsAfter, err := liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if len(colsBefore) != len(colsAfter) {
		Log.Fail(t, "Column count changed on idempotent rerun: before=", len(colsBefore), " after=", len(colsAfter))
		return
	}
	for c := range colsBefore {
		if !colsAfter[c] {
			Log.Fail(t, "Column disappeared on idempotent rerun: ", c)
			return
		}
	}
}

// TestPostgresSchemaMigration_WriteAfterMigration verifies that after a
// migration adds a column back, new rows can be written successfully and
// the restored column is present in the live schema.
func TestPostgresSchemaMigration_WriteAfterMigration(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	clean(db)
	defer cleanup(db)

	res, _ := CreateResources(25013, 1, ifs.Info_Level)

	// Seed one record, drop a column, then seed another record to force
	// migration on the same fresh plugin.
	if writeOneRecord(t, db, res, 1) == nil {
		return
	}
	if _, err := db.Exec("ALTER TABLE testproto DROP COLUMN myInt64;"); err != nil {
		Log.Fail(t, "Drop failed: ", err)
		return
	}
	if writeOneRecord(t, db, res, 2) == nil {
		return
	}

	// Verify the column was restored.
	cols, err := liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if !cols["myint64"] {
		Log.Fail(t, "Migration did not re-add column myInt64. Live columns: ", cols)
		return
	}

	// Verify both rows exist.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM testproto").Scan(&count); err != nil {
		Log.Fail(t, "Count query failed: ", err)
		return
	}
	if count != 2 {
		Log.Fail(t, "Expected 2 rows after migration, got ", count)
		return
	}
}

// TestPostgresSchemaMigration_PreservesExistingColumns verifies that
// migration leaves columns that are already present untouched — no
// duplicate or re-added columns even if the test explicitly triggers a
// migration pass against an up-to-date table.
func TestPostgresSchemaMigration_PreservesExistingColumns(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	clean(db)
	defer cleanup(db)

	res, _ := CreateResources(25014, 1, ifs.Info_Level)

	if writeOneRecord(t, db, res, 1) == nil {
		return
	}
	cols1, err := liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}

	// Fresh plugin → verifyTable sees table exists → migrateTable runs on
	// an already-synchronized schema. This must not add duplicate columns
	// (Postgres would reject that with a "column already exists" error,
	// so any such bug surfaces here).
	if writeOneRecord(t, db, res, 2) == nil {
		return
	}

	cols2, err := liveColumns(db, "testproto")
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if len(cols1) != len(cols2) {
		Log.Fail(t, "Column set changed on no-drift rerun: before=", cols1, " after=", cols2)
		return
	}
}
