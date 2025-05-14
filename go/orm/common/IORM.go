package common

import (
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8types/go/ifs"
)

type IORM interface {
	Read(ifs.IQuery) (*types.RelationalData, error)
	Write(*types.RelationalData) error
	ReadObjects(ifs.IQuery, ifs.IResources) ifs.IElements
	WriteObjects(ifs.IElements, ifs.IResources) error
	Close() error
}
