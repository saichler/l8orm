package persist

import (
	"errors"
	"reflect"

	"github.com/saichler/l8ql/go/gsql/interpreter"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/strings"
)

func ElementToQuery(pb ifs.IElements, elem interface{}, vnic ifs.IVNic) (ifs.IQuery, error) {
	asideValue := reflect.ValueOf(pb.Element())
	aside := asideValue.Elem().Type().Name()
	bside := reflect.ValueOf(elem).Elem().Type().Name()
	if aside == bside {
		rnode, ok := vnic.Resources().Introspector().NodeByTypeName(bside)
		if ok {
			gsql := strings.New("select * from ", bside, " where ")
			fields, e := vnic.Resources().Introspector().Decorators().Fields(rnode, l8reflect.L8DecoratorType_Primary)
			if e != nil {
				return nil, e
			}
			for i, field := range fields {
				if i > 0 {
					gsql.Add(" and ")
				}
				v := asideValue.Elem().FieldByName(field)
				gsql.Add(field)
				gsql.Add("=")
				if v.Kind() == reflect.String {
					gsql.Add("'", v.String(), "'")
				} else {
					gsql.Add(v.String())
				}

			}

			q, e := interpreter.NewQuery(gsql.String(), vnic.Resources())
			return q, e
		}
	}
	return nil, errors.New("Element does not match " + bside + " != " + aside)
}

func KeyOf(elements ifs.IElements, resources ifs.IResources) string {
	key, _, err := resources.Introspector().Decorators().PrimaryKeyDecoratorValue(elements.Element())
	if err != nil {
		resources.Logger().Error(err.Error())
	}
	return key
}
