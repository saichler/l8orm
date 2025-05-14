package persist

import (
	"database/sql"
	"errors"
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/orm/persist/stmt"
	"github.com/saichler/l8orm/go/types"
	"github.com/saichler/l8types/go/ifs"
)

func (this *Postgres) Write(data *types.RelationalData) error {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	rootNode, ok := this.res.Introspector().NodeByTypeName(data.RootTypeName)
	if !ok {
		return errors.New("Cannot find node for root type name " + data.RootTypeName)
	}
	err := this.verifyTables(rootNode)
	if err != nil {
		return err
	}
	err = this.writeData(data)
	if err != nil {
		return err
	}
	return nil
}

func (this *Postgres) writeData(data *types.RelationalData) error {
	var tx *sql.Tx
	var er error

	tx, er = this.db.Begin()
	if er != nil {
		return er
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
			return errors.New("No node was found for " + tableName)
		}
		statement := stmt.NewStatement(node, table.Columns, nil, this.res.Registry())
		insert, err := statement.InsertStatement(tx)
		if err != nil {
			return err
		}
		for _, instRows := range table.InstanceRows {
			for _, attrRows := range instRows.AttributeRows {
				for _, row := range attrRows.Rows {
					args, e := statement.RowValues(row)
					if e != nil {
						return err
					}
					_, e = insert.Exec(args...)
					if e != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (this *Postgres) WriteObjects(elems ifs.IElements, resources ifs.IResources) error {
	relData := convert.ConvertTo(elems, resources)
	if relData.Error() != nil {
		return relData.Error()
	}
	return this.Write(relData.Element().(*types.RelationalData))
}
