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
	"testing"
	"time"

	"github.com/saichler/l8orm/go/orm/plugins/postgres"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
)

// TestTSDBWriteAndRead tests basic TSDB write and read operations.
// Writes notifications for a property and reads them back within a time range.
func TestTSDBWriteAndRead(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTsdb(db)
	defer cleanup(db)

	tsdb := postgres.NewTsdb(db, false)
	defer tsdb.Close()

	now := time.Now().Unix()
	propertyId := "device-001.cpu"

	// Write 5 data points
	notifications := make([]*l8notify.L8TSDBNotification, 5)
	for i := 0; i < 5; i++ {
		notifications[i] = &l8notify.L8TSDBNotification{
			PropertyId: propertyId,
			Point: &l8api.L8TimeSeriesPoint{
				Stamp: now + int64(i*60),
				Value: float64(50 + i*5),
			},
		}
	}

	err := tsdb.AddTSDB(notifications)
	if err != nil {
		Log.Fail(t, "AddTSDB failed:", err)
		return
	}

	// Read back with a range covering all points
	points, err := tsdb.GetTSDB(propertyId, now-1, now+int64(5*60))
	if err != nil {
		Log.Fail(t, "GetTSDB failed:", err)
		return
	}

	if len(points) != 5 {
		Log.Fail(t, "Expected 5 points, got:", len(points))
		return
	}

	// Verify values are correct
	for i, p := range points {
		expected := float64(50 + i*5)
		if p.Value != expected {
			Log.Fail(t, "Point", i, "expected value", expected, "got", p.Value)
			return
		}
	}
}

// TestTSDBMultipleProperties tests TSDB with multiple property IDs.
// Verifies that reads are properly filtered by property ID.
func TestTSDBMultipleProperties(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTsdb(db)
	defer cleanup(db)

	tsdb := postgres.NewTsdb(db, false)
	defer tsdb.Close()

	now := time.Now().Unix()

	// Write points for two different properties
	notifications := []*l8notify.L8TSDBNotification{
		{PropertyId: "device-001.cpu", Point: &l8api.L8TimeSeriesPoint{Stamp: now, Value: 80.0}},
		{PropertyId: "device-001.memory", Point: &l8api.L8TimeSeriesPoint{Stamp: now, Value: 60.0}},
		{PropertyId: "device-001.cpu", Point: &l8api.L8TimeSeriesPoint{Stamp: now + 60, Value: 85.0}},
		{PropertyId: "device-001.memory", Point: &l8api.L8TimeSeriesPoint{Stamp: now + 60, Value: 65.0}},
	}

	err := tsdb.AddTSDB(notifications)
	if err != nil {
		Log.Fail(t, "AddTSDB failed:", err)
		return
	}

	// Read only CPU points
	cpuPoints, err := tsdb.GetTSDB("device-001.cpu", now-1, now+120)
	if err != nil {
		Log.Fail(t, "GetTSDB cpu failed:", err)
		return
	}
	if len(cpuPoints) != 2 {
		Log.Fail(t, "Expected 2 cpu points, got:", len(cpuPoints))
		return
	}

	// Read only memory points
	memPoints, err := tsdb.GetTSDB("device-001.memory", now-1, now+120)
	if err != nil {
		Log.Fail(t, "GetTSDB memory failed:", err)
		return
	}
	if len(memPoints) != 2 {
		Log.Fail(t, "Expected 2 memory points, got:", len(memPoints))
		return
	}
}

// TestTSDBTimeRange tests that time range filtering works correctly.
// Writes points across a range and reads a subset.
func TestTSDBTimeRange(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTsdb(db)
	defer cleanup(db)

	tsdb := postgres.NewTsdb(db, false)
	defer tsdb.Close()

	now := time.Now().Unix()
	propertyId := "device-002.temp"

	// Write 10 points, one per minute
	notifications := make([]*l8notify.L8TSDBNotification, 10)
	for i := 0; i < 10; i++ {
		notifications[i] = &l8notify.L8TSDBNotification{
			PropertyId: propertyId,
			Point: &l8api.L8TimeSeriesPoint{
				Stamp: now + int64(i*60),
				Value: float64(20 + i),
			},
		}
	}

	err := tsdb.AddTSDB(notifications)
	if err != nil {
		Log.Fail(t, "AddTSDB failed:", err)
		return
	}

	// Read only the middle 5 points (minutes 3-7)
	start := now + int64(3*60)
	end := now + int64(7*60)
	points, err := tsdb.GetTSDB(propertyId, start, end)
	if err != nil {
		Log.Fail(t, "GetTSDB failed:", err)
		return
	}

	if len(points) != 5 {
		Log.Fail(t, "Expected 5 points in range, got:", len(points))
		return
	}
}

// TestTSDBEmptyResult tests that querying a non-existent property returns empty.
func TestTSDBEmptyResult(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTsdb(db)
	defer cleanup(db)

	tsdb := postgres.NewTsdb(db, false)
	defer tsdb.Close()

	// Write an empty batch to trigger table creation
	err := tsdb.AddTSDB([]*l8notify.L8TSDBNotification{})
	if err != nil {
		Log.Fail(t, "AddTSDB empty failed:", err)
		return
	}

	now := time.Now().Unix()
	points, err := tsdb.GetTSDB("nonexistent.prop", now-100, now+100)
	if err != nil {
		Log.Fail(t, "GetTSDB failed:", err)
		return
	}
	if len(points) != 0 {
		Log.Fail(t, "Expected 0 points, got:", len(points))
		return
	}
}

// TestTSDBNilNotifications tests that nil notifications are handled gracefully.
func TestTSDBNilNotifications(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	cleanTsdb(db)
	defer cleanup(db)

	tsdb := postgres.NewTsdb(db, false)
	defer tsdb.Close()

	// Mix of valid and nil notifications
	notifications := []*l8notify.L8TSDBNotification{
		{PropertyId: "device-003.cpu", Point: &l8api.L8TimeSeriesPoint{Stamp: time.Now().Unix(), Value: 42.0}},
		nil,
		{PropertyId: "device-003.cpu", Point: nil},
		{PropertyId: "device-003.cpu", Point: &l8api.L8TimeSeriesPoint{Stamp: time.Now().Unix() + 60, Value: 43.0}},
	}

	err := tsdb.AddTSDB(notifications)
	if err != nil {
		Log.Fail(t, "AddTSDB with nils failed:", err)
		return
	}

	points, err := tsdb.GetTSDB("device-003.cpu", time.Now().Unix()-1, time.Now().Unix()+120)
	if err != nil {
		Log.Fail(t, "GetTSDB failed:", err)
		return
	}
	if len(points) != 2 {
		Log.Fail(t, "Expected 2 valid points, got:", len(points))
		return
	}
}
