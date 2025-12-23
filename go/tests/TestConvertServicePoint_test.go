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
	"github.com/saichler/l8orm/go/types/l8orms"
	"testing"

	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8reflect/go/tests/utils"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
)

// TestMain is the test entry point that sets up and tears down the test topology.
func TestMain(m *testing.M) {
	setup()
	m.Run()
	tear()
}

// TestConvertService tests the ConvertService as a distributed Layer 8 service.
// Verifies that conversion operations work correctly across the service mesh.
func TestConvertService(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	sla := ifs.NewServiceLevelAgreement(&convert.ConvertService{}, convert.ServiceName, 0, false, nil)
	nic.Resources().Services().Activate(sla, nic)

	before := utils.CreateTestModelInstance(1)
	nic2 := topo.VnicByVnetNum(1, 4)

	resp := nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, ifs.POST, before, 5)
	if resp.Error() == nil {
		Log.Fail(t, "Expected an error as we did not register the type")
		return
	}

	/*
		node, _ := nic.Resources().Introspector().Inspect(before)
		nic2.Resources().Introspector().Inspect(before)
		nic2.Resources().Introspector().Inspect(&types2.RelationalData{})
		helping.AddPrimaryKeyDecorator(node, "MyString")
	*/
	nic2.Resources().Introspector().Inspect(&l8orms.L8OrmRData{})
	resp = nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, ifs.POST, before, 5)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, "[error]", resp.Error())
		return
	}

	rlData := resp.Element().(*l8orms.L8OrmRData)
	if len(rlData.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	resp = nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, ifs.GET, rlData, 5)
	if resp.Error() != nil {
		Log.Fail(t, resp.Error().Error())
		return
	}

	after := resp.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(nic.Resources(), false, false)
	upd.Update(before, after)
	if len(upd.Changes()) > 0 {
		Log.Fail(t, "Expected no changes:", len(upd.Changes()))
		return
	}
}

// TestConvertServiceMulti tests conversion of multiple objects through the service.
// Verifies distributed conversion handles multiple instances correctly.
func TestConvertServiceMulti(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	sla := ifs.NewServiceLevelAgreement(&convert.ConvertService{}, convert.ServiceName, 0, false, nil)
	nic.Resources().Services().Activate(sla, nic)

	before1 := utils.CreateTestModelInstance(1)
	before2 := utils.CreateTestModelInstance(2)

	//node, _ := nic.Resources().Introspector().Inspect(before1)
	nic2 := topo.VnicByVnetNum(1, 3)
	nic2.Resources().Registry().Register(&l8orms.L8OrmRData{})
	//nic2.Resources().Introspector().Inspect(before1)
	//introspecting.AddPrimaryKeyDecorator(node, "MyString")

	resp := nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, ifs.POST,
		[]*testtypes.TestProto{before1, before2}, 5)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	rlData := resp.Element().(*l8orms.L8OrmRData)
	if len(rlData.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	if len(rlData.Tables["TestProto"].InstanceRows[""].AttributeRows[""].Rows) != 2 {
		Log.Fail(t, "Expected 2 instances")
		return
	}

	resp = nic2.Request(nic.Resources().SysConfig().LocalUuid, convert.ServiceName, 0, ifs.GET, rlData, 5)
	if resp.Error() != nil {
		Log.Fail(t, resp.Error().Error())
		return
	}

	if len(resp.Elements()) != 2 {
		Log.Fail(t, "Expected 2 elements:", len(resp.Elements()))
		return
	}

	after := resp.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(nic.Resources(), false, false)
	if after.MyString == before1.MyString {
		upd.Update(before1, after)
	} else {
		upd.Update(before2, after)
	}

	if len(upd.Changes()) > 0 {
		Log.Fail(t, "Expected no changes:", len(upd.Changes()))
		return
	}
}
