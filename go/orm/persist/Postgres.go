package persist

import (
	"database/sql"
	"errors"
	"github.com/saichler/l8utils/go/utils/strings"
	"github.com/saichler/l8types/go/ifs"
	types2 "github.com/saichler/l8types/go/types"
	strings2 "strings"
	"sync"
)

type Postgres struct {
	db       *sql.DB
	verifyed map[string]bool
	mtx      *sync.Mutex
	res      ifs.IResources
}

func NewPostgres(db *sql.DB, resourcs ifs.IResources) *Postgres {
	return &Postgres{db: db, verifyed: make(map[string]bool), mtx: &sync.Mutex{}, res: resourcs}
}

func collectTables(node *types2.RNode, tables map[string]bool) {
	tables[node.TypeName] = true
	if node.Attributes != nil {
		for _, attr := range node.Attributes {
			if attr.IsStruct {
				_, ok := tables[attr.TypeName]
				if !ok {
					collectTables(attr, tables)
				}
			}
		}
	}
}

func (this *Postgres) verifyTables(rootNode *types2.RNode) error {
	tables := make(map[string]bool)
	collectTables(rootNode, tables)
	for tableName, _ := range tables {
		_, ok := this.verifyed[tableName]
		if !ok {
			err := this.verifyTable(tableName)
			if err != nil {
				return err
			}
			this.verifyed[tableName] = true
		}
	}
	return nil
}

func (this *Postgres) verifyTable(tableName string) error {
	q := strings.New("select * from ", tableName, " where false;")
	_, err := this.db.Exec(q.String())
	if err != nil && strings2.Contains(err.Error(), "does not exist") {
		return this.createTable(tableName)
	}
	return err
}

func (this *Postgres) createTable(tableName string) error {
	q := strings.New("create table ", tableName, " (\n")
	q.Add("ParentKey text,\n")
	q.Add("RecKey text,\n")
	node, ok := this.res.Introspector().NodeByTypeName(tableName)
	if !ok {
		return errors.New("Cannot find node for table " + tableName)
	}
	for attrName, attr := range node.Attributes {
		if attr.IsStruct {
			continue
		}
		q.Add(attrName)
		q.Add(" ")
		q.Add(postgresTypeOf(attr))
		q.Add(",\n")
	}
	q.Add("CONSTRAINT ", tableName, "_key PRIMARY KEY (ParentKey, RecKey)\n);")
	_, err := this.db.Exec(q.String())
	return err
}

func postgresTypeOf(node *types2.RNode) string {
	if node.IsMap || node.IsSlice {
		return "text"
	}
	switch node.TypeName {
	case "string":
		return "text"
	case "int32":
		return "integer"
	case "int64":
		return "bigint"
	case "float64":
		return "float8"
	case "float32":
		return "real"
	case "bool":
		return "boolean"
	}
	//default to enum for now - @TODO - reflect find what is the kind
	return "integer"
}

func (this *Postgres) Close() error {
	this.db.Close()
	return nil
}
