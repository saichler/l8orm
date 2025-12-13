package persist

import (
	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

type OrmService struct {
	orm common.IORM
	sla *ifs.ServiceLevelAgreement
}

func Activate(serviceName string, serviceArea byte, item, itemList interface{},
	vnic ifs.IVNic, orm common.IORM, callback ifs.IServiceCallback, keys ...string) {
	sla := ifs.NewServiceLevelAgreement(&OrmService{}, serviceName, serviceArea, false, callback)
	sla.SetServiceItem(item)
	sla.SetServiceItemList(itemList)
	sla.SetPrimaryKeys(keys...)
	sla.SetArgs(orm)
	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *OrmService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Logger().Info("ORM Activated for ", sla.ServiceName(), " area ", sla.ServiceArea())
	this.sla = sla
	this.orm = this.sla.Args()[0].(common.IORM)
	_, err := vnic.Resources().Registry().Register(&l8orms.L8OrmRData{})
	if err != nil {
		return err
	}
	_, err = vnic.Resources().Registry().Register(sla.ServiceItemList())
	if err != nil {
		return err
	}
	err = vnic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(sla.ServiceItem(), sla.PrimaryKeys()...)
	if err != nil {
		return err
	}
	if sla.UniqueKeys() != nil {
		err = vnic.Resources().Introspector().Decorators().AddUniqueKeyDecorator(sla.ServiceItem(), sla.UniqueKeys()...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *OrmService) DeActivate() error {
	err := this.orm.Close()
	this.orm = nil
	return err
}

func (this *OrmService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.POST, pb, vnic)
}

func (this *OrmService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.PUT, pb, vnic)
}
func (this *OrmService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.PATCH, pb, vnic)
}
func (this *OrmService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *OrmService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {

	if pb.IsFilterMode() {
		q, e := ElementToQuery(pb, this.sla.ServiceItem(), vnic)
		if e != nil {
			return object.NewError(e.Error())
		}
		result := this.orm.Read(q, vnic.Resources())
		if result.Error() == nil {
			return result
		}
		return pb
	}

	// This is a query
	query, err := pb.Query(vnic.Resources())
	if err != nil {
		return object.NewError(err.Error())
	}
	return this.orm.Read(query, vnic.Resources())
}
func (this *OrmService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *OrmService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

func (this *OrmService) TransactionConfig() ifs.ITransactionConfig {
	return this
}

func (this *OrmService) Replication() bool {
	return true
}
func (this *OrmService) ReplicationCount() int {
	return 2
}
func (this *OrmService) Voter() bool {
	return true
}

func (this *OrmService) KeyOf(elements ifs.IElements, resources ifs.IResources) string {
	return KeyOf(elements, resources)
	/*
		query, err := elements.Query(resources)
		if err != nil {
			resources.Logger().Error(err)
			return ""
		}
		if query != nil {
			resources.Logger().Debug("KeyOf query ")
			return query.KeyOf()
		}

		elem := reflect.ValueOf(elements.Element()).Elem()
		elemTypeName := elem.Type().Name()
		resources.Logger().Debug("Key Of element of type ", elemTypeName)
		node, _ := resources.Introspector().NodeByTypeName(elemTypeName)
		keyDecorator, _ := helping.PrimaryKeyDecorator(node).([]string)
		field := elem.FieldByName(keyDecorator[0])
		str := field.String()
		if str == "" {
			panic("Empty key for type " + elemTypeName)
		}
		return str
	*/
}

func (this *OrmService) WebService() ifs.IWebService {
	return this.sla.WebService()
	/*
		fmt.Println("Web Service ", this.serviceName, " ", this.serviceArea)
		return web.New(this.serviceName, this.serviceArea,
			nil, nil,
			nil, nil,
			nil, nil,
			nil, nil,
			&l8api.L8Query{}, this.elem)
	*/
	return nil
}
