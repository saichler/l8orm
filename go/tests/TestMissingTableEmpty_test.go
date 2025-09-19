package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/saichler/l8orm/go/orm/persist"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
	"github.com/saichler/l8reflect/go/tests/utils"
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

		eg2.Resources().Services().RegisterServiceHandlerType(&persist.OrmService{})
		eg2.Resources().Services().Activate(persist.ServiceType, serviceName, 0, eg2.Resources(), eg2, p, &testtypes.TestProto{})
	}

	time.Sleep(time.Second * 2)

	destination := ""

	for i := 1; i <= 4; i++ {
		eg2 := topo.VnicByVnetNum(2, i)
		if i == 4 {
			destination = eg2.Resources().SysConfig().LocalUuid
		}
		hc := health.Health(eg2.Resources())
		hp := hc.Health(eg2.Resources().SysConfig().LocalUuid)
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
	for _, v := range before.MyString2ModelMap {
		v.MySubs = nil
	}

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
