package convert

import (
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/types/go/common"
)

const (
	ServiceName      = "Convert"
	ServicePointType = "ConvertServicePoint"
)

type ConvertServicePoint struct {
}

func (this *ConvertServicePoint) Activate(serviceName string, serviceArea uint16,
	r common.IResources, l common.IServicePointCacheListener, args ...interface{}) error {
	r.Registry().Register(&types.RelationalData{})
	return nil
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
func (this *ConvertServicePoint) Failed(pb common.IElements, resourcs common.IResources, msg common.IMessage) common.IElements {
	return nil
}

func (this *ConvertServicePoint) Transactional() bool   { return false }
func (this *ConvertServicePoint) ReplicationCount() int { return 0 }
func (this *ConvertServicePoint) ReplicationScore() int { return 0 }
