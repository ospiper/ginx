package rest

import (
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

type Pagination interface {
	IsPagination()
	Apply(tx *gorm.DB) *gorm.DB
	StartIndex() int
	EndIndex() int
}

func PaginationHeader(p Pagination, total int64) (int, string) {
	code := http.StatusPartialContent
	if p.StartIndex() == 0 && p.EndIndex() >= int(total)-1 {
		code = http.StatusOK
	}
	return code, fmt.Sprintf("items %d-%d/%d", p.StartIndex(), p.EndIndex(), total)
}

type Range struct {
	Start int
	End   int
}

func (*Range) IsPagination() {}

func (r *Range) Apply(tx *gorm.DB) *gorm.DB {
	if r == nil {
		return tx
	}
	return tx.Offset(r.Start).Limit(r.End - r.Start + 1)
}

func (r *Range) StartIndex() int {
	return r.Start
}

func (r *Range) EndIndex() int {
	return r.End
}

type Page struct {
	Page  int `form:"page" json:"page"`
	Limit int `form:"limit" json:"limit"`
}

func (*Page) IsPagination() {}

func (f *Page) Apply(tx *gorm.DB) *gorm.DB {
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

func (f *Page) StartIndex() int {
	return (f.Page - 1) * f.Limit
}

func (f *Page) EndIndex() int {
	return f.Page*f.Limit - 1
}
