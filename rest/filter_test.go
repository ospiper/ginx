package rest

import (
	"reflect"
	"testing"

	"gorm.io/gorm/clause"
)

func TestPostgresDialectJSONIncludesAny(t *testing.T) {
	expr := PostgresDialect{}.JSONIncludesAny("meta", []string{"aa", "bb", "cc"})

	got, ok := expr.(clause.Expr)
	if !ok {
		t.Fatalf("JSONIncludesAny() returned %T, want clause.Expr", expr)
	}
	wantSQL := "jsonb_exists_any(?::jsonb, ARRAY[?,?,?]::text[])"
	if got.SQL != wantSQL {
		t.Errorf("JSONIncludesAny() SQL = %q, want %q", got.SQL, wantSQL)
	}
	wantVars := []any{clause.Column{Name: "meta"}, "aa", "bb", "cc"}
	if !reflect.DeepEqual(got.Vars, wantVars) {
		t.Errorf("JSONIncludesAny() Vars = %#v, want %#v", got.Vars, wantVars)
	}
}

func TestPostgresDialectJSONIncludesAll(t *testing.T) {
	expr := PostgresDialect{}.JSONIncludesAll("meta", []string{"aa", "bb"})

	got, ok := expr.(clause.Expr)
	if !ok {
		t.Fatalf("JSONIncludesAll() returned %T, want clause.Expr", expr)
	}
	wantSQL := "jsonb_exists_all(?::jsonb, ARRAY[?,?]::text[])"
	if got.SQL != wantSQL {
		t.Errorf("JSONIncludesAll() SQL = %q, want %q", got.SQL, wantSQL)
	}
	wantVars := []any{clause.Column{Name: "meta"}, "aa", "bb"}
	if !reflect.DeepEqual(got.Vars, wantVars) {
		t.Errorf("JSONIncludesAll() Vars = %#v, want %#v", got.Vars, wantVars)
	}
}

func TestPostgresDialectJSONContains(t *testing.T) {
	expr, err := PostgresDialect{}.JSONContains("meta", map[string]any{
		"name": "aaaa",
		"year": float64(1999),
	})
	if err != nil {
		t.Fatalf("JSONContains() error = %v", err)
	}

	got, ok := expr.(clause.Expr)
	if !ok {
		t.Fatalf("JSONContains() returned %T, want clause.Expr", expr)
	}
	if got.SQL != "?::jsonb @> ?::jsonb" {
		t.Errorf("JSONContains() SQL = %q, want %q", got.SQL, "?::jsonb @> ?::jsonb")
	}
	wantVars := []any{clause.Column{Name: "meta"}, `{"name":"aaaa","year":1999}`}
	if !reflect.DeepEqual(got.Vars, wantVars) {
		t.Errorf("JSONContains() Vars = %#v, want %#v", got.Vars, wantVars)
	}
}

func TestSetDialectRejectsNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("SetDialect(nil) did not panic")
		}
	}()

	SetDialect(nil)
}

func TestParseFilterJSONIncludesAny(t *testing.T) {
	SetDialect(PostgresDialect{})
	field, filter := parseFilter("meta_j_inc_any", []any{" aa ", "", "bb", "aa"})
	if field != "meta" {
		t.Fatalf("parseFilter() field = %q, want %q", field, "meta")
	}

	expr, err := filter()
	if err != nil {
		t.Fatalf("JSON includes-any filter error = %v", err)
	}
	got, ok := expr.(clause.Expr)
	if !ok {
		t.Fatalf("JSON includes-any filter returned %T, want clause.Expr", expr)
	}
	wantVars := []any{clause.Column{Name: "meta"}, "aa", "bb", "aa"}
	if !reflect.DeepEqual(got.Vars, wantVars) {
		t.Errorf("JSON includes-any Vars = %#v, want %#v", got.Vars, wantVars)
	}
}

func TestParseFilterJSONIncludesAll(t *testing.T) {
	SetDialect(PostgresDialect{})
	field, filter := parseFilter("meta_j_inc_all", []string{"aa", "bb"})
	if field != "meta" {
		t.Fatalf("parseFilter() field = %q, want %q", field, "meta")
	}

	expr, err := filter()
	if err != nil {
		t.Fatalf("JSON includes-all filter error = %v", err)
	}
	got, ok := expr.(clause.Expr)
	if !ok {
		t.Fatalf("JSON includes-all filter returned %T, want clause.Expr", expr)
	}
	if got.SQL != "jsonb_exists_all(?::jsonb, ARRAY[?,?]::text[])" {
		t.Errorf("JSON includes-all SQL = %q", got.SQL)
	}
}

func TestParseFilterJSONEqualUsesContainment(t *testing.T) {
	SetDialect(PostgresDialect{})
	field, filter := parseFilter("meta_j_eq", map[string]any{
		"name": "aaaa",
		"year": float64(1999),
	})
	if field != "meta" {
		t.Fatalf("parseFilter() field = %q, want %q", field, "meta")
	}

	expr, err := filter()
	if err != nil {
		t.Fatalf("JSON equality filter error = %v", err)
	}
	got, ok := expr.(clause.Expr)
	if !ok {
		t.Fatalf("JSON equality filter returned %T, want clause.Expr", expr)
	}
	if got.SQL != "?::jsonb @> ?::jsonb" {
		t.Errorf("JSON equality SQL = %q", got.SQL)
	}
}

func TestJSONKeyPredicatesRejectInvalidArrays(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "string", value: "aa,bb"},
		{name: "mixed element types", value: []any{"aa", float64(1)}},
		{name: "empty array", value: []string{}},
		{name: "only blank keys", value: []string{" ", "\t"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, filter := parseFilter("meta_j_inc_any", tt.value)
			if _, err := filter(); err == nil {
				t.Fatal("JSON key predicate returned nil error")
			}
		})
	}
}

func Test_explode(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "single string",
			args: args{
				v: "hello world",
			},
			want: []string{"hello", "world"},
		},
		{
			name: "multiple strings",
			args: args{
				v: []string{"hello 1", "world  2"},
			},
			want: []string{"hello", "1", "world", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := explode(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("explode() = %v, want %v", got, tt.want)
			}
		})
	}
}
