package tests

import (
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/orm/plugins/postgres"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8reflect/go/tests/utils"
	"github.com/saichler/l8srlz/go/serialize/object"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
	"testing"
)

func TestPostgresPatch(t *testing.T) {
	db := openDBConection()
	defer cleanup(db)

	res, _ := CreateResources(25000, 1, ifs.Info_Level)

	original := utils.CreateTestModelInstance(1)
	originalString := original.MyString
	originalInt32 := original.MyInt32
	originalFloat32 := original.MyFloat32

	resp := convert.ConvertTo(ifs.POST, object.New(nil, original), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	relData := resp.Element().(*l8orms.L8OrmRData)
	p := postgres.NewPostgres(db, res)
	err := p.WriteRelational(ifs.POST, relData)
	if err != nil {
		Log.Fail(t, "Error writing initial record", err)
		return
	}

	patch := &testtypes.TestProto{
		MyString: originalString,
		MyInt64:  999999,
	}

	resp = convert.ConvertTo(ifs.PATCH, object.New(nil, patch), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	relData = resp.Element().(*l8orms.L8OrmRData)
	err = p.WriteRelational(ifs.PATCH, relData)
	if err != nil {
		Log.Fail(t, "Error patching record", err)
		return
	}

	qr, err := interpreter.NewQuery("select * from testproto where mystring="+originalString, res)
	if err != nil {
		Log.Fail(t, err)
		return
	}
	relData, _, err = p.ReadRelational(qr)
	if err != nil {
		Log.Fail(t, "Error reading record", err)
		return
	}

	elems := object.New(nil, relData)
	readObjects := convert.ConvertFrom(elems, nil, res)
	if readObjects != nil && readObjects.Error() != nil {
		Log.Fail(t, "Error converting from relational", readObjects.Error())
		return
	}
	if len(readObjects.Elements()) != 1 {
		Log.Fail(t, "Expected 1 element, got:", len(readObjects.Elements()))
		return
	}

	result := readObjects.Elements()[0].(*testtypes.TestProto)

	if result.MyString != originalString {
		Log.Fail(t, "MyString should be", originalString, "but got", result.MyString)
		return
	}

	if result.MyInt64 != 999999 {
		Log.Fail(t, "MyInt64 should be 999999 (patched) but got", result.MyInt64)
		return
	}

	if result.MyInt32 != originalInt32 {
		Log.Fail(t, "MyInt32 should be", originalInt32, "(unchanged) but got", result.MyInt32)
		return
	}

	if result.MyFloat32 != originalFloat32 {
		Log.Fail(t, "MyFloat32 should be", originalFloat32, "(unchanged) but got", result.MyFloat32)
		return
	}

	Log.Info("PATCH test passed: MyInt64 updated to 999999, other fields unchanged")
}
