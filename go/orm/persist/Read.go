package persist

import (
	"database/sql"
	"errors"
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/orm/persist/stmt"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/serializer/go/serialize/object"
	"github.com/saichler/types/go/common"
	"strings"
)

func (this *Postgres) Read(query common.IQuery) (*types.RelationalData, error) {
	data, err := convert.NewRelationsDataForQuery(query)
	if err != nil {
		return nil, err
	}

	this.mtx.Lock()
	defer this.mtx.Unlock()

	var tx *sql.Tx
	var er error

	tx, er = this.db.Begin()
	if er != nil {
		return nil, er
	}

	defer func() {
		if er != nil {
			er = tx.Rollback()
		} else {
			er = tx.Commit()
		}
	}()

	for tableName, table := range data.Tables {
		node, ok := this.res.Introspector().NodeByTypeName(tableName)
		if !ok {
			return nil, errors.New("table not found " + data.RootTypeName)
		}
		statement := stmt.NewStatement(node, table.Columns, query, this.res.Registry())
		st, err := statement.SelectStatement(tx)
		if err != nil {
			return nil, err
		}
		if st == nil {
			continue
		}
		rows, err := st.Query()
		if err != nil {
			return nil, err
		}
		dataRow, err := this.readRows(rows, statement)
		if err != nil {
			return nil, err
		}
		for _, row := range dataRow {
			fldName := nameOfField(row.RecKey)
			if table.InstanceRows == nil {
				table.InstanceRows = make(map[string]*types.InstanceRows)
			}
			if table.InstanceRows[row.ParentKey] == nil {
				table.InstanceRows[row.ParentKey] = &types.InstanceRows{}
			}
			if table.InstanceRows[row.ParentKey].AttributeRows == nil {
				table.InstanceRows[row.ParentKey].AttributeRows = make(map[string]*types.AttributeRows)
			}
			if table.InstanceRows[row.ParentKey].AttributeRows[fldName] == nil {
				table.InstanceRows[row.ParentKey].AttributeRows[fldName] = &types.AttributeRows{}
			}
			if table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows == nil {
				table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows = make([]*types.Row, 0)
			}
			attrRows := table.InstanceRows[row.ParentKey].AttributeRows[fldName]
			attrRows.Rows = append(attrRows.Rows, row)
		}
	}
	return data, nil
}

func nameOfField(recKey string) string {
	index := strings.Index(recKey, "[")
	if index == -1 {
		return recKey
	}
	return recKey[0:index]
}

func (this *Postgres) readRows(rows *sql.Rows, statement *stmt.Statement) ([]*types.Row, error) {
	result := make([]*types.Row, 0)
	for rows.Next() {
		row, err := statement.Row(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
}

func (this *Postgres) ReadObjects(q common.IQuery, resources common.IResources) common.IElements {
	relData, err := this.Read(q)
	if err != nil {
		return object.NewError(err.Error())
	}
	return convert.ConvertFrom(object.New(nil, relData), resources)
}
