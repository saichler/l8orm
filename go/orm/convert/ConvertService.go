package convert

import (
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8types/go/ifs"
)

const (
	ServiceName = "Convert"
	ServiceType = "ConvertService"
)

type ConvertService struct {
}

func (this *ConvertService) Activate(serviceName string, serviceArea uint16,
	r ifs.IResources, l ifs.IServiceCacheListener, args ...interface{}) error {
	r.Registry().Register(&types.RelationalData{})
	return nil
}

func (this *ConvertService) DeActivate() error {
	return nil
}

func (this *ConvertService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertTo(pb, vnic.Resources())
}
func (this *ConvertService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ConvertService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ConvertService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ConvertService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertFrom(pb, vnic.Resources())
}
func (this *ConvertService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *ConvertService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg ifs.IMessage) ifs.IElements {
	return nil
}

func (this *ConvertService) TransactionMethod() ifs.ITransactionMethod {
	return nil
}

func (this *ConvertService) WebService() ifs.IWebService {
	return nil
}
