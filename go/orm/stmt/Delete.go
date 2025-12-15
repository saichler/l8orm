package stmt

import (
	"database/sql"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
)

func (this *Statement) DeleteStatement(tx *sql.Tx, parentKeyPattern string) (*sql.Stmt, error) {
	del := strings.New("DELETE FROM ")
	del.Add(this.node.TypeName)

	// If parentKeyPattern is provided, delete by ParentKey pattern (for child tables)
	if parentKeyPattern != "" {
		del.Add(" WHERE ParentKey LIKE '")
		del.Add(parentKeyPattern)
		del.Add("%'")
	} else if this.query != nil && this.query.Criteria() != nil {
		// For root table, use the query criteria
		ok, whereClause := expression(this.query.Criteria(), this.query.RootType().TypeName)
		if ok {
			del.Add(" WHERE ")
			del.Add(whereClause)
		}
	}

	del.Add(";")
	return tx.Prepare(del.String())
}

func (this *Statement) DeleteByKeysStatement(tx *sql.Tx, keys []string) (*sql.Stmt, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	del := strings.New("DELETE FROM ")
	del.Add(this.node.TypeName)
	del.Add(" WHERE ")

	first := true
	for _, key := range keys {
		if !first {
			del.Add(" OR ")
		}
		first = false
		del.Add("ParentKey LIKE '")
		del.Add(key)
		del.Add("%'")
	}

	del.Add(";")
	return tx.Prepare(del.String())
}

func (this *Statement) Query2DeleteSql(query ifs.IQuery, typeName string) (string, bool) {
	del := strings.New("DELETE FROM ")
	del.Add(typeName)

	if query.Criteria() == nil {
		return del.String(), true
	}

	if typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			del.Add(" WHERE ")
			del.Add(str)
		}
	}
	return del.String(), true
}
