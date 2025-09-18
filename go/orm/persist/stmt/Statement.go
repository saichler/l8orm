package stmt

import (
	"database/sql"
	"reflect"

	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/strings"
)

type Statement struct {
	fields  []string
	values  map[string]int
	columns map[string]int32
	registy ifs.IRegistry
	node    *l8reflect.L8Node
	query   ifs.IQuery

	insertStmt *sql.Stmt
	selectStmt *sql.Stmt
}

func NewStatement(node *l8reflect.L8Node, columns map[string]int32, query ifs.IQuery, registy ifs.IRegistry) *Statement {
	return &Statement{node: node, columns: columns, registy: registy, query: query}
}

func (this *Statement) RowValues(row *types.Row) ([]interface{}, error) {
	result := make([]interface{}, len(this.values))
	result[0] = row.ParentKey
	result[1] = row.RecKey
	for attrName, attr := range this.node.Attributes {
		if attr.IsStruct {
			continue
		}
		fieldPos := this.values[attrName]
		rowPos := this.columns[attrName]
		data := row.ColumnValues[rowPos]
		val, err := getValueForPostgres(data, this.registy)
		if err != nil {
			return nil, err
		}
		result[fieldPos-1] = val
	}
	return result, nil
}

func fieldsOf(node *l8reflect.L8Node) ([]string, map[string]int) {
	fields := []string{"ParentKey", "RecKey"}
	values := map[string]int{"ParentKey": 1, "RecKey": 2}
	index := 3
	for attrName, attr := range node.Attributes {
		if attr.IsStruct {
			continue
		}
		fields = append(fields, attrName)
		values[attrName] = index
		index++
	}
	return fields, values
}

func getValueForPostgres(data []byte, r ifs.IRegistry) (interface{}, error) {
	obj := object.NewDecode(data, 0, r)
	val, err := obj.Get()
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
		str := strings.New()
		str.TypesPrefix = true
		val = str.ToString(v)
	}
	return val, nil
}
