package convert

import (
	"bytes"
	"errors"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/reflect/go/reflect/helping"
	"github.com/saichler/serializer/go/serialize/object"
	"github.com/saichler/shared/go/share/strings"
	"github.com/saichler/types/go/common"
	types2 "github.com/saichler/types/go/types"
	"reflect"
	"strconv"
)

func ConvertTo(any interface{}, res common.IResources) (*types.RelationalData, error) {
	if any == nil {
		return nil, nil
	}

	v := reflect.ValueOf(any)
	data := &types.RelationalData{}
	data.Tables = make(map[string]*types.Table)
	data.RootTypeName = TypeOf(v)

	node, ok := res.Introspector().Node(data.RootTypeName)
	if !ok {
		return nil, errors.New("No node for type " + v.Type().Name())
	}

	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			err := convertTo(v, "", strconv.Itoa(i), node, data, res)
			if err != nil {
				return nil, err
			}
		}
	} else if v.Kind() == reflect.Map {
		mapKeys := v.MapKeys()
		for _, mapKey := range mapKeys {
			mapKeyStr := strings.New()
			mapKeyStr.TypesPrefix = true
			err := convertTo(v, "", mapKeyStr.ToString(mapKey), node, data, res)
			if err != nil {
				return nil, err
			}
		}
	} else {
		err := convertTo(v, "", "", node, data, res)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func TypeOf(v reflect.Value) string {
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
		return v.Type().Elem().Elem().Name()
	} else if v.Kind() == reflect.Ptr {
		return v.Elem().Type().Name()
	} else if v.Kind() == reflect.Struct {
		return v.Type().Name()
	}
	panic("Unknown type: " + v.Type().Name())
}

func convertTo(value reflect.Value, parentKey, myKey string, node *types2.RNode, data *types.RelationalData, res common.IResources) error {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	table, attributeRows := TableAndRowsCreate(node, data, parentKey)
	SetColumns(table, node)

	row := &types.Row{}
	row.ParentKey = parentKey
	row.RecKey = RecKey(node, value, myKey, res.Registry())
	row.ColumnValues = make(map[int32][]byte)

	subTableAttributes := make(map[string]*types2.RNode)
	for attrName, attrNode := range node.Attributes {
		if attrNode.IsStruct {
			subTableAttributes[attrName] = attrNode
			continue
		}
		fieldValue := value.FieldByName(attrName)
		if fieldValue.IsValid() {
			col := table.Columns[attrName]
			err := SetValueToRow(row, col, fieldValue)
			if err != nil {
				return err
			}
		}
	}

	for attrName, attrNode := range subTableAttributes {
		fieldValue := value.FieldByName(attrName)
		if fieldValue.IsValid() {
			if attrNode.IsMap {
				mapKeys := fieldValue.MapKeys()
				for _, mapKey := range mapKeys {
					mapValue := fieldValue.MapIndex(mapKey)
					mapValueStr := strings.New()
					mapValueStr.TypesPrefix = true
					err := convertTo(mapValue, KeyForRow(row), mapValueStr.ToString(mapKey), attrNode, data, res)
					if err != nil {
						return err
					}
				}
			} else if attrNode.IsSlice {
				for i := 0; i < fieldValue.Len(); i++ {
					sliceValue := fieldValue.Index(i)
					err := convertTo(sliceValue, KeyForRow(row), strconv.Itoa(i), attrNode, data, res)
					if err != nil {
						return err
					}
				}
			} else {
				err := convertTo(fieldValue, KeyForRow(row), "", attrNode, data, res)
				if err != nil {
					return err
				}
			}
		}
	}

	attributeRows.Rows = append(attributeRows.Rows, row)

	return nil
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

func TableAndRowsCreate(node *types2.RNode, data *types.RelationalData, parentKey string) (*types.Table, *types.AttributeRows) {
	table := data.Tables[node.TypeName]
	if table == nil {
		table = &types.Table{}
		table.Name = node.TypeName
		data.Tables[node.TypeName] = table
	}
	if table.InstanceRows == nil {
		table.InstanceRows = make(map[string]*types.InstanceRows)
	}
	instanceRows, ok := table.InstanceRows[parentKey]
	if !ok {
		instanceRows = &types.InstanceRows{}
		table.InstanceRows[parentKey] = instanceRows
	}
	if instanceRows.AttributeRows == nil {
		instanceRows.AttributeRows = make(map[string]*types.AttributeRows)
	}
	attributeRows, ok := instanceRows.AttributeRows[node.FieldName]
	if !ok {
		attributeRows = &types.AttributeRows{}
		instanceRows.AttributeRows[node.FieldName] = attributeRows
	}
	if attributeRows.Rows == nil {
		attributeRows.Rows = make([]*types.Row, 0)
	}
	return table, attributeRows
}

func SetValueToRow(row *types.Row, col int32, val reflect.Value) error {
	obj := object.NewEncode([]byte{}, 0)
	err := obj.Add(val.Interface())
	if err != nil {
		return err
	}
	row.ColumnValues[col] = obj.Data()
	return nil
}

func RecKey(node *types2.RNode, value reflect.Value, myKey string, reg common.IRegistry) string {
	key := helping.PrimaryDecorator(node, value, reg)
	if key == nil {
		str := strings.New(node.FieldName)
		str.Add("[")
		str.Add(myKey)
		str.Add("]")
		return str.String()
	} else {
		str := strings.New(node.FieldName)
		str.Add("[")
		str.Add(str.ToString(reflect.ValueOf(key)))
		str.Add("]")
		return str.String()
	}
}

func KeyForRow(row *types.Row) string {
	buff := bytes.Buffer{}
	buff.WriteString(row.ParentKey)
	buff.WriteString(row.RecKey)
	return buff.String()
}
