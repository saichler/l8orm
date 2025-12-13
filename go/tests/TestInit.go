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

var topo *TestTopology

func init() {
	Log.SetLogLevel(Trace_Level)
}

func setup() {
	setupTopology()
}

func tear() {
	shutdownTopology()
}

func reset(name string) {
	Log.Info("*** ", name, " end ***")
	topo.ResetHandlers()
}

func setupTopology() {
	topo = NewTestTopology(4, []int{20000, 30000, 40000}, Info_Level)
}

func shutdownTopology() {
	topo.Shutdown()
}

func openDBConection() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		"127.0.0.1", 5432, "postgres", "admin", "postgres")
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	return db
}

func cleanup(db *sql.DB) {
	defer db.Close()
	clean(db)
}

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

func writeRecords(size int, res IResources, t *testing.T) (bool, *sql.DB, *postgres.Postgres) {
	db := openDBConection()
	recs := make([]*testtypes.TestProto, size)
	for i := 0; i < 100; i++ {
		recs[i] = utils.CreateTestModelInstance(i)
	}

	resp := convert.ConvertTo(object.New(nil, recs), res)
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
