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

// Package tests provides integration and unit tests for the L8 ORM system.
// Tests cover object-relational conversion, PostgreSQL persistence operations,
// service integration, pagination, and distributed service mesh scenarios.
package tests

import (
	"database/sql"
	"fmt"
	"github.com/saichler/l8orm/go/orm/plugins/postgres"
	"github.com/saichler/l8orm/go/types/l8orms"
	"testing"

	_ "github.com/lib/pq"
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8reflect/go/tests/utils"
	"github.com/saichler/l8srlz/go/serialize/object"
	. "github.com/saichler/l8test/go/infra/t_resources"
	. "github.com/saichler/l8test/go/infra/t_topology"
	. "github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
)

// topo holds the test network topology for distributed tests.
var topo *TestTopology

// init sets the log level for tests.
func init() {
	Log.SetLogLevel(Trace_Level)
}

// setup initializes the test environment.
func setup() {
	setupTopology()
}

// tear cleans up the test environment.
func tear() {
	shutdownTopology()
}

// reset resets test handlers between test cases.
func reset(name string) {
	Log.Info("*** ", name, " end ***")
	topo.ResetHandlers()
}

// setupTopology creates the test network topology with 4 nodes across 3 virtual networks.
func setupTopology() {
	topo = NewTestTopology(4, []int{20000, 30000, 40000}, Info_Level)
}

// shutdownTopology tears down the test network topology.
func shutdownTopology() {
	topo.Shutdown()
}

// openDBConection creates a connection to the local PostgreSQL test database.
func openDBConection() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		"127.0.0.1", 5432, "erp", "abcAdmin", "erp")
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	return db
}

// cleanup closes the database connection after cleaning up test tables.
func cleanup(db *sql.DB) {
	defer db.Close()
	clean(db)
}

// clean drops all test tables to reset the database state between tests.
func clean(db *sql.DB) {
	_, e := db.Exec("drop table testproto;")
	if e != nil {
		Log.Error(e)
	}
	_, e = db.Exec("drop table testprotosub;")
	if e != nil {
		Log.Error(e)
	}
	_, e = db.Exec("drop table testprotosubsub;")
	if e != nil {
		Log.Error(e)
	}
}

// writeRecords is a helper that creates test records and writes them to PostgreSQL.
// Returns the success status, database connection, and Postgres instance.
func writeRecords(size int, res IResources, t *testing.T) (bool, *sql.DB, *postgres.Postgres) {
	db := openDBConection()
	recs := make([]*testtypes.TestProto, size)
	for i := 0; i < 100; i++ {
		recs[i] = utils.CreateTestModelInstance(i)
	}

	resp := convert.ConvertTo(POST, object.New(nil, recs), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return false, nil, nil
	}

	relData := resp.Element().(*l8orms.L8OrmRData)

	p := postgres.NewPostgres(db, res)
	err := p.WriteRelational(POST, relData)
	if err != nil {
		Log.Fail(t, "Error writing relationship", err)
		return false, nil, nil
	}
	return true, db, p
}
