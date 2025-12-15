package common

import (
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8types/go/ifs"
)

type IORM interface {
	Read(ifs.IQuery, ifs.IResources) ifs.IElements
	Write(ifs.Action, ifs.IElements, ifs.IResources) error
	Delete(ifs.IQuery, ifs.IResources) error
	Close() error
}

type IORMRelational interface {
	IORM
	ReadRelational(ifs.IQuery) (*l8orms.L8OrmRData, error)
	WriteRelational(ifs.Action, *l8orms.L8OrmRData) error
}
