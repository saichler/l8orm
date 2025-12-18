package persist

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
)

func (this *OrmService) do(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	pbBefore, cont := this.Before(action, pb, vnic)
	if !cont {
		return object.New(nil, &l8web.L8Empty{})
	}
	if pbBefore != nil {
		if pbBefore.Error() != nil {
			return pbBefore
		}
		pb = pbBefore
	}

	err := this.orm.Write(action, pb, vnic.Resources())

	if err != nil {
		return object.NewError(err.Error())
	}
	pbAfter, cont := this.After(action, pb, vnic)
	if !cont {

	}
	if pbAfter != nil {
		return object.New(nil, &l8web.L8Empty{})
	}

	return object.New(nil, &l8web.L8Empty{})
}
