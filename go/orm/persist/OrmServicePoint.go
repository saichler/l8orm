package persist

import (
	common2 "github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/serializer/go/serialize/object"
	"github.com/saichler/types/go/common"
	types2 "github.com/saichler/types/go/types"
	"reflect"
)

type OrmServicePoint struct {
	orm         common2.IORM
	serviceName string
}

func ActivateOrmService(orm common2.IORM, serviceArea uint16, resources common.IResources) error {
	this := &OrmServicePoint{}
	this.orm = orm
	this.serviceName = "Orm-" + reflect.ValueOf(orm).Elem().Type().Name()
	err := resources.ServicePoints().RegisterServicePoint(this)
	if err != nil {
		return err
	}
	err = resources.ServicePoints().Activate(this.serviceName, serviceArea, nil)
	resources.Registry().Register(&types.RelationalData{})
	resources.Registry().Register(&types2.Query{})
	return nil
}

func (this *OrmServicePoint) Post(pb common.IElements, resourcs common.IResources) common.IElements {
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
func (this *OrmServicePoint) EndPoint() string {
	return this.serviceName
}
func (this *OrmServicePoint) ServiceName() string {
	return this.serviceName
}
func (this *OrmServicePoint) Transactional() bool   { return true }
func (this *OrmServicePoint) ReplicationCount() int { return 0 }
func (this *OrmServicePoint) ReplicationScore() int { return 0 }
func (this *OrmServicePoint) ServiceModel() common.IElements {
	return object.New(nil, &types.RelationalData{})
}
