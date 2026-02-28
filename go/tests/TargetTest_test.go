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
	"fmt"
	"github.com/saichler/l8orm/go/orm/plugins/postgres"
	"github.com/saichler/l8pollaris/go/pollaris/targets"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/probler/go/prob/common"
	"github.com/saichler/probler/go/prob/common/creates"
	"strconv"
	"testing"
)

// TestSingleTarget tests basic CRUD operations on a single target device.
// Verifies that a device can be created, queried by ID, and filtered.
func TestSingleTarget(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	nic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(&l8tpollaris.L8PTarget{}, "TargetId")
	device := creates.CreateDevice("60.50."+strconv.Itoa(40)+"."+strconv.Itoa(1), common.NetworkDevice_Links_ID, "sim")
	targets.Activate("postgres", "probler", nic)

	ts, _ := targets.Targets(nic)
	ts.Post(object.New(nil, device), nic)

	q, _ := object.NewQuery("select * from L8PTarget where targetid="+device.TargetId, nic.Resources())
	resp := ts.Get(q, nic)
	qDevice := resp.Element().(*l8tpollaris.L8PTarget)
	if len(qDevice.Hosts) == 0 {
		nic.Resources().Logger().Fail(t, "Query No Hosts")
		fmt.Println("Query No Hosts")
		return
	}

	filter := &l8tpollaris.L8PTarget{TargetId: device.TargetId}
	resp = ts.Get(object.New(nil, filter), nic)

	cDevice := resp.Element().(*l8tpollaris.L8PTarget)
	if len(cDevice.Hosts) == 0 {
		nic.Resources().Logger().Fail(t, "Filter No Hosts")
		fmt.Println("Filter No Hosts")
		return
	}
}

// TestTarget tests bulk CRUD operations with 100 target devices.
// Verifies POST, PATCH, and DELETE operations work correctly on multiple records.
func TestTarget(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	nic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(&l8tpollaris.L8PTarget{}, "TargetId")
	size := 100
	devices := make([]*l8tpollaris.L8PTarget, size)
	ip := 1
	sub := 40
	for i := 0; i < size; i++ {
		device := creates.CreateDevice("60.50."+strconv.Itoa(sub)+"."+strconv.Itoa(ip), common.NetworkDevice_Links_ID, "sim")
		devices[i] = device
		ip++
		if ip > 254 {
			sub++
			ip = 1
		}
	}

	db := openDBConection(nic.Resources())
	defer cleanup(db)
	// Clean up any leftover target tables from previous test runs
	cleanTargetTables(db)
	p := postgres.NewPostgres(db, nic.Resources())
	err := p.Write(ifs.POST, object.New(nil, devices), nic.Resources())
	if err != nil {
		nic.Resources().Logger().Fail(t, err.Error())
		return
	}

	rows, err := db.Query("select count(*) from L8PTarget where state=1;")
	if err != nil {
		nic.Resources().Logger().Fail(t, err.Error())
		return
	}
	rows.Next()
	count := 0
	rows.Scan(&count)
	if count != size {
		nic.Resources().Logger().Fail(t, "Count ", count, " is not equal to size ", size)
		return
	}

	devices2 := make([]*l8tpollaris.L8PTarget, size)
	for i, device := range devices {
		device2 := &l8tpollaris.L8PTarget{TargetId: device.TargetId, State: l8tpollaris.L8PTargetState_Up}
		devices2[i] = device2
	}

	err = p.Write(ifs.PATCH, object.New(nil, devices2), nic.Resources())
	if err != nil {
		nic.Resources().Logger().Fail(t, err.Error())
		return
	}

	rows, err = db.Query("select count(*) from L8PTarget where state=2;")
	if err != nil {
		nic.Resources().Logger().Fail(t, err.Error())
		return
	}
	rows.Next()
	count = 0
	rows.Scan(&count)
	if count != size {
		nic.Resources().Logger().Fail(t, "Count 2 ", count, " is not equal to size ", size)
		return
	}

	q, _ := interpreter.NewQuery("select * from L8PTarget", nic.Resources())
	err = p.Delete(q, nic.Resources())
	if err != nil {
		nic.Resources().Logger().Fail(t, err.Error())
	}

	rows, err = db.Query("select count(*) from L8PTarget where state=2;")
	if err != nil {
		nic.Resources().Logger().Fail(t, err.Error())
		return
	}
	rows.Next()
	count = 0
	rows.Scan(&count)
	if count != 0 {
		nic.Resources().Logger().Fail(t, "Count 3 ", count, " is not equal to size ", size)
		return
	}
}

// TestTargetService tests the target service with 10,000 devices and pagination.
// Verifies that the service handles large datasets and returns proper metadata.
func TestTargetService(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	targets.Activate("postgres", "problerdb", nic)
	size := 10000
	devices := make([]*l8tpollaris.L8PTarget, size)
	ip := 1
	sub := 80
	for i := 0; i < size; i++ {
		device := creates.CreateDevice("70.50."+strconv.Itoa(sub)+"."+strconv.Itoa(ip), common.NetworkDevice_Links_ID, "sim")
		devices[i] = device
		ip++
		if ip > 254 {
			sub++
			ip = 1
		}
	}

	ts, _ := targets.Targets(nic)
	resp := ts.Post(object.New(nil, devices), nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Post", resp.Error())
		return
	}

	q, _ := object.NewQuery("select * from l8ptarget", nic.Resources())
	resp = ts.Get(q, nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Get", resp.Error())
		return
	}
	for _, d := range resp.Elements() {
		dev := d.(*l8tpollaris.L8PTarget)
		if dev.TargetId == "" {
			nic.Resources().Logger().Fail(t, "TargetId", dev.TargetId)
			return
		}
	}

	q, _ = object.NewQuery("select * from l8ptarget limit 10 page 0", nic.Resources())
	resp = ts.Get(q, nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Get limit", resp.Error())
		return
	}

	if len(resp.Elements()) != 10 {
		nic.Resources().Logger().Fail(t, "Get limit", len(resp.Elements()))
		return
	}
	list, err := resp.AsList(nic.Resources().Registry())
	if err != nil {
		nic.Resources().Logger().Fail(t, "AsList", resp.Error())
		return
	}
	tlist := list.(*l8tpollaris.L8PTargetList)
	if tlist.Metadata == nil {
		nic.Resources().Logger().Fail(t, "Expected metadata")
		return
	}
	fmt.Println(tlist.Metadata.KeyCount.Counts["Total"])

}
