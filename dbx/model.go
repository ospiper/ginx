package dbx

import (
	"time"

	"gorm.io/gorm"
)

// ModelStruct receives an assigned id parameter and returns a NEW model instance.
type ModelStruct[T any] interface {
	ID(id int64) T
}

type FullTextIndexer interface {
	FullTextIndexColumn(baseColumn string) string
}

type Preloader interface {
	Preloads() []string
}

type Model struct {
	ID        int64          `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type Deletable struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
