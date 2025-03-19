package tests

import (
	"github.com/saichler/reflect/go/reflect/introspecting"
	"github.com/saichler/servicepoints/go/points/service_points"
	"github.com/saichler/shared/go/share/registry"
	"github.com/saichler/shared/go/share/resources"
	"github.com/saichler/shared/go/tests/infra"
	"github.com/saichler/types/go/common"
	"github.com/saichler/types/go/types"
)

func createResources() common.IResources {
	reg := registry.NewRegistry()
	secure, err := common.LoadSecurityProvider("security.so")
	if err != nil {
		panic("Failed to load security provider")
	}
	config := &types.VNicConfig{MaxDataSize: resources.DEFAULT_MAX_DATA_SIZE,
		RxQueueSize: resources.DEFAULT_QUEUE_SIZE,
		TxQueueSize: resources.DEFAULT_QUEUE_SIZE,
		LocalAlias:  "vnet"}
	ins := introspecting.NewIntrospect(reg)
	sps := service_points.NewServicePoints(ins, config)

	res := resources.NewResources(reg, secure, sps, infra.Log, nil, nil, config, ins)
	return res
}
