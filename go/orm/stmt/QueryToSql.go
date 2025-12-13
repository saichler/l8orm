package stmt

import (
	"bytes"
	"fmt"
	"github.com/saichler/l8types/go/ifs"
	"reflect"
)

func (this *Statement) Query2Sql(query ifs.IQuery, typeName string) (string, bool) {
	buff := bytes.Buffer{}
	if query.Properties() == nil || len(query.Properties()) == 0 {
		buff.WriteString("Select ")
		if this.fields == nil {
			this.fields, this.values = fieldsOf(this.node)
		}
		first := true
		for _, fieldName := range this.fields {
			if !first {
				buff.WriteString(",")
			}
			first = false
			buff.WriteString(fieldName)
		}
	} else {
		buff.WriteString("Select ParentKey,RecKey")
		this.fields = []string{"ParentKey", "RecKey"}
		for _, prop := range query.Properties() {
			buff.WriteString(",")
			if prop.Node().Parent.TypeName == typeName {
				this.fields = append(this.fields, prop.Node().FieldName)
				buff.WriteString(prop.Node().FieldName)
			}
		}
		if len(this.fields) == 2 {
			return "", false
		}
	}
	buff.WriteString(" from ")
	buff.WriteString(typeName)

	if query.Criteria() == nil {
		return buff.String(), true
	}

	if typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" where ")
			buff.WriteString(str)
		}

		// Add ORDER BY clause if SortBy is specified
		if query.SortBy() != "" {
			buff.WriteString(" ORDER BY ")
			buff.WriteString(query.SortBy())
			if query.Descending() {
				buff.WriteString(" DESC")
			} else {
				buff.WriteString(" ASC")
			}
		}

		// Add LIMIT clause if Limit is specified
		if query.Limit() > 0 {
			buff.WriteString(fmt.Sprintf(" LIMIT %d", query.Limit()))
		}

		// Add OFFSET clause for pagination (Page starts from 0)
		if query.Page() > 0 && query.Limit() > 0 {
			offset := query.Page() * query.Limit()
			buff.WriteString(fmt.Sprintf(" OFFSET %d", offset))
		}
	}
	fmt.Println(buff.String())
	return buff.String(), true
}

func expression(exp ifs.IExpression, typeName string) (bool, string) {
	if isNil(exp) {
		return false, ""
	}

	buff := bytes.Buffer{}
	condOK, condStr := condition(exp.Condition(), typeName)
	if condOK {
		buff.WriteString("(")
		buff.WriteString(condStr)
	}

	nextOK, nextStr := expression(exp.Next(), typeName)
	if nextOK {
		if !condOK {
			buff.WriteString("(")
		}
		buff.WriteString(exp.Operator())
		buff.WriteString(nextStr)
	}
	if nextOK || condOK {
		buff.WriteString(")")
	}
	return condOK || nextOK, buff.String()
}

func condition(cond ifs.ICondition, typeName string) (bool, string) {
	if isNil(cond) {
		return false, ""
	}
	result := bytes.Buffer{}
	okCond, exp1 := comparator(cond.Comparator(), typeName)
	if okCond {
		result.WriteString(exp1)
	}
	okNext, exp2 := condition(cond.Next(), typeName)
	if okNext {
		result.WriteString(cond.Operator())
		result.WriteString(exp2)
	}
	return okCond || okNext, result.String()
}

func comparator(comp ifs.IComparator, typeName string) (bool, string) {
	if isNil(comp) {
		return false, ""
	}
	leftOK := false
	leftString := false
	rightOK := false
	rightString := false

	if !isNil(comp.LeftProperty()) && comp.LeftProperty().Node().Parent.TypeName == typeName {
		leftOK = true
		if comp.LeftProperty().IsString() {
			leftString = true
		}
	}

	if !isNil(comp.RightProperty()) && comp.RightProperty().Node().Parent.TypeName == typeName {
		rightOK = true
		if comp.RightProperty().IsString() {
			rightString = true
		}
	}

	buff := bytes.Buffer{}
	if leftString && !rightString {
		buff.WriteString(comp.Left())
		buff.WriteString(comp.Operator())
		buff.WriteString("'")
		buff.WriteString(comp.Right())
		buff.WriteString("'")
	} else if !leftString && rightString {
		buff.WriteString("'")
		buff.WriteString(comp.Left())
		buff.WriteString("'")
		buff.WriteString(comp.Operator())
		buff.WriteString(comp.Right())
	} else {
		buff.WriteString(comp.Left())
		buff.WriteString(comp.Operator())
		buff.WriteString(comp.Right())
	}
	return leftOK || rightOK, buff.String()
}

func isNil(any interface{}) bool {
	if any == nil {
		return true
	}
	return reflect.ValueOf(any).IsNil()
}
