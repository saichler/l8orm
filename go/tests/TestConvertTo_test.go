package tests

import (
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/reflect/go/reflect/updating"
	"github.com/saichler/reflect/go/tests/utils"
	"github.com/saichler/shared/go/tests/infra"
	"github.com/saichler/types/go/testtypes"
	"github.com/saichler/types/go/types"
	"testing"
)

func TestConvert(t *testing.T) {
	before := utils.CreateTestModelInstance(1)
	res := createResources()
	node, _ := res.Introspector().Inspect(before)
	res.Introspector().AddDecorator(types.DecoratorType_Primary, []string{"MyString"}, node)
	r, e := convert.ConvertTo(before, res)
	if e != nil {
		infra.Log.Fail(t, e)
		return
	}

	if len(r.Tables) != 3 {
		infra.Log.Fail(t, "Expected 3 tables")
		return
	}

	i, e := convert.ConvertFrom(r, res)
	if e != nil {
		infra.Log.Fail(t, e)
		return
	}
	if i == nil {
		infra.Log.Fail(t, i)
		return
	}

	after := i.(*testtypes.TestProto)

	upd := updating.NewUpdater(res.Introspector(), false)
	err := upd.Update(before, after)
	if err != nil {
		infra.Log.Fail(t, err)
		return
	}
	if len(upd.Changes()) != 0 {
		infra.Log.Fail(t, upd.Changes())
	}
}
