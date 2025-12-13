package persist

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

func (this *OrmService) do(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	pbBefore := this.Before(action, pb, vnic)
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
	pbAfter := this.After(action, pb, vnic)
	if pbAfter != nil {
		return pbAfter
	}

	return object.New(nil, nil)
}
