package stmt

import (
	"database/sql"
	"github.com/saichler/l8utils/go/utils/strings"
	"strconv"
)

func (this *Statement) UpdateStatement(tx *sql.Tx) (*sql.Stmt, error) {
	if this.updateStmt == nil {
		err := this.createUpdateStatement(tx)
		if err != nil {
			return nil, err
		}
	}
	return this.updateStmt, nil
}

func (this *Statement) createUpdateStatement(tx *sql.Tx) error {
	if this.fields == nil {
		this.fields, this.values = fieldsOf(this.node)
	}

	update := strings.New("UPDATE ", this.node.TypeName, " SET ")
	first := true

	for _, field := range this.fields {
		if field == "ParentKey" || field == "RecKey" {
			continue
		}
		if !first {
			update.Add(", ")
		}
		first = false
		update.Add(field, "=COALESCE($", strconv.Itoa(this.values[field]), ", ", field, ")")
	}

	update.Add(" WHERE ParentKey=$1 AND RecKey=$2;")

	st, err := tx.Prepare(update.String())
	if err != nil {
		return err
	}
	this.updateStmt = st
	return nil
}
