package stmt

import (
	"database/sql"
	"github.com/saichler/l8types/go/types/l8api"
)

func (this *Statement) MetaData(tx *sql.Tx) *l8api.L8MetaData {
	stmt, err := this.metadataStatement(tx)
	if err != nil {
		return nil
	}
	metadata := &l8api.L8MetaData{}
	metadata.KeyCount = &l8api.L8Count{}
	metadata.KeyCount.Counts = make(map[string]int32)
	totalRecords := 0
	rows, err := stmt.Query()
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&totalRecords)
		if err != nil {
			return nil
		}
	}
	metadata.KeyCount.Counts["Total"] = int32(totalRecords)
	return metadata
}

func (this *Statement) metadataStatement(tx *sql.Tx) (*sql.Stmt, error) {
	if this.metaDataStmt == nil {
		err := this.createMetadataStatement(tx)
		if err != nil {
			return nil, err
		}
	}
	return this.metaDataStmt, nil
}

func (this *Statement) createMetadataStatement(tx *sql.Tx) error {
	sql := this.Query2CountSql(this.query, this.node.TypeName)
	st, err := tx.Prepare(sql)
	if err != nil {
		return err
	}
	this.metaDataStmt = st
	return nil
}
