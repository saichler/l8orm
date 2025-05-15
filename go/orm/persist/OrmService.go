package persist

import (
	common2 "github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/l8types/go/types"
	"github.com/saichler/l8utils/go/utils/web"
	"github.com/saichler/reflect/go/reflect/introspecting"
	"google.golang.org/protobuf/proto"
	"reflect"
)

const (
	ServiceType = "OrmService"
)

type OrmService struct {
	orm         common2.IORM
	serviceName string
	serviceArea uint16
	elem        proto.Message
}

func (this *OrmService) Activate(serviceName string, serviceArea uint16,
	r ifs.IResources, l ifs.IServiceCacheListener, args ...interface{}) error {
	r.Logger().Info("ORM Activated for ", serviceName, " area ", serviceArea)
	this.orm = args[0].(common2.IORM)
	this.elem = args[1].(proto.Message)
	r.Registry().Register(&types.RelationalData{})
	r.Registry().Register(&types2.Query{})
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
	query, err := pb.Query(vnic.Resources())
	if err != nil {
		return object.NewError(err.Error())
	}
	return this.orm.ReadObjects(query, vnic.Resources())
}
func (this *OrmService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *OrmService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg ifs.IMessage) ifs.IElements {
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
	return web.New(this.serviceName, this.serviceArea,
		nil, nil,
		nil, nil,
		nil, nil,
		nil, nil,
		&types2.Query{}, this.elem)
}
