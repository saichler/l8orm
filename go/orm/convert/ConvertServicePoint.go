package convert

import (
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/serializer/go/serialize/object"
	"github.com/saichler/types/go/common"
	types2 "github.com/saichler/types/go/types"
)

const (
	ServiceName = "Convert"
	ENDPOINT    = "convert"
)

type ConvertServicePoint struct {
}

func RegisterConvertCenter(serviceArea int32, resources common.IResources) {
	this := &ConvertServicePoint{}
	err := resources.ServicePoints().RegisterServicePoint(this, serviceArea)
	if err != nil {
		panic(err)
	}
}

func (this *ConvertServicePoint) Post(pb common.IElements, resourcs common.IResources) common.IElements {
	return ConvertTo(pb, resourcs)
}
func (this *ConvertServicePoint) Put(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *ConvertServicePoint) Patch(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *ConvertServicePoint) Delete(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *ConvertServicePoint) Get(pb common.IElements, resourcs common.IResources) common.IElements {
	return ConvertFrom(pb, resourcs)
}
func (this *ConvertServicePoint) GetCopy(pb common.IElements, resourcs common.IResources) common.IElements {
	return nil
}
func (this *ConvertServicePoint) Failed(pb common.IElements, resourcs common.IResources, msg *types2.Message) common.IElements {
	return nil
}
func (this *ConvertServicePoint) EndPoint() string {
	return ENDPOINT
}
func (this *ConvertServicePoint) ServiceName() string {
	return ServiceName
}
func (this *ConvertServicePoint) Transactional() bool   { return false }
func (this *ConvertServicePoint) ReplicationCount() int { return 0 }
func (this *ConvertServicePoint) ReplicationScore() int { return 0 }
func (this *ConvertServicePoint) ServiceModel() common.IElements {
	return object.New(nil, &types.RelationalData{})
}
