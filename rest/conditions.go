package sentinel

import (
	"github.com/ospiper/ginx/dbx"
	"github.com/ospiper/ginx/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FilterSet map[string]*Filter

func (s FilterSet) Apply(tx *gorm.DB, model any) *gorm.DB {
	if s == nil {
		return tx
	}
	var clauses []clause.Expression
	for k, f := range s {
		clauses = append(clauses, f.Clause(k, model)...)
	}
	return tx.Clauses(clauses...)
}

type Filter struct {
	Eq         []string `json:"eq,omitempty"`
	NotEq      []string `json:"not_eq,omitempty"`
	Gt         []string `json:"gt,omitempty"`
	Lt         []string `json:"lt,omitempty"`
	Gte        []string `json:"gte,omitempty"`
	Lte        []string `json:"lte,omitempty"`
	Like       []string `json:"like,omitempty"`
	NotLike    []string `json:"not_like,omitempty"`
	Between    []string `json:"between,omitempty"`
	NotBetween []string `json:"not_between,omitempty"`
	In         []string `json:"in,omitempty"`
	NotIn      []string `json:"not_in,omitempty"`
	Regex      []string `json:"regex,omitempty"`
	Ts         []string `json:"ts,omitempty"` // full text
}

/*
verbs:
(not_)eq
(not_)gt
(not_)lt
(not_)gte
(not_)lte
(not_)like
(not_)between
(not_)in
(not_)regex
(not_)ts
*/

func (f *Filter) Clause(field string, model any) []clause.Expression {
	if f == nil {
		return nil
	}
	var ret []clause.Expression
	if len(f.Eq) > 0 {
		ret = append(ret, clause.Eq{Column: field, Value: f.Eq[0]})
	}
	if len(f.NotEq) > 0 {
		ret = append(ret, clause.Neq{Column: field, Value: f.NotEq[0]})
	}
	if len(f.Gt) > 0 {
		ret = append(ret, clause.Gt{Column: field, Value: f.Gt[0]})
	}
	if len(f.Gte) > 0 {
		ret = append(ret, clause.Gte{Column: field, Value: f.Gte[0]})
	}
	if len(f.Lt) > 0 {
		ret = append(ret, clause.Lt{Column: field, Value: f.Lt[0]})
	}
	if len(f.Lte) > 0 {
		ret = append(ret, clause.Lte{Column: field, Value: f.Lte[0]})
	}
	if len(f.Like) > 0 {
		ret = append(ret, clause.Like{Column: field, Value: f.Like[0]})
	}
	if len(f.NotLike) > 0 {
		ret = append(ret, clause.Not(clause.Like{Column: field, Value: f.Like[0]}))
	}
	if len(f.Between) >= 2 {
		for i := 1; i < len(f.Between); i += 2 {
			ret = append(ret, clause.Expr{
				SQL:  "? between (?, ?)",
				Vars: []any{clause.Column{Name: field}, f.Between[i-1], f.Between[i]},
			})
		}
	}
	if len(f.NotBetween) >= 2 {
		for i := 1; i < len(f.Between); i += 2 {
			ret = append(ret, clause.Not(clause.Expr{
				SQL:  "? between (?, ?)",
				Vars: []any{clause.Column{Name: field}, f.Between[i-1], f.Between[i]},
			}))
		}
	}
	if len(f.In) > 0 {
		ret = append(ret, clause.Eq{
			Column: field,
			Value:  f.In,
		})
	}
	if len(f.NotIn) > 0 {
		ret = append(ret, clause.Neq{
			Column: field,
			Value:  f.NotIn,
		})
	}
	if len(f.Regex) > 0 {
		ret = append(ret, clause.Expr{
			SQL:  "? ~* ?",
			Vars: []any{clause.Column{Name: field}, f.Regex[0]},
		})
	}
	if len(f.Ts) > 0 {
		_f := field
		ts, ok := util.As[dbx.FullTextIndexer](model)
		if ok {
			_f = ts.FullTextIndexColumn(field)
		}
		ret = append(ret, clause.Expr{
			SQL:  "? @@ to_tsquery(?)",
			Vars: []any{clause.Column{Name: _f}, f.Ts[0]},
		})
	}
	return ret
}

type Pagination struct {
	Page  int `form:"page" json:"page"`
	Limit int `form:"limit" json:"limit"`
}

func (f *Pagination) Apply(tx *gorm.DB) *gorm.DB {
	if f == nil {
		return tx
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	return tx.Offset((f.Page - 1) * f.Limit).Limit(f.Limit)
}

type Order struct {
	Column string
	Desc   bool
}

func (o *Order) Apply(tx *gorm.DB) *gorm.DB {
	var suffix string
	if o.Desc {
		suffix = " desc"
	}
	return tx.Order(o.Column + suffix)
}

type FindConditions struct {
	Filters    FilterSet
	Orders     []Order
	Pagination *Pagination
}

func (c *FindConditions) Apply(tx *gorm.DB, model any) *gorm.DB {
	if c == nil {
		return tx
	}
	tx = c.Filters.Apply(tx, model)
	for _, order := range c.Orders {
		tx = order.Apply(tx)
	}
	tx = c.Pagination.Apply(tx)

	return tx
}
