package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
)

var filterVerbs = []struct {
	suffix string
	build  func(k string, v any) FilterFunc
}{
	{"_j_inc_any", JIncAny},
	{"_j_inc_all", JIncAll},
	{"_j_eq", JEq},
	{"_is_null", IsNull},
	{"_neq_any", Neq},
	{"_inc_any", IncAny},
	{"_eq_any", Eq},
	{"_between", Between},
	{"_regex", Regex},
	{"_ilike", ILike},
	{"_like", Like},
	{"_neq", Neq},
	{"_eq", Eq},
	{"_q", Q},
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
		for _, s := range explode(v) {
			exp = append(exp, clause.Like{
				Column: k,
				Value:  fmt.Sprintf("%%%s%%", s),
			})
		}
		return clause.Or(exp...), nil
	}
}

func JIncAny(k string, v any) FilterFunc {
	d := dialect
	return func() (clause.Expression, error) {
		keys, err := jsonKeys(v)
		if err != nil {
			return nil, err
		}
		return d.JSONIncludesAny(k, keys), nil
	}
}

func JIncAll(k string, v any) FilterFunc {
	d := dialect
	return func() (clause.Expression, error) {
		keys, err := jsonKeys(v)
		if err != nil {
			return nil, err
		}
		return d.JSONIncludesAll(k, keys), nil
	}
}

func JEq(k string, v any) FilterFunc {
	d := dialect
	return func() (clause.Expression, error) {
		return d.JSONContains(k, v)
	}
}

func Like(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		exp := make([]clause.Expression, 0)
		for _, s := range explode(v) {
			exp = append(exp, clause.Like{
				Column: k,
				Value:  fmt.Sprintf("%%%s%%", s),
			})
		}
		return clause.And(exp...), nil
	}
}

func ILike(k string, v any) FilterFunc {
	return func() (clause.Expression, error) {
		exp := make([]clause.Expression, 0)
		for _, s := range explode(v) {
			exp = append(exp, clause.Expr{
				SQL: "? ILIKE ?",
				Vars: []any{
					clause.Column{Name: k},
					fmt.Sprintf("%%%s%%", s),
				},
			})
		}
		return clause.And(exp...), nil
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
	matched := -1
	for i, verb := range filterVerbs {
		if strings.HasSuffix(k, verb.suffix) && (matched < 0 || len(verb.suffix) > len(filterVerbs[matched].suffix)) {
			matched = i
		}
	}
	if matched < 0 {
		return k, Eq(k, v)
	}
	verb := filterVerbs[matched]
	field := k[:len(k)-len(verb.suffix)]
	return field, verb.build(field, v)
}

func jsonKeys(v any) ([]string, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, errors.New("json key predicate: expect an array of strings")
	}
	keys := make([]string, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		key, ok := rv.Index(i).Interface().(string)
		if !ok {
			return nil, errors.New("json key predicate: expect an array of strings")
		}
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("json key predicate: expect at least one key")
	}
	return keys, nil
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

func explode(v any) []string {
	var ret []string
	for _, t := range asSlice(v) {
		for _, i := range strings.Split(fmt.Sprint(t), " ") {
			r := strings.TrimSpace(i)
			if r == "" {
				continue
			}
			ret = append(ret, r)
		}
	}
	return ret
}
