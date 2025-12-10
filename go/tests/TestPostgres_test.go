package tests

import (
	"fmt"
	"testing"

	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/orm/persist"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8reflect/go/tests/utils"
	"github.com/saichler/l8srlz/go/serialize/object"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
)

func TestPostgres(t *testing.T) {
	db := openDBConection()
	defer cleanup(db)

	before1 := utils.CreateTestModelInstance(1)
	before2 := utils.CreateTestModelInstance(2)
	res, _ := CreateResources(25000, 1, ifs.Info_Level)

	resp := convert.ConvertTo(object.New(nil, []*testtypes.TestProto{before1, before2}), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	relData := resp.Element().(*types.RelationalData)

	p := persist.NewPostgres(db, res)
	err := p.Write(relData)
	if err != nil {
		Log.Fail(t, "Error writing relationship", err)
		return
	}

	qr, err := interpreter.NewQuery("select * from testproto", res)
	if err != nil {
		Log.Fail(err)
		return
	}
	relData, err = p.Read(qr)
	if err != nil {
		Log.Fail(t, "Error reading relationship", err)
		return
	}

	elems := object.New(nil, relData)
	readObjects := convert.ConvertFrom(elems, res)
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
	relData, err := p.Read(query)
	if err != nil {
		Log.Fail(t, err)
	}
	elems := object.New(nil, relData)
	readObjects := convert.ConvertFrom(elems, res)
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
	relData, err := p.Read(query)
	if err != nil {
		Log.Fail(t, err)
		return
	}
	elems := object.New(nil, relData)
	readObjects := convert.ConvertFrom(elems, res)
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
