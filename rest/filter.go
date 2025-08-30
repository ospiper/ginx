package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
)

var filterVerbs = map[string]func(k string, v any) FilterFunc{
	"_eq":      Eq,
	"_eq_any":  Eq,
	"_neq":     Neq,
	"_neq_any": Neq,
	"_inc_any": IncAny,
	"_is_null": IsNull,
	"_regex":   Regex,
	"_between": Between,
	"_like":    IncAny,
	"_q":       Q,
}

func Eq(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		return clause.Eq{Column: k, Value: v}, nil
	}
}

func Neq(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		return clause.Neq{Column: k, Value: v}, nil
	}
}

func IncAny(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		exp := make([]clause.Expression, 0)
		for _, s := range asSlice(v) {
			exp = append(exp, clause.Like{
				Column: k,
				Value:  fmt.Sprintf("%%%s%%", s),
			})
		}
		return clause.Or(exp...), nil
	}
}

func IsNull(k string, _ any) FilterFunc {
	return func() (clause.Expression, error) {
		return clause.Expr{
			SQL:  "? IS NULL",
			Vars: []any{clause.Column{Name: k}},
		}, nil
	}
}

func Regex(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		return clause.Expr{
			SQL:  "? ~* ?",
			Vars: []any{clause.Column{Name: k}, v},
		}, nil
	}
}

func Between(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		vs := asSlice(v)
		if len(vs) != 2 {
			return nil, errors.New("between: expect 2 arguments")
		}
		return clause.Expr{
			SQL:  "? between (?, ?)",
			Vars: []any{clause.Column{Name: k}, vs[0], vs[1]},
		}, nil
	}
}

func Q(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		return clause.Expr{
			SQL:  "? @@ to_tsquery(?)",
			Vars: []any{clause.Column{Name: k + "_index"}, v},
		}, nil
	}
}

func buildFilters(fs string) ([]FilterFunc, error) {
	f := make(map[string]any)
	err := json.Unmarshal([]byte(fs), &f)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}
	var ret []FilterFunc
	for k, v := range f {
		_, expr := parseFilter(k, v)
		ret = append(ret, expr)
	}
	return ret, nil
}

func parseFilter(k string, v any) (string, FilterFunc) {
	// parse k
	// could be field or field_verb
	var field string
	var expr FilterFunc
	for verbSuffix, fn := range filterVerbs {
		if strings.HasSuffix(k, verbSuffix) {
			field = k[:len(k)-len(verbSuffix)]
			expr = fn(field, v)
			break
		}
	}
	if expr == nil {
		field = k
		expr = Eq(k, v)
	}
	return field, expr
}

func asSlice(v any) []any {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []any{v}
	}
	ret := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		ret[i] = rv.Index(i).Interface()
	}
	return ret
}
