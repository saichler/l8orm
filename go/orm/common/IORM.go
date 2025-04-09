package common

import (
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/types/go/common"
)

type IORM interface {
	Read(common.IQuery) (*types.RelationalData, error)
	Write(*types.RelationalData) error
	ReadObjects(common.IQuery, common.IResources) common.IElements
	WriteObjects(common.IElements, common.IResources) error
}
