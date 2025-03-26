package tests

import (
	"github.com/saichler/l8orm/go/orm/convert"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/reflect/go/reflect/introspecting"
	"github.com/saichler/reflect/go/reflect/updating"
	"github.com/saichler/reflect/go/tests/utils"
	"github.com/saichler/types/go/testtypes"
	"testing"
)

func TestConvert(t *testing.T) {
	before := utils.CreateTestModelInstance(1)
	res, _ := CreateResources(25000, 1)
	node, _ := res.Introspector().Inspect(before)
	introspecting.AddPrimaryKeyDecorator(node, "MyString")
	r, e := convert.ConvertTo(before, res)
	if e != nil {
		Log.Fail(t, e)
		return
	}

	if len(r.Tables) != 3 {
		Log.Fail(t, "Expected 3 tables")
		return
	}

	i, e := convert.ConvertFrom(r, res)
	if e != nil {
		Log.Fail(t, e)
		return
	}
	if i == nil {
		Log.Fail(t, i)
		return
	}

	after := i.(*testtypes.TestProto)

	upd := updating.NewUpdater(res.Introspector(), false)
	err := upd.Update(before, after)
	if err != nil {
		Log.Fail(t, err)
		return
	}
	if len(upd.Changes()) != 0 {
		Log.Fail(t, upd.Changes())
	}
}
