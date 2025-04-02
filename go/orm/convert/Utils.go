package convert

import (
	"errors"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/types/go/common"
	types2 "github.com/saichler/types/go/types"
	"strings"
	"sync"
)

func NewRelationsDataForQuery(query common.IQuery) (*types.RelationalData, error) {
	rlData := &types.RelationalData{}
	rlData.RootTypeName = query.RootType().TypeName
	rlData.Tables = make(map[string]*types.Table)
	properties := make([]string, len(query.Properties()))
	for i, p := range query.Properties() {
		id, _ := p.PropertyId()
		index := strings.Index(id, ".")
		properties[i] = id[index+1:]
	}
	err := addTable(query.RootType(), rlData, properties...)
	return rlData, err
}

func NewRelationalDataForType(typeName string, introspector common.IIntrospector) (*types.RelationalData, error) {
	node, ok := introspector.NodeByTypeName(typeName)
	if !ok {
		return nil, errors.New("Did not find any node for " + typeName)
	}
	rlData := &types.RelationalData{}
	rlData.RootTypeName = typeName
	rlData.Tables = make(map[string]*types.Table)
	err := addTable(node, rlData, typeName)
	return rlData, err
}

func addTable(node *types2.RNode, rlData *types.RelationalData, properties ...string) error {
	_, ok := rlData.Tables[node.TypeName]
	if ok && properties == nil {
		return nil
	}
	table := &types.Table{}
	table.Name = node.TypeName
	rlData.Tables[node.TypeName] = table

	if properties == nil || len(properties) == 0 {
		SetColumns(table, node)
		for _, attr := range node.Attributes {
			if attr.IsStruct {
				err := addTable(attr, rlData)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	subProperties := make([]string, 0)
	for _, property := range properties {
		attrName, subProperty := getAttrName(property, node)
		if subProperty != "" {
			subProperties = append(subProperties, property)
			continue
		}
		attr := node.Attributes[attrName]
		if attr.IsStruct {
			return errors.New("Trying to get attribute " + attr.FieldName + " from " +
				node.TypeName + ", which is a table")
		}
		addColumn(table, attr.FieldName)
	}
	for _, subProperty := range subProperties {
		attrName, property := getAttrName(subProperty, node)
		attr := node.Attributes[attrName]
		err := addTable(attr, rlData, property)
		if err != nil {
			return err
		}
	}
	return nil
}

var attributes = make(map[string]map[string]string)
var mtx = &sync.RWMutex{}

func getAttrName(property string, node *types2.RNode) (string, string) {
	var attrName string
	var attrProp string
	index := strings.Index(property, ".")
	if index != -1 {
		attrName = property[0:index]
		attrProp = property[index+1:]
	} else {
		attrName = property
	}

	if node.Attributes[attrName] != nil {
		return attrName, attrProp
	}

	attrName = strings.ToLower(attrName)
	mtx.RLock()
	defer mtx.RUnlock()
	_, ok := attributes[node.TypeName]
	if !ok {
		mtx.RUnlock()
		mtx.Lock()
		attributes[node.TypeName] = make(map[string]string)
		for name, _ := range node.Attributes {
			attributes[node.TypeName][strings.ToLower(name)] = name
		}
		mtx.Unlock()
		mtx.RLock()
	}
	attrName = attributes[node.TypeName][attrName]
	if attrName == "" {
		panic(property + " does not exist")
	}
	return attrName, attrProp
}

func SetColumns(table *types.Table, node *types2.RNode) {
	if table.Columns == nil {
		table.Columns = make(map[string]int32)
		for attrName, attrNode := range node.Attributes {
			if attrNode.IsStruct {
				continue
			}
			table.Columns[attrName] = int32(len(table.Columns) + 1)
		}
	}
}

func addColumn(table *types.Table, attrName string) {
	if table.Columns == nil {
		table.Columns = make(map[string]int32)
	}
	table.Columns[attrName] = int32(len(table.Columns) + 1)
}
