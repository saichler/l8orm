package persist

import (
	"fmt"
	"reflect"

	common2 "github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8reflect/go/reflect/introspecting"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/web"
	"google.golang.org/protobuf/proto"
)

const (
	ServiceType = "OrmService"
)

type OrmService struct {
	orm         common2.IORM
	serviceName string
	serviceArea byte
	elem        proto.Message
}

func (this *OrmService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Logger().Info("ORM Activated for ", sla.ServiceName(), " area ", sla.ServiceArea())
	this.orm = sla.Args()[0].(common2.IORM)
	this.elem = sla.ServiceItem().(proto.Message)
	vnic.Resources().Registry().Register(&types.RelationalData{})
	vnic.Resources().Registry().Register(&l8api.L8Query{})
	this.serviceName = sla.ServiceName()
	this.serviceArea = sla.ServiceArea()
	return nil
}

func (this *OrmService) DeActivate() error {
	this.orm.Close()
	this.orm = nil
	return nil
}

func (this *OrmService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	vnic.Resources().Logger().Info("Received Element to persist")
	relData, ok := pb.Element().(*types.RelationalData)
	var err error
	if ok {
		err = this.orm.Write(relData)
	} else {
		err = this.orm.WriteObjects(pb, vnic.Resources())
	}
	if err != nil {
		return object.NewError(err.Error())
	}
	return object.New(nil, nil)
}

func (this *OrmService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *OrmService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.Post(pb, vnic)
}
func (this *OrmService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *OrmService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {

	if pb.IsFilterMode() {
		asideValue := reflect.ValueOf(pb.Element())
		aside := asideValue.Elem().Type().Name()
		bside := reflect.ValueOf(this.elem).Elem().Type().Name()
		if aside == bside {
			rnode, ok := vnic.Resources().Introspector().NodeByTypeName(bside)
			if ok {
				fields := introspecting.PrimaryKeyDecorator(rnode).([]string)
				v := asideValue.Elem().FieldByName(fields[0])
				gsql := "select * from " + bside + " where " + fields[0] + "=" + v.String()
				vnic.Resources().Logger().Info("Constructed Query is: ", gsql)
				q1, err := object.NewQuery(gsql, vnic.Resources())
				if err != nil {
					panic(gsql + " " + err.Error())
				}
				q2, err := q1.Query(vnic.Resources())
				if err != nil {
					panic(gsql + " " + err.Error())
				}
				result := this.orm.ReadObjects(q2, vnic.Resources())
				if result.Error() == nil {
					return result
				}
				return pb
			}
		}
	}

	// This is a query
	query, err := pb.Query(vnic.Resources())
	if err != nil {
		return object.NewError(err.Error())
	}
	return this.orm.ReadObjects(query, vnic.Resources())
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
	keyDecorator, _ := introspecting.PrimaryKeyDecorator(node).([]string)
	field := elem.FieldByName(keyDecorator[0])
	str := field.String()
	if str == "" {
		panic("Empty key for type " + elemTypeName)
	}
	return str
}

func (this *OrmService) WebService() ifs.IWebService {
	fmt.Println("Web Service ", this.serviceName, " ", this.serviceArea)
	return web.New(this.serviceName, this.serviceArea,
		nil, nil,
		nil, nil,
		nil, nil,
		nil, nil,
		&l8api.L8Query{}, this.elem)
}
