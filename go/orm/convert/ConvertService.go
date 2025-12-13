package convert

import (
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8types/go/ifs"
)

const (
	ServiceName = "Convert"
)

type ConvertService struct {
}

func (this *ConvertService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8orms.L8OrmRData{})
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
func (this *ConvertService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

func (this *ConvertService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *ConvertService) WebService() ifs.IWebService {
	return nil
}
