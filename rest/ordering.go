package rest

import (
	"gorm.io/gorm"
)

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
