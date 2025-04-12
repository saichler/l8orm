package tests

import (
	"github.com/saichler/l8orm/go/orm/persist"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/layer8/go/overlay/health"
	"github.com/saichler/reflect/go/reflect/introspecting"
	"github.com/saichler/reflect/go/reflect/updating"
	"github.com/saichler/reflect/go/tests/utils"
	"github.com/saichler/types/go/common"
	"github.com/saichler/types/go/testtypes"
	"testing"
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

	upd := updating.NewUpdater(eg1.Resources().Introspector(), false)

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
