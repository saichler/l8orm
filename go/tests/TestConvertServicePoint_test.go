package tests

import (
	"github.com/saichler/l8orm/go/orm/convert"
	types2 "github.com/saichler/l8orm/go/types"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/layer8/go/overlay/health"
	"github.com/saichler/reflect/go/reflect/updating"
	"github.com/saichler/reflect/go/tests/utils"
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
	convert.RegisterConvertCenter(0, nic.Resources())
	hc := health.Health(nic.Resources())
	hp := hc.HealthPoint(nic.Resources().Config().LocalUuid)
	hp.Services.ServiceToAreas[convert.ServiceName] = &types.ServiceAreas{}
	hp.Services.ServiceToAreas[convert.ServiceName].Areas = make(map[int32]*types.ServiceAreaInfo)
	hp.Services.ServiceToAreas[convert.ServiceName].Areas[0] = &types.ServiceAreaInfo{}
	nic.Unicast(nic.Resources().Config().RemoteUuid, health.ServiceName, 0, types.Action_PATCH, hp)
	time.Sleep(time.Second)

	before := utils.CreateTestModelInstance(1)
	nic2 := topo.VnicByVnetNum(1, 4)
	resp := nic2.Request(nic.Resources().Config().LocalUuid, convert.ServiceName, 0, types.Action_POST, before)
	if resp.Error() == nil {
		Log.Fail(t, "Expected an error as we did not register the type")
		return
	}

	nic.Resources().Introspector().Inspect(before)
	nic2.Resources().Introspector().Inspect(before)
	nic2.Resources().Introspector().Inspect(&types2.RelationalData{})

	resp = nic2.Request(nic.Resources().Config().LocalUuid, convert.ServiceName, 0, types.Action_POST, before)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	rlData := resp.Element().(*types2.RelationalData)
	if len(rlData.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	resp = nic2.Request(nic.Resources().Config().LocalUuid, convert.ServiceName, 0, types.Action_GET, rlData)
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
