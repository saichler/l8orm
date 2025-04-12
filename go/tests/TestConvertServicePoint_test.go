package tests

import (
	"github.com/saichler/l8orm/go/orm/convert"
	types2 "github.com/saichler/l8orm/go/types"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/layer8/go/overlay/health"
	"github.com/saichler/reflect/go/reflect/introspecting"
	"github.com/saichler/reflect/go/reflect/updating"
	"github.com/saichler/reflect/go/tests/utils"
	"github.com/saichler/types/go/common"
	"github.com/saichler/types/go/testtypes"
	"github.com/saichler/types/go/types"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	setup()
	m.Run()
	tear()
}

func TestConvertServicePoint(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	nic.Resources().ServicePoints().AddServicePointType(&convert.ConvertServicePoint{})
	nic.Resources().ServicePoints().Activate(convert.ServicePointType, convert.ServiceName, 0, nic.Resources(), nic)

	hc := health.Health(nic.Resources())
	hp := hc.HealthPoint(nic.Resources().SysConfig().LocalUuid)
	hp.Services.ServiceToAreas[convert.ServiceName] = &types.ServiceAreas{}
	hp.Services.ServiceToAreas[convert.ServiceName].Areas = make(map[int32]*types.ServiceAreaInfo)
	hp.Services.ServiceToAreas[convert.ServiceName].Areas[0] = &types.ServiceAreaInfo{}
	nic.Unicast(nic.Resources().SysConfig().RemoteUuid, health.ServiceName, 0, common.PATCH, hp)
	time.Sleep(time.Second)

	before := utils.CreateTestModelInstance(1)
	nic2 := topo.VnicByVnetNum(1, 4)
	resp := nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, common.POST, before)
	if resp.Error() == nil {
		Log.Fail(t, "Expected an error as we did not register the type")
		return
	}

	node, _ := nic.Resources().Introspector().Inspect(before)
	nic2.Resources().Introspector().Inspect(before)
	nic2.Resources().Introspector().Inspect(&types2.RelationalData{})
	introspecting.AddPrimaryKeyDecorator(node, "MyString")

	resp = nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, common.POST, before)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	rlData := resp.Element().(*types2.RelationalData)
	if len(rlData.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	resp = nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, common.GET, rlData)
	if resp.Error() != nil {
		Log.Fail(t, resp.Error().Error())
		return
	}

	after := resp.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(nic.Resources().Introspector(), false)
	upd.Update(before, after)
	if len(upd.Changes()) > 0 {
		Log.Fail(t, "Expected no changes:", len(upd.Changes()))
		return
	}
}

func TestConvertServicePointMulti(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	nic.Resources().ServicePoints().AddServicePointType(&convert.ConvertServicePoint{})
	nic.Resources().ServicePoints().Activate(convert.ServicePointType, convert.ServiceName, 0, nic.Resources(), nic)

	before1 := utils.CreateTestModelInstance(1)
	before2 := utils.CreateTestModelInstance(2)

	//node, _ := nic.Resources().Introspector().Inspect(before1)
	nic2 := topo.VnicByVnetNum(1, 3)
	nic2.Resources().Registry().Register(&types2.RelationalData{})
	//nic2.Resources().Introspector().Inspect(before1)
	//introspecting.AddPrimaryKeyDecorator(node, "MyString")

	resp := nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, common.POST,
		[]*testtypes.TestProto{before1, before2})
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	rlData := resp.Element().(*types2.RelationalData)
	if len(rlData.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	if len(rlData.Tables["TestProto"].InstanceRows[""].AttributeRows[""].Rows) != 2 {
		Log.Fail(t, "Expected 2 instances")
		return
	}

	resp = nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, common.GET, rlData)
	if resp.Error() != nil {
		Log.Fail(t, resp.Error().Error())
		return
	}

	if len(resp.Elements()) != 2 {
		Log.Fail(t, "Expected 2 elements:", len(resp.Elements()))
		return
	}

	after := resp.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(nic.Resources().Introspector(), false)
	upd.Update(before1, after)
	if len(upd.Changes()) > 0 {
		Log.Fail(t, "Expected no changes:", len(upd.Changes()))
		return
	}
}
