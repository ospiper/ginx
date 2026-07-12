package rest

import (
	"encoding/json"
	"reflect"
	"strings"

	"gorm.io/gorm/clause"
)

// Dialect builds database-specific JSON filter expressions.
type Dialect interface {
	JSONIncludesAny(column string, keys []string) clause.Expression
	JSONIncludesAll(column string, keys []string) clause.Expression
	JSONContains(column string, value any) (clause.Expression, error)
}

var dialect Dialect = PostgresDialect{}

// SetDialect configures the process-wide filter dialect. It must be called
// before the application starts serving requests.
func SetDialect(d Dialect) {
	if d == nil || isNilDialect(d) {
		panic("rest: nil dialect")
	}
	dialect = d
}

func isNilDialect(d Dialect) bool {
	v := reflect.ValueOf(d)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// PostgresDialect builds PostgreSQL-specific filter expressions.
type PostgresDialect struct{}

// JSONIncludesAny tests whether a JSON value contains any of the top-level keys.
func (PostgresDialect) JSONIncludesAny(column string, keys []string) clause.Expression {
	return postgresJSONKeyExists("jsonb_exists_any", column, keys)
}

// JSONIncludesAll tests whether a JSON value contains all of the top-level keys.
func (PostgresDialect) JSONIncludesAll(column string, keys []string) clause.Expression {
	return postgresJSONKeyExists("jsonb_exists_all", column, keys)
}

func postgresJSONKeyExists(function, column string, keys []string) clause.Expression {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(keys)), ",")
	vars := make([]any, 1, len(keys)+1)
	vars[0] = clause.Column{Name: column}
	for _, key := range keys {
		vars = append(vars, key)
	}
	return clause.Expr{
		SQL:  function + "(?::jsonb, ARRAY[" + placeholders + "]::text[])",
		Vars: vars,
	}
}

// JSONContains tests whether a JSON value contains the requested JSON structure.
func (PostgresDialect) JSONContains(column string, value any) (clause.Expression, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return clause.Expr{
		SQL:  "?::jsonb @> ?::jsonb",
		Vars: []any{clause.Column{Name: column}, string(encoded)},
	}, nil
}
