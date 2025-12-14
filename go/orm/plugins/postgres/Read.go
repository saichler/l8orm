package postgres

import (
	"database/sql"
	"errors"
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/orm/stmt"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"strings"
)

func (this *Postgres) ReadRelational(query ifs.IQuery) (*l8orms.L8OrmRData, *l8api.L8MetaData, error) {
	data, err := convert.NewRelationsDataForQuery(query)
	if err != nil {
		return nil, nil, err
	}

	this.mtx.Lock()
	defer this.mtx.Unlock()

	var tx *sql.Tx
	var er error

	tx, er = this.db.Begin()
	if er != nil {
		return nil, nil, er
	}

	defer func() {
		if er != nil {
			er = tx.Rollback()
		} else {
			er = tx.Commit()
		}
	}()

	var rootTableStatement *stmt.Statement

	for tableName, table := range data.Tables {
		node, ok := this.res.Introspector().NodeByTypeName(tableName)
		if !ok {
			return nil, nil, errors.New("table not found " + data.RootTypeName)
		}
		statement := stmt.NewStatement(node, table.Columns, query, this.res.Registry())
		st, err := statement.SelectStatement(tx)
		if err != nil {
			return nil, nil, err
		}
		if st == nil {
			continue
		}

		if strings.ToLower(tableName) == strings.ToLower(query.RootType().TypeName) {
			rootTableStatement = statement
		}

		rows, err := st.Query()
		if err != nil {
			return nil, nil, err
		}
		dataRow, err := this.readRows(rows, statement)
		if err != nil {
			return nil, nil, err
		}
		for _, row := range dataRow {
			fldName := nameOfField(row.RecKey)
			if table.InstanceRows == nil {
				table.InstanceRows = make(map[string]*l8orms.L8OrmInstanceRows)
			}
			if table.InstanceRows[row.ParentKey] == nil {
				table.InstanceRows[row.ParentKey] = &l8orms.L8OrmInstanceRows{}
			}
			if table.InstanceRows[row.ParentKey].AttributeRows == nil {
				table.InstanceRows[row.ParentKey].AttributeRows = make(map[string]*l8orms.L8OrmAttributeRows)
			}
			if table.InstanceRows[row.ParentKey].AttributeRows[fldName] == nil {
				table.InstanceRows[row.ParentKey].AttributeRows[fldName] = &l8orms.L8OrmAttributeRows{}
			}
			if table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows == nil {
				table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows = make([]*l8orms.L8OrmRow, 0)
			}
			attrRows := table.InstanceRows[row.ParentKey].AttributeRows[fldName]
			attrRows.Rows = append(attrRows.Rows, row)
		}
	}
	return data, rootTableStatement.MetaData(tx), nil
}

func nameOfField(recKey string) string {
	index := strings.Index(recKey, "[")
	if index == -1 {
		return recKey
	}
	return recKey[0:index]
}

func (this *Postgres) readRows(rows *sql.Rows, statement *stmt.Statement) ([]*l8orms.L8OrmRow, error) {
	result := make([]*l8orms.L8OrmRow, 0)
	for rows.Next() {
		row, err := statement.Row(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
}

func (this *Postgres) Read(q ifs.IQuery, resources ifs.IResources) ifs.IElements {
	relData, metadata, err := this.ReadRelational(q)
	if err != nil {
		return object.NewError(err.Error())
	}
	return convert.ConvertFrom(object.New(nil, relData), metadata, resources)
}
