package rest

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FilterFunc func() (clause.Expression, error)
type FindConditions struct {
	Filters    []FilterFunc
	Orders     []Order
	Pagination Pagination
	Preloads   []string
}

func (c *FindConditions) Apply(tx *gorm.DB) (*gorm.DB, error) {
	if c == nil {
		return tx, nil
	}
	clauses, err := ApplyFilterFunc(c.Filters)
	if err != nil {
		return nil, err
	}
	tx = tx.Clauses(clauses...)
	for _, order := range c.Orders {
		tx = order.Apply(tx)
	}
	fmt.Println(c.Pagination)
	tx = c.Pagination.Apply(tx)
	for _, p := range c.Preloads {
		tx = tx.Preload(p)
	}
	return tx, nil
}

func ApplyFilterFunc(fns []FilterFunc) ([]clause.Expression, error) {
	clauses := make([]clause.Expression, 0, len(fns))
	for _, fn := range fns {
		cl, err := fn()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, cl)
	}
	return clauses, nil
}
