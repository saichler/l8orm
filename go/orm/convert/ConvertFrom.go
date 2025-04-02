package convert

import (
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/serializer/go/serialize/object"
	strings2 "github.com/saichler/shared/go/share/strings"
	"github.com/saichler/types/go/common"
	types2 "github.com/saichler/types/go/types"
	"reflect"
	"strconv"
	"strings"
)

func ConvertFrom(o common.IElements, res common.IResources) common.IElements {
	data := o.Element().(*types.RelationalData)
	node, ok := res.Introspector().Node(data.RootTypeName)
	if !ok {
		return object.NewError("No node for type " + data.RootTypeName)
	}
	resp, e := convertFrom(node, "", data, res)
	if e != nil {
		return object.NewError(e.Error())
	}
	return object.New(nil, resp)
}

func convertFrom(node *types2.RNode, parentKey string, data *types.RelationalData, res common.IResources) (interface{}, error) {
	table, attributeRows := TableAndRowsGet(node, data, parentKey)

	//No data for this attribute
	if table == nil {
		return nil, nil
	}

	info, e := res.Registry().Info(node.TypeName)
	if e != nil {
		return nil, e
	}

	rows := make(map[string]interface{}, 0)
	subTableAttributes := make(map[string]*types2.RNode)
	subAttributesFull := false
	for _, row := range attributeRows.Rows {
		instance, err := info.NewInstance()
		if err != nil {
			return nil, err
		}
		value := reflect.ValueOf(instance).Elem()
		for attrName, attrNode := range node.Attributes {
			if attrNode.IsStruct {
				if !subAttributesFull {
					subTableAttributes[attrName] = attrNode
				}
				continue
			}
			colIndex, ok := table.Columns[attrName]
			if !ok {
				continue
			}
			e = SetValueFromRow(colIndex, attrName, value, row, res.Registry())
			if e != nil {
				return nil, e
			}
		}

		for attrName, attrNode := range subTableAttributes {
			v, e := convertFrom(attrNode, KeyForRow(row), data, res)
			if e != nil {
				return nil, e
			}
			if v == nil {
				continue
			}
			SetValueFromSubTable(attrName, value, v)
		}

		subAttributesFull = true
		rows[row.RecKey] = instance
	}

	if node.IsSlice {
		return convertRowsToSlice(rows, node, res.Registry())
	}
	if node.IsMap {
		v, e := convertRowsToMap(rows, node, res.Registry())
		if e != nil {
			panic(e)
		}
		if v == nil {
			panic(node.FieldName)
		}
		return v, e
	}
	return rows, nil
}

func convertRowsToSlice(rows map[string]interface{}, node *types2.RNode, r common.IRegistry) (interface{}, error) {
	info, e := r.Info(node.TypeName)
	if e != nil {
		return nil, e
	}

	slice := reflect.MakeSlice(reflect.SliceOf(reflect.New(info.Type()).Type()), len(rows), len(rows))
	for key, value := range rows {
		index, e := GetSliceIndex(key)
		if e != nil {
			return nil, e
		}
		slice.Index(index).Set(reflect.ValueOf(value))
	}
	return slice.Interface(), nil
}

func convertRowsToMap(rows map[string]interface{}, node *types2.RNode, r common.IRegistry) (interface{}, error) {
	valInfo, e := r.Info(node.TypeName)
	if e != nil {
		return nil, e
	}
	keyInfo, e := r.Info(node.KeyTypeName)
	if e != nil {
		return nil, e
	}

	newMap := reflect.MakeMap(reflect.MapOf(keyInfo.Type(), reflect.New(valInfo.Type()).Type()))

	for key, value := range rows {
		index, e := GetMapIndex(key, r)
		if e != nil {
			return nil, e
		}
		newMap.SetMapIndex(index, reflect.ValueOf(value))
	}
	return newMap.Interface(), nil
}

func GetMapIndex(mapKey string, r common.IRegistry) (reflect.Value, error) {
	index1 := strings.LastIndex(mapKey, "[")
	index2 := strings.LastIndex(mapKey, "]")
	return strings2.FromString(mapKey[index1+1:index2], r)
}

func GetSliceIndex(sliceKey string) (int, error) {
	index1 := strings.LastIndex(sliceKey, "[")
	index2 := strings.LastIndex(sliceKey, "]")
	index, e := strconv.Atoi(sliceKey[index1+1 : index2])
	return index, e
}

func SetValueFromRow(colIndex int32, attrName string, value reflect.Value, row *types.Row, r common.IRegistry) error {
	fieldValue := value.FieldByName(attrName)
	data := row.ColumnValues[colIndex]
	if data == nil || len(data) == 0 {
		return nil
	}
	obj := object.NewDecode(data, 0, r)
	v, e := obj.Get()
	if e != nil {
		return e
	}
	if fieldValue.Kind() == reflect.Int32 {
		fieldValue.SetInt(reflect.ValueOf(v).Int())
	} else {
		fieldValue.Set(reflect.ValueOf(v))
	}
	return nil
}

func SetValueFromSubTable(attrName string, value reflect.Value, i interface{}) {
	fieldValue := value.FieldByName(attrName)
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Map && fieldValue.Kind() != reflect.Map {
		mapKeys := v.MapKeys()
		fieldValue.Set(reflect.ValueOf(v.MapIndex(mapKeys[0]).Interface()))
		return
	}
	fieldValue.Set(reflect.ValueOf(i))
}

func TableAndRowsGet(node *types2.RNode, data *types.RelationalData, parentKey string) (*types.Table, *types.AttributeRows) {
	table := data.Tables[node.TypeName]
	if table == nil {
		return nil, nil
	}
	if table.InstanceRows == nil {
		return nil, nil
	}
	instanceRows, ok := table.InstanceRows[parentKey]
	if !ok {
		return nil, nil
	}
	if instanceRows.AttributeRows == nil {
		return nil, nil
	}
	attributeRows, ok := instanceRows.AttributeRows[node.FieldName]
	if !ok {
		return nil, nil
	}
	if attributeRows.Rows == nil {
		return nil, nil
	}
	return table, attributeRows
}
