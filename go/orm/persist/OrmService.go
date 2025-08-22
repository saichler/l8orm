package persist

import (
	"fmt"
	"reflect"

	common2 "github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/l8types/go/types"
	"github.com/saichler/l8utils/go/utils/web"
	"github.com/saichler/reflect/go/reflect/introspecting"
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

func (this *OrmService) Activate(serviceName string, serviceArea byte,
	r ifs.IResources, l ifs.IServiceCacheListener, args ...interface{}) error {
	r.Logger().Info("ORM Activated for ", serviceName, " area ", serviceArea)
	this.orm = args[0].(common2.IORM)
	this.elem = args[1].(proto.Message)
	r.Registry().Register(&types.RelationalData{})
	r.Registry().Register(&types2.Query{})
	this.serviceName = serviceName
	this.serviceArea = serviceArea
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
	return nil
}
func (this *OrmService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *OrmService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	// in case the pb is an instance of the element and not a query.
	ins, ok := pb.Element().(proto.Message)
	if ok {
		aside := reflect.ValueOf(ins).Elem().Type().Name()
		bside := reflect.ValueOf(this.elem).Elem().Type().Name()
		if aside == bside {
			rnode, ok := vnic.Resources().Introspector().NodeByTypeName(bside)
			if ok {
				fields := introspecting.PrimaryKeyDecorator(rnode).([]string)
				v := reflect.ValueOf(ins).FieldByName(fields[0])
				gsql := "select * from " + bside + " where " + fields[0] + "=" + v.String()
				q1, err := object.NewQuery(gsql, vnic.Resources())
				if err == nil {
					panic(err)
				}
				q2, err := q1.Query(vnic.Resources())
				if err == nil {
					panic(err)
				}
				return this.orm.ReadObjects(q2, vnic.Resources())
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

func (this *OrmService) TransactionMethod() ifs.ITransactionMethod {
	return this
}

func (this *OrmService) Replication() bool {
	return true
}
func (this *OrmService) ReplicationCount() int {
	return 2
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
		&types2.Query{}, this.elem)
}
