package persist

import (
	common2 "github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/serializer/go/serialize/object"
	"github.com/saichler/types/go/common"
	types2 "github.com/saichler/types/go/types"
)

const (
	ServicePointType = "OrmServicePoint"
)

type OrmServicePoint struct {
	orm         common2.IORM
	serviceName string
}

func (this *OrmServicePoint) Activate(serviceName string, serviceArea uint16,
	r common.IResources, l common.IServicePointCacheListener, args ...interface{}) error {
	this.orm = args[0].(common2.IORM)
	r.Registry().Register(&types.RelationalData{})
	r.Registry().Register(&types2.Query{})
	return nil
}

func (this *OrmServicePoint) DeActivate() error {
	this.orm.Close()
	this.orm = nil
	return nil
}

func (this *OrmServicePoint) Post(pb common.IElements, resourcs common.IResources) common.IElements {
	resourcs.Logger().Info("Received Element to persist")
	relData, ok := pb.Element().(*types.RelationalData)
	var err error
	if ok {
		err = this.orm.Write(relData)
	} else {
		err = this.orm.WriteObjects(pb, resourcs)
	}
	if err != nil {
		return object.NewError(err.Error())
	}
	return object.New(nil, nil)
}

func (this *OrmServicePoint) Put(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *OrmServicePoint) Patch(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *OrmServicePoint) Delete(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *OrmServicePoint) Get(pb common.IElements, resourcs common.IResources) common.IElements {
	query, err := pb.Query(resourcs)
	if err != nil {
		return object.NewError(err.Error())
	}
	return this.orm.ReadObjects(query, resourcs)
}
func (this *OrmServicePoint) GetCopy(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *OrmServicePoint) Failed(pb common.IElements, resourcs common.IResources, msg common.IMessage) common.IElements {
	return nil
}

func (this *OrmServicePoint) Transactional() bool   { return true }
func (this *OrmServicePoint) ReplicationCount() int { return 0 }
func (this *OrmServicePoint) ReplicationScore() int { return 0 }
