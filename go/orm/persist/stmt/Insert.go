package stmt

import (
	"database/sql"
	"github.com/saichler/l8utils/go/utils/strings"
	"strconv"
)

func (this *Statement) InsertStatement(tx *sql.Tx) (*sql.Stmt, error) {
	if this.insertStmt == nil {
		err := this.createInsertStatement(tx)
		if err != nil {
			return nil, err
		}
	}
	return this.insertStmt, nil
}

func (this *Statement) createInsertStatement(tx *sql.Tx) error {
	insertInto := strings.New("insert into ", this.node.TypeName)
	if this.fields == nil {
		this.fields, this.values = fieldsOf(this.node)
	}
	fields := strings.New(" (")
	values := strings.New(" values (")
	conflict := strings.New("ON CONFLICT (ParentKey,RecKey) DO UPDATE SET ")
	first := true
	firstConflict := true
	for _, field := range this.fields {
		if !first {
			fields.Add(",")
			values.Add(",")
		}
		first = false
		fields.Add(field)
		values.Add("$")
		values.Add(strconv.Itoa(this.values[field]))
		if field != "ParentKey" && field != "RecKey" {
			if !firstConflict {
				conflict.Add(",")
			}
			firstConflict = false
			conflict.Add(field)
			conflict.Add("=")
			conflict.Add("$")
			conflict.Add(strconv.Itoa(this.values[field]))
			conflict.Add(" ")
		}
	}
	fields.Add(") ")
	values.Add(") ")
	insertInto.Add(fields.String())
	insertInto.Add(values.String())
	insertInto.Add(conflict.String())
	insertInto.Add(";")

	st, err := tx.Prepare(insertInto.String())
	if err != nil {
		return err
	}
	this.insertStmt = st
	return nil
}
