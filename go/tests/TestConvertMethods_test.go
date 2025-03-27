package tests

import (
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/types"
	. "github.com/saichler/l8test/go/infra/t_resources"
	"github.com/saichler/reflect/go/reflect/introspecting"
	"github.com/saichler/reflect/go/reflect/updating"
	"github.com/saichler/reflect/go/tests/utils"
	"github.com/saichler/serializer/go/serialize/object"
	"github.com/saichler/types/go/testtypes"
	"testing"
)

func TestConvert(t *testing.T) {
	before := utils.CreateTestModelInstance(1)
	res, _ := CreateResources(25000, 1)
	node, _ := res.Introspector().Inspect(before)
	introspecting.AddPrimaryKeyDecorator(node, "MyString")
	resp := convert.ConvertTo(object.New(nil, before), res)
	if resp != nil && resp.Error() != nil {
		Log.Fail(t, resp.Error())
		return
	}

	r := resp.Element().(*types.RelationalData)

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
