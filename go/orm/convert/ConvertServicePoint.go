package convert

import (
	"errors"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/types/go/common"
	types2 "github.com/saichler/types/go/types"
	"google.golang.org/protobuf/proto"
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

func (this *ConvertServicePoint) Post(pb proto.Message, resourcs common.IResources) (proto.Message, error) {
	resp, err := ConvertTo(pb, resourcs)
	return resp, err
}
func (this *ConvertServicePoint) Put(pb proto.Message, resourcs common.IResources) (proto.Message, error) {
	return nil, nil
}
func (this *ConvertServicePoint) Patch(pb proto.Message, resourcs common.IResources) (proto.Message, error) {
	return nil, nil
}
func (this *ConvertServicePoint) Delete(pb proto.Message, resourcs common.IResources) (proto.Message, error) {
	return nil, nil
}
func (this *ConvertServicePoint) Get(pb proto.Message, resourcs common.IResources) (proto.Message, error) {
	data, ok := pb.(*types.RelationalData)
	if !ok {
		return nil, errors.New("Data is not of type *types.RelationalData")
	}
	pbs, err := ConvertFrom(data, resourcs)
	if err != nil {
		return nil, err
	}
	return pbs.(proto.Message), nil
}
func (this *ConvertServicePoint) GetCopy(pb proto.Message, resourcs common.IResources) (proto.Message, error) {
	return nil, nil
}
func (this *ConvertServicePoint) Failed(pb proto.Message, resourcs common.IResources, msg *types2.Message) (proto.Message, error) {
	return nil, nil
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
func (this *ConvertServicePoint) ServiceModel() proto.Message {
	return &types.RelationalData{}
}
