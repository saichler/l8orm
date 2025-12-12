package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8orm/go/orm/persist"
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8reflect/go/tests/utils"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
)

func TestPostgresService(t *testing.T) {
	time.Sleep(time.Second * 2)
	db := openDBConection()
	defer cleanup(db)
	eg1 := topo.VnicByVnetNum(1, 2)
	eg2 := topo.VnicByVnetNum(2, 2)

	/*
		node, _ := eg1.Resources().Introspector().Inspect(&testtypes.TestProto{})
		introspecting.AddPrimaryKeyDecorator(node, "MyString")

		node, _ = eg2.Resources().Introspector().Inspect(&testtypes.TestProto{})
		introspecting.AddPrimaryKeyDecorator(node, "MyString")
	*/
	serviceName := "postgres"
	p := persist.NewPostgres(db, eg2.Resources())
	persist.Activate(serviceName, 0, &testtypes.TestProto{}, &testtypes.TestProtoList{}, eg2, p, nil, "MyString")

	time.Sleep(time.Second)

	hp := health.HealthOf(eg2.Resources().SysConfig().LocalUuid, eg2.Resources())
	sv, ok := hp.Services.ServiceToAreas[serviceName]
	if !ok {
		Log.Fail(t, "Orm service is missing")
		return
	}
	_, ok = sv.Areas[0]
	if !ok {
		Log.Fail(t, "Orm service is missing")
		return
	}

	before := utils.CreateTestModelInstance(5)
	eg1.Resources().Registry().Register(before)

	elems := eg1.ProximityRequest(serviceName, 0, ifs.POST, before, 5)
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	elems = eg1.ProximityRequest(serviceName, 0, ifs.GET, "select * from TestProto", 5)
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	after := elems.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(eg1.Resources(), false, false)

	err := upd.Update(before, after)
	if err != nil {
		Log.Fail(t, "failed updating:", err.Error())
		return
	}

	if len(upd.Changes()) > 0 {
		Log.Fail(t, "Expected no changes")
		return
	}
}

func TestPostgresServiceReplication(t *testing.T) {
	db := openDBConection()
	defer cleanup(db)
	eg1 := topo.VnicByVnetNum(1, 2)

	serviceName := "postgres"

	for i := 1; i <= 4; i++ {
		eg2 := topo.VnicByVnetNum(2, i)

		p := persist.NewPostgres(db, eg2.Resources())
		persist.Activate(serviceName, 0, &testtypes.TestProto{}, &testtypes.TestProtoList{}, eg2, p, nil, "MyString")
	}

	time.Sleep(time.Second * 2)

	destination := ""

	for i := 1; i <= 4; i++ {
		eg2 := topo.VnicByVnetNum(2, i)
		if i == 4 {
			destination = eg2.Resources().SysConfig().LocalUuid
		}
		hp := health.HealthOf(eg2.Resources().SysConfig().LocalUuid, eg2.Resources())
		sv, ok := hp.Services.ServiceToAreas[serviceName]
		if !ok {
			Log.Fail(t, "Orm service is missing")
			return
		}
		_, ok = sv.Areas[0]
		if !ok {
			Log.Fail(t, "Orm service is missing")
			return
		}
	}

	before := utils.CreateTestModelInstance(5)
	eg1.Resources().Registry().Register(before)

	Log.Info("Post")
	elems := eg1.ProximityRequest(serviceName, 0, ifs.POST, before, 5)
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	time.Sleep(time.Second)

	Log.Info("First")

	elems = eg1.ProximityRequest(serviceName, 0, ifs.GET, "select * from TestProto where MyString="+before.MyString, 5)
	if !checkResponse(elems, eg1.Resources(), before, t) {
		return
	}

	Log.Info("Second")

	elems = eg1.Request(destination, serviceName, 0, ifs.GET, "select * from TestProto where MyString="+before.MyString, 5)
	if !checkResponse(elems, eg1.Resources(), before, t) {
		return
	}

	before = utils.CreateTestModelInstance(8)

	Log.Info("Post 2")
	elems = eg1.ProximityRequest(serviceName, 0, ifs.POST, before, 5)
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	time.Sleep(time.Second)

	Log.Info("Third")
	elems = eg1.Request(destination, serviceName, 0, ifs.GET, "select * from TestProto", 5)
	fmt.Println(len(elems.Elements()))
}

func checkResponse(elems ifs.IElements, resources ifs.IResources, before *testtypes.TestProto, t *testing.T) bool {
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return false
	}
	after := elems.Element().(*testtypes.TestProto)
	upd := updating.NewUpdater(resources, false, false)
	err := upd.Update(before, after)
	if err != nil {
		Log.Fail(t, "failed updating:", err.Error())
		return false
	}
	if len(upd.Changes()) > 0 {
		Log.Fail(t, "Expected no changes")
		return false
	}
	return true
}
