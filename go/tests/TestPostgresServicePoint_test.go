package tests

import (
	"fmt"
	"github.com/saichler/l8orm/go/orm/persist"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/layer8/go/overlay/health"
	"github.com/saichler/reflect/go/reflect/introspecting"
	"github.com/saichler/reflect/go/reflect/updating"
	"github.com/saichler/reflect/go/tests/utils"
	"github.com/saichler/types/go/common"
	"github.com/saichler/types/go/testtypes"
	"testing"
	"time"
)

func TestPostgresServicePoint(t *testing.T) {
	db := openDBConection()
	defer cleanup(db)
	eg1 := topo.VnicByVnetNum(1, 2)
	eg2 := topo.VnicByVnetNum(2, 2)

	node, _ := eg1.Resources().Introspector().Inspect(&testtypes.TestProto{})
	introspecting.AddPrimaryKeyDecorator(node, "MyString")

	node, _ = eg2.Resources().Introspector().Inspect(&testtypes.TestProto{})
	introspecting.AddPrimaryKeyDecorator(node, "MyString")

	serviceName := "postgres"
	p := persist.NewPostgres(db, eg2.Resources())

	eg2.Resources().ServicePoints().AddServicePointType(&persist.OrmServicePoint{})
	eg2.Resources().ServicePoints().Activate(persist.ServicePointType, serviceName, 0, eg2.Resources(), eg2, p)

	time.Sleep(time.Second)

	hc := health.Health(eg2.Resources())
	hp := hc.HealthPoint(eg2.Resources().SysConfig().LocalUuid)
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

	elems := eg1.SingleRequest(serviceName, 0, common.POST, before)
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	elems = eg1.SingleRequest(serviceName, 0, common.GET, "select * from TestProto")
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	after := elems.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(eg1.Resources().Introspector(), false, false)

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

func TestPostgresServicePointReplication(t *testing.T) {
	db := openDBConection()
	defer cleanup(db)
	eg1 := topo.VnicByVnetNum(1, 2)

	node, _ := eg1.Resources().Introspector().Inspect(&testtypes.TestProto{})
	introspecting.AddPrimaryKeyDecorator(node, "MyString")

	serviceName := "postgres"

	for i := 1; i <= 4; i++ {
		eg2 := topo.VnicByVnetNum(2, i)
		node, _ = eg2.Resources().Introspector().Inspect(&testtypes.TestProto{})
		introspecting.AddPrimaryKeyDecorator(node, "MyString")

		p := persist.NewPostgres(db, eg2.Resources())

		eg2.Resources().ServicePoints().AddServicePointType(&persist.OrmServicePoint{})
		eg2.Resources().ServicePoints().Activate(persist.ServicePointType, serviceName, 0, eg2.Resources(), eg2, p)
	}

	time.Sleep(time.Second * 2)

	destination := ""

	for i := 1; i <= 4; i++ {
		eg2 := topo.VnicByVnetNum(2, i)
		if i == 4 {
			destination = eg2.Resources().SysConfig().LocalUuid
		}
		hc := health.Health(eg2.Resources())
		hp := hc.HealthPoint(eg2.Resources().SysConfig().LocalUuid)
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
	elems := eg1.SingleRequest(serviceName, 0, common.POST, before)
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	time.Sleep(time.Second)

	Log.Info("First")

	elems = eg1.SingleRequest(serviceName, 0, common.GET, "select * from TestProto where MyString="+before.MyString)
	if !checkResponse(elems, eg1.Resources(), before, t) {
		return
	}

	Log.Info("Second")

	elems = eg1.Request(destination, serviceName, 0, common.GET, "select * from TestProto where MyString="+before.MyString)
	if !checkResponse(elems, eg1.Resources(), before, t) {
		return
	}

	before = utils.CreateTestModelInstance(8)

	Log.Info("Post 2")
	elems = eg1.SingleRequest(serviceName, 0, common.POST, before)
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return
	}

	time.Sleep(time.Second)

	Log.Info("Third")
	elems = eg1.Request(destination, serviceName, 0, common.GET, "select * from TestProto")
	fmt.Println(len(elems.Elements()))
}

func checkResponse(elems common.IElements, resources common.IResources, before *testtypes.TestProto, t *testing.T) bool {
	if elems.Error() != nil {
		Log.Fail(t, elems.Error())
		return false
	}
	after := elems.Element().(*testtypes.TestProto)
	upd := updating.NewUpdater(resources.Introspector(), false, false)
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
