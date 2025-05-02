package tests

import (
	"fmt"
	"github.com/saichler/l8orm/go/orm/persist"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/layer8/go/overlay/health"
	"github.com/saichler/reflect/go/reflect/introspecting"
	"github.com/saichler/reflect/go/tests/utils"
	"github.com/saichler/types/go/common"
	"github.com/saichler/types/go/testtypes"
	"testing"
	"time"
)

func TestMissingTableEmpty(t *testing.T) {
	db := openDBConection()
	clean(db)
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

	before := utils.CreateTestModelInstance(8)
	before.MySingle.MySubs = nil
	for _, v := range before.MyString2ModelMap {
		v.MySubs = nil
	}

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
	for _, v := range before.MyString2ModelMap {
		v.MySubs = nil
	}

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
