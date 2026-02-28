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
	"github.com/saichler/l8orm/go/types/l8orms"
	"testing"

	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8reflect/go/tests/utils"
	"github.com/saichler/l8srlz/go/serialize/object"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
)

// TestPostgres tests direct PostgreSQL write and read operations.
// Verifies that objects survive the complete round-trip through the database.
func TestPostgres(t *testing.T) {
	nic := topo.VnicByVnetNum(2, 2)
	db := openDBConection(nic.Resources())
	clean(db)
	defer cleanup(db)

	before1 := utils.CreateTestModelInstance(1)
	before2 := utils.CreateTestModelInstance(2)
	res, _ := CreateResources(25000, 1, ifs.Info_Level)

	resp := convert.ConvertTo(ifs.POST, object.New(nil, []*testtypes.TestProto{before1, before2}), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	relData := resp.Element().(*l8orms.L8OrmRData)

	p := postgres.NewPostgres(db, res)
	err := p.WriteRelational(ifs.POST, relData)
	if err != nil {
		Log.Fail(t, "Error writing relationship", err)
		return
	}

	qr, err := interpreter.NewQuery("select * from testproto", res)
	if err != nil {
		Log.Fail(err)
		return
	}
	relData, _, err = p.ReadRelational(qr)
	if err != nil {
		Log.Fail(t, "Error reading relationship", err)
		return
	}

	elems := object.New(nil, relData)
	readObjects := convert.ConvertFrom(elems, nil, res)
	if readObjects != nil && readObjects.Error() != nil {
		Log.Fail(t, "Error reading relationship", readObjects.Error())
		return
	}
	if len(readObjects.Elements()) != 2 {
		Log.Fail(t, "Expected 2 elements:", len(readObjects.Elements()))
		return
	}

	var after1 *testtypes.TestProto
	var after2 *testtypes.TestProto
	for _, obj := range readObjects.Elements() {
		tp := obj.(*testtypes.TestProto)
		if tp.MyString == before1.MyString {
			after1 = tp
		} else if tp.MyString == before2.MyString {
			after2 = tp
		}
	}

	upd := updating.NewUpdater(res, true, true)
	upd.Update(before1, after1)
	if len(upd.Changes()) > 0 {
		for _, chg := range upd.Changes() {
			fmt.Println(chg.PropertyId())
		}
		panic("")
		Log.Fail(t, "Expected no changes in instance 1")
		return
	}

	upd = updating.NewUpdater(res, true, true)
	upd.Update(before2, after2)
	if len(upd.Changes()) > 0 {
		Log.Fail(t, "Expected no changes in instance 2")
		return
	}
}

// TestPostgresFields tests selective field projection in queries.
// Verifies that only requested fields are populated in results.
func TestPostgresFields(t *testing.T) {
	res, _ := CreateResources(25000, 1, ifs.Info_Level)
	ok, db, p := writeRecords(100, res, t)
	defer cleanup(db)
	if !ok {
		return
	}
	qr, err := object.NewQuery("select mystring from testproto", res)
	if err != nil {
		Log.Fail(t, "Error creating query", err)
		return
	}
	query, _ := qr.Query(res)
	relData, _, err := p.ReadRelational(query)
	if err != nil {
		Log.Fail(t, err)
	}
	elems := object.New(nil, relData)
	readObjects := convert.ConvertFrom(elems, nil, res)
	if readObjects != nil && readObjects.Error() != nil {
		Log.Fail(t, "Error reading relationship", readObjects.Error())
		return
	}
	if len(readObjects.Elements()) != 100 {
		Log.Fail(t, "Expected 100 elements")
		return
	}
	for _, elem := range readObjects.Elements() {
		tp := elem.(*testtypes.TestProto)
		if tp.MyString == "" {
			Log.Fail(t, "expected mystring to not be blank ")
			return
		}
		if tp.MyInt32 != 0 {
			Log.Fail(t, "expected myint32 to be 0")
			return
		}
	}
}

// TestPostgresCriteria tests query filtering with WHERE criteria.
// Verifies that criteria are correctly translated to SQL and filter results.
func TestPostgresCriteria(t *testing.T) {
	res, _ := CreateResources(25000, 1, ifs.Info_Level)
	ok, db, p := writeRecords(100, res, t)
	defer cleanup(db)
	if !ok {
		return
	}
	value := CreateTestModelInstance(53).MyString
	qr, err := object.NewQuery("select mystring from testproto where mystring="+value, res)
	if err != nil {
		Log.Fail(t, "Error creating query", err)
		return
	}
	query, _ := qr.Query(res)
	relData, _, err := p.ReadRelational(query)
	if err != nil {
		Log.Fail(t, err)
		return
	}
	elems := object.New(nil, relData)
	readObjects := convert.ConvertFrom(elems, nil, res)
	if readObjects != nil && readObjects.Error() != nil {
		Log.Fail(t, "Error reading relationship", readObjects.Error())
		return
	}
	if len(readObjects.Elements()) != 1 {
		Log.Fail(t, "Expected 1 elements")
		return
	}
	for _, elem := range readObjects.Elements() {
		tp := elem.(*testtypes.TestProto)
		if tp.MyString == "" {
			Log.Fail("expected mystring to not be blank ")
			return
		}
		if tp.MyInt32 != 0 {
			Log.Fail("expected myint32 to be 0")
			return
		}
	}
}
