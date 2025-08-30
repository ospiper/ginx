package dbx

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// ModelStruct receives an assigned id parameter and returns a NEW model instance.
type ModelStruct[T any] interface {
	NewWithID(id int64) T
}

type Preloader interface {
	Preloads() []string
}

type WithID interface {
	GetID() int64
}

type Model struct {
	ID        int64          `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (m Model) GetID() int64 {
	return m.ID
}

type Deletable struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d Deletable) GetID() int64 {
	return d.ID
}

type Permanent struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (p Permanent) GetID() int64 {
	return p.ID
}

func (Permanent) BeforeDelete(tx *gorm.DB) error {
	return errors.New("cannot delete permanent model")
}
