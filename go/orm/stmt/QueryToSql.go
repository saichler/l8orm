/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package stmt

import (
	"bytes"
	"fmt"
	"github.com/saichler/l8types/go/ifs"
	"reflect"
)

func (this *Statement) Query2CountSql(query ifs.IQuery, typeName string) string {
	buff := bytes.Buffer{}
	buff.WriteString("SELECT COUNT(*) FROM ")
	buff.WriteString(typeName)

	if query != nil && query.Criteria() != nil && typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" WHERE ")
			buff.WriteString(str)
		}
	}
	return buff.String()
}

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
		buff.WriteString(stripQuotes(comp.Right()))
		buff.WriteString("'")
	} else if !leftString && rightString {
		buff.WriteString("'")
		buff.WriteString(stripQuotes(comp.Left()))
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

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// Query2RecKeysSql generates SQL to fetch only RecKeys (no LIMIT/OFFSET)
// Used by PrimaryIndex to cache all matching RecKeys for pagination
func (this *Statement) Query2RecKeysSql(query ifs.IQuery, typeName string) string {
	buff := bytes.Buffer{}
	buff.WriteString("SELECT RecKey FROM ")
	buff.WriteString(typeName)

	if query.Criteria() != nil && typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" WHERE ")
			buff.WriteString(str)
		}
	}

	// Add ORDER BY (always include, no LIMIT/OFFSET)
	if query.SortBy() != "" {
		buff.WriteString(" ORDER BY ")
		buff.WriteString(query.SortBy())
		if query.Descending() {
			buff.WriteString(" DESC")
		} else {
			buff.WriteString(" ASC")
		}
	}

	return buff.String()
}

// Query2SqlByRecKeys generates SQL to fetch rows by specific RecKeys
// Used by PrimaryIndex to fetch full data for a page of RecKeys
func (this *Statement) Query2SqlByRecKeys(typeName string, recKeys []string) string {
	buff := bytes.Buffer{}
	buff.WriteString("SELECT ")

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

	buff.WriteString(" FROM ")
	buff.WriteString(typeName)
	buff.WriteString(" WHERE RecKey IN (")

	first = true
	for _, key := range recKeys {
		if !first {
			buff.WriteString(",")
		}
		first = false
		buff.WriteString("'")
		buff.WriteString(escapeSQL(key))
		buff.WriteString("'")
	}
	buff.WriteString(")")

	return buff.String()
}

// escapeSQL escapes single quotes in SQL string values
func escapeSQL(s string) string {
	result := bytes.Buffer{}
	for _, c := range s {
		if c == '\'' {
			result.WriteRune('\'')
		}
		result.WriteRune(c)
	}
	return result.String()
}
