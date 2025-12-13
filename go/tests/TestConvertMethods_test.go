package tests

import (
	"github.com/saichler/l8orm/go/types/l8orms"
	"testing"

	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8reflect/go/tests/utils"
	"github.com/saichler/l8srlz/go/serialize/object"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/testtypes"
)

func TestConvert(t *testing.T) {
	before := utils.CreateTestModelInstance(1)
	res, _ := CreateResources(25000, 1, ifs.Info_Level)
	resp := convert.ConvertTo(object.New(nil, before), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	r := resp.Element().(*l8orms.L8OrmRData)

	if len(r.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	resp = convert.ConvertFrom(resp, res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	if resp.Element() == nil {
		Log.Fail(t, "Nil Response")
		return
	}

	after := resp.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(res, false, false)
	err := upd.Update(before, after)
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if len(upd.Changes()) != 0 {
		Log.Fail(t, upd.Changes())
	}
}

func TestConvertMultiValue(t *testing.T) {
	before1 := utils.CreateTestModelInstance(1)
	before2 := utils.CreateTestModelInstance(2)
	res, _ := CreateResources(25000, 1, ifs.Info_Level)

	resp := convert.ConvertTo(object.New(nil, []*testtypes.TestProto{before1, before2}), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	r := resp.Element().(*l8orms.L8OrmRData)

	if len(r.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	resp = convert.ConvertFrom(resp, res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	if len(resp.Elements()) != 2 {
		Log.Fail(t, "Expected 2 elements")
		return
	}

	if resp.Element() == nil {
		Log.Fail(t, "Nil Response")
		return
	}

	after := resp.Element().(*testtypes.TestProto)

	upd := updating.NewUpdater(res, false, false)
	err := upd.Update(before1, after)
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if len(upd.Changes()) != 0 {
		Log.Fail(t, upd.Changes())
	}
}

func TestConvertMultiValueNoKey(t *testing.T) {
	before1 := utils.CreateTestModelInstance(1)
	before2 := utils.CreateTestModelInstance(2)
	res, _ := CreateResources(25000, 1, ifs.Info_Level)

	resp := convert.ConvertTo(object.New(nil, []*testtypes.TestProto{before1, before2}), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	r := resp.Element().(*l8orms.L8OrmRData)

	if len(r.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	resp = convert.ConvertFrom(resp, res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	if len(resp.Elements()) != 2 {
		Log.Fail(t, "Expected 2 elements")
		return
	}

	if resp.Element() == nil {
		Log.Fail(t, "Nil Response")
		return
	}

	var after *testtypes.TestProto
	for _, elem := range resp.Elements() {
		after = elem.(*testtypes.TestProto)
		if after.MyString == before1.MyString {
			break
		}
	}

	upd := updating.NewUpdater(res, false, false)
	err := upd.Update(before1, after)
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if len(upd.Changes()) != 0 {
		Log.Fail(t, upd.Changes())
	}
}
