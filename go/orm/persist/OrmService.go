/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package persist provides a Layer 8 service wrapper for ORM operations.
// It implements IServicePointHandler to expose database CRUD operations as a distributed
// service, enabling remote clients to persist and query data through the service mesh.
package persist

import (
	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// OrmService wraps an IORM implementation as a Layer 8 service.
// It handles service lifecycle, request routing, and transaction management
// for database operations exposed through the service mesh.
type OrmService struct {
	orm common.IORM                   // The underlying ORM implementation
	sla *ifs.ServiceLevelAgreement    // Service configuration and metadata
}

// Activate is a convenience function to register an OrmService with the service mesh.
// It creates the service level agreement with the specified configuration and activates
// the service on the given virtual NIC. The keys parameter specifies primary key fields.
func Activate(serviceName string, serviceArea byte, item, itemList interface{},
	vnic ifs.IVNic, orm common.IORM, callback ifs.IServiceCallback, keys ...string) {
	sla := ifs.NewServiceLevelAgreement(&OrmService{}, serviceName, serviceArea, false, callback)
	sla.SetServiceItem(item)
	sla.SetServiceItemList(itemList)
	sla.SetPrimaryKeys(keys...)
	sla.SetArgs(orm)
	vnic.Resources().Services().Activate(sla, vnic)
}

// Activate initializes the OrmService when registered with the service mesh.
// It configures primary key and unique key decorators, and registers necessary types.
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

// DeActivate cleans up the OrmService, closing the underlying database connection.
func (this *OrmService) DeActivate() error {
	err := this.orm.Close()
	this.orm = nil
	return err
}

// Post handles POST requests to create new records in the database.
func (this *OrmService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.POST, pb, vnic)
}

// Put handles PUT requests to replace existing records in the database.
func (this *OrmService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.PUT, pb, vnic)
}
// Patch handles PATCH requests for partial updates to existing records.
func (this *OrmService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return this.do(ifs.PATCH, pb, vnic)
}
// Delete handles DELETE requests to remove records matching a query or filter.
// Supports both query-based deletion and filter mode using an example object.
func (this *OrmService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	if pb.IsFilterMode() {
		q, e := ElementToQuery(pb, this.sla.ServiceItem(), vnic)
		if e != nil {
			return object.NewError(e.Error())
		}
		err := this.orm.Delete(q, vnic.Resources())
		return object.New(err, nil)
	}

	// This is a query
	query, err := pb.Query(vnic.Resources())
	if err != nil {
		return object.NewError(err.Error())
	}
	err = this.orm.Delete(query, vnic.Resources())
	return object.New(err, nil)
}
// Get handles GET requests to retrieve records from the database.
// Supports both query-based retrieval and filter mode using an example object.
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
// GetCopy handles copy requests. Currently not implemented.
func (this *OrmService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
// Failed handles failure notifications from the service mesh. Currently not implemented.
func (this *OrmService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns this service as the transaction configuration.
// The OrmService itself implements ITransactionConfig for transaction management.
func (this *OrmService) TransactionConfig() ifs.ITransactionConfig {
	return this
}

// Replication returns whether this service supports data replication.
// Returns false as ORM operations are not replicated by default.
func (this *OrmService) Replication() bool {
	return false
}
// ReplicationCount returns the number of replicas for this service.
// Returns 0 as replication is not enabled by default.
func (this *OrmService) ReplicationCount() int {
	return 0
}
// Voter returns whether this service participates in consensus voting.
// Returns true to enable distributed consensus for write operations.
func (this *OrmService) Voter() bool {
	return true
}

// KeyOf extracts the primary key value from the given elements.
// Used for transaction coordination and cache key generation.
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

// WebService returns the web service configuration for REST API exposure.
// Delegates to the service level agreement's web service configuration.
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
