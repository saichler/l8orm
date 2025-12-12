package persist

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

func (this *OrmService) Before(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	if this.sla.Callback() != nil {
		elems := make([]interface{}, len(pb.Elements()))
		for i, elem := range pb.Elements() {
			before, err := this.sla.Callback().Before(elem, action, pb.Notification(), vnic)
			if err != nil {
				return object.NewError(err.Error())
			}
			if before != nil {
				elems[i] = before
			} else {
				elems[i] = elem
			}
		}
		if pb.Notification() {
			return object.NewNotify(elems)
		}
		return object.New(nil, elems)
	}
	return pb
}

func (this *OrmService) After(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	if this.sla.Callback() != nil {
		elems := make([]interface{}, len(pb.Elements()))
		for i, elem := range pb.Elements() {
			after, err := this.sla.Callback().After(elem, action, pb.Notification(), vnic)
			if err != nil {
				return object.NewError(err.Error())
			}
			if after != nil {
				elems[i] = after
			} else {
				elems[i] = elem
			}
		}
		if pb.Notification() {
			return object.NewNotify(elems)
		}
		return object.New(nil, elems)
	}
	return pb
}
