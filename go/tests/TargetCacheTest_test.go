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
	"database/sql"
	"fmt"
	"github.com/saichler/l8orm/go/orm/persist"
	"github.com/saichler/l8orm/go/orm/plugins/postgres"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/probler/go/prob/common"
	"github.com/saichler/probler/go/prob/common/creates"
	"strconv"
	"testing"
)

// activateCachedTargetService registers a cache-enabled ORM service for L8PTarget.
func activateCachedTargetService(serviceName string, db *sql.DB, nic ifs.IVNic) {
	p := postgres.NewPostgres(db, nic.Resources())
	persist.ActivateWithCache(serviceName, 91, &l8tpollaris.L8PTarget{},
		&l8tpollaris.L8PTargetList{}, nic, p, nil, true, "TargetId")
}

// cachedTargetService retrieves the cache-enabled target service handler.
func cachedTargetService(serviceName string, nic ifs.IVNic) (ifs.IServiceHandler, bool) {
	return nic.Resources().Services().ServiceHandler(serviceName, 91)
}

// TestSingleTargetCached tests basic CRUD with cache enabled on a single target device.
func TestSingleTargetCached(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTargetTables(db)
	defer func() { cleanTargetTables(db); db.Close() }()

	serviceName := "cachtgt1"
	activateCachedTargetService(serviceName, db, nic)

	device := creates.CreateDevice("80.50.40.1", common.NetworkDevice_Links_ID, "sim")

	ts, ok := cachedTargetService(serviceName, nic)
	if !ok {
		nic.Resources().Logger().Fail(t, "Service not found")
		return
	}
	resp := ts.Post(object.New(nil, device), nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Post", resp.Error())
		return
	}

	// Query by ID — should be served from cache
	q, _ := object.NewQuery("select * from L8PTarget where targetid="+device.TargetId, nic.Resources())
	resp = ts.Get(q, nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Get query", resp.Error())
		return
	}
	qDevice := resp.Element().(*l8tpollaris.L8PTarget)
	if len(qDevice.Hosts) == 0 {
		nic.Resources().Logger().Fail(t, "Query No Hosts")
		return
	}

	// Filter by primary key — should hit cache
	filter := &l8tpollaris.L8PTarget{TargetId: device.TargetId}
	resp = ts.Get(object.New(nil, filter), nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Get filter", resp.Error())
		return
	}
	cDevice := resp.Element().(*l8tpollaris.L8PTarget)
	if len(cDevice.Hosts) == 0 {
		nic.Resources().Logger().Fail(t, "Filter No Hosts")
		return
	}
}

// TestTargetCached tests bulk CRUD operations with cache enabled on 100 target devices.
func TestTargetCached(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTargetTables(db)
	defer func() { cleanTargetTables(db); db.Close() }()

	serviceName := "cachtgt2"
	activateCachedTargetService(serviceName, db, nic)

	size := 100
	devices := make([]*l8tpollaris.L8PTarget, size)
	ip := 1
	sub := 40
	for i := 0; i < size; i++ {
		device := creates.CreateDevice("80.60."+strconv.Itoa(sub)+"."+strconv.Itoa(ip), common.NetworkDevice_Links_ID, "sim")
		devices[i] = device
		ip++
		if ip > 254 {
			sub++
			ip = 1
		}
	}

	ts, ok := cachedTargetService(serviceName, nic)
	if !ok {
		nic.Resources().Logger().Fail(t, "Service not found")
		return
	}

	// POST — should populate cache
	resp := ts.Post(object.New(nil, devices), nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Post", resp.Error())
		return
	}

	// Verify POST via direct DB count
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

	// PATCH — should update cache
	devices2 := make([]*l8tpollaris.L8PTarget, size)
	for i, device := range devices {
		devices2[i] = &l8tpollaris.L8PTarget{TargetId: device.TargetId, State: l8tpollaris.L8PTargetState_Up}
	}
	resp = ts.Patch(object.New(nil, devices2), nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Patch", resp.Error())
		return
	}

	// Verify PATCH via direct DB count
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

	// GET by filter — should hit cache after PATCH
	filter := &l8tpollaris.L8PTarget{TargetId: devices[0].TargetId}
	resp = ts.Get(object.New(nil, filter), nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Get filter", resp.Error())
		return
	}
	fetched := resp.Element().(*l8tpollaris.L8PTarget)
	if fetched.State != l8tpollaris.L8PTargetState_Up {
		nic.Resources().Logger().Fail(t, "Expected state Up after patch, got", fetched.State)
		return
	}

	// DELETE — should remove from cache and DB
	q, _ := object.NewQuery("select * from L8PTarget", nic.Resources())
	resp = ts.Delete(q, nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Delete", resp.Error())
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
	if count != 0 {
		nic.Resources().Logger().Fail(t, "Count 3 ", count, " is not equal to 0")
		return
	}
}

// TestTargetServiceCached tests the cache-enabled target service with 10,000 devices and pagination.
func TestTargetServiceCached(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTargetTables(db)
	defer func() { cleanTargetTables(db); db.Close() }()

	serviceName := "cachtgt3"
	activateCachedTargetService(serviceName, db, nic)

	size := 10000
	devices := make([]*l8tpollaris.L8PTarget, size)
	ip := 1
	sub := 80
	for i := 0; i < size; i++ {
		device := creates.CreateDevice("80.70."+strconv.Itoa(sub)+"."+strconv.Itoa(ip), common.NetworkDevice_Links_ID, "sim")
		devices[i] = device
		ip++
		if ip > 254 {
			sub++
			ip = 1
		}
	}

	ts, ok := cachedTargetService(serviceName, nic)
	if !ok {
		nic.Resources().Logger().Fail(t, "Service not found")
		return
	}

	resp := ts.Post(object.New(nil, devices), nic)
	if resp.Error() != nil {
		nic.Resources().Logger().Fail(t, "Post", resp.Error())
		return
	}

	// GET all — first call populates cache from DB, subsequent calls should use cache
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

	// GET with pagination — should be served from cache
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
