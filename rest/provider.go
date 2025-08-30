package rest

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ospiper/ginx/dbx"
	"github.com/ospiper/ginx/util"
)

const (
	defaultBatchSize = 100
)

type Provider[T dbx.ModelStruct[T]] interface {
	GetDB() *gorm.DB
	Model(ctx context.Context) *gorm.DB
	Migrate() error

	FindOne(ctx context.Context, id int64) (*T, error)
	Find(ctx context.Context, conditions *FindConditions) ([]*T, error)
	FindAssoc(ctx context.Context, parentModel any, assocName string, conditions *FindConditions) ([]*T, error)
	Count(ctx context.Context, filter []FilterFunc) (int64, error)
	CountAssoc(ctx context.Context, parentModel any, assocName string, filter []FilterFunc) (int64, error)

	Insert(ctx context.Context, v *T) error
	InsertMany(ctx context.Context, vs []*T) error
	InsertBatch(ctx context.Context, vs []*T, batchSize int) error

	Update(ctx context.Context, id int64, v *T) error
	UpdateFields(ctx context.Context, id int64, fields map[string]any) (*T, error)

	Delete(ctx context.Context, id int64) error
	DeleteMany(ctx context.Context, ids []int64) error
}

type providerImpl[T dbx.ModelStruct[T]] struct {
	db *gorm.DB
}

func NewProvider[T dbx.ModelStruct[T]](db *gorm.DB) Provider[T] {
	// todo field check
	// but how to use the sync map for parsing fields without interrupting gorm mechanics

	//var t T
	//ns := schema.NamingStrategy{}
	//cache := &sync.Map{}
	//sc, err := schema.Parse(&t, cache, ns)
	//if err != nil {
	//	panic(err)
	//}
	//for _, f := range sc.Fields {
	//	fmt.Println(f.DBName)
	//}
	return &providerImpl[T]{db: db}
}

type _assertion struct {
	dbx.Model
}

func (a _assertion) NewWithID(id int64) _assertion {
	return _assertion{dbx.Model{ID: id}}
}

var (
	// compile time type assertion
	_ = NewProvider[_assertion](nil)

	ErrNotFound     = errors.New("record not found")
	ErrNotDeletable = errors.New("cannot delete record")
)

func (w *providerImpl[T]) GetDB() *gorm.DB {
	return w.db
}

func (w *providerImpl[T]) Model(ctx context.Context) *gorm.DB {
	var m T
	return w.db.WithContext(ctx).Model(&m)
}

func (w *providerImpl[T]) Migrate() error {
	var m T
	return w.db.AutoMigrate(&m)
}

func (w *providerImpl[T]) FindOne(ctx context.Context, id int64) (*T, error) {
	ret := new(T)
	tx := w.db.WithContext(ctx).Model(ret)
	pld, ok := util.As[dbx.Preloader](ret)
	if ok {
		for _, c := range pld.Preloads() {
			tx = tx.Preload(c)
		}
	}

	err := tx.
		First(ret, id).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ret, nil
}

func (w *providerImpl[T]) Find(ctx context.Context, conditions *FindConditions) ([]*T, error) {
	var res []*T
	var m T
	tx := w.db.WithContext(ctx)
	tx, err := conditions.Apply(tx)
	if err != nil {
		return nil, err
	}

	pld, ok := util.As[dbx.Preloader](m)
	if ok {
		//fmt.Println("preload", pld.Preloads())
		for _, c := range pld.Preloads() {
			tx = tx.Preload(c)
		}
	}

	err = tx.Find(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (w *providerImpl[T]) FindAssoc(ctx context.Context, parentModel any, assocName string, conditions *FindConditions) ([]*T, error) {
	var res []*T
	var m T
	tx := w.db.WithContext(ctx).Model(parentModel)
	tx, err := conditions.Apply(tx)
	if err != nil {
		return nil, err
	}
	pld, ok := util.As[dbx.Preloader](m)
	if ok {
		//fmt.Println("preload", pld.Preloads())
		for _, c := range pld.Preloads() {
			tx = tx.Preload(c)
		}
	}

	err = tx.Association(assocName).Find(&res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (w *providerImpl[T]) Count(ctx context.Context, filters []FilterFunc) (int64, error) {
	var m T
	var cnt int64
	tx := w.db.WithContext(ctx).Model(&m)
	clauses, err := ApplyFilterFunc(filters)
	if err != nil {
		return 0, err
	}
	tx = tx.Clauses(clauses...)
	err = tx.Count(&cnt).
		Error
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (w *providerImpl[T]) CountAssoc(ctx context.Context, parentModel any, assocName string, filters []FilterFunc) (int64, error) {
	tx := w.db.WithContext(ctx).Model(parentModel)
	clauses, err := ApplyFilterFunc(filters)
	if err != nil {
		return 0, err
	}
	tx = tx.Clauses(clauses...)
	assoc := tx.Association(assocName)
	return assoc.Count(), assoc.Error
}

func (w *providerImpl[T]) Insert(ctx context.Context, v *T) error {
	return w.db.WithContext(ctx).
		Create(v).
		Error
}

func (w *providerImpl[T]) InsertMany(ctx context.Context, vs []*T) error {
	return w.InsertBatch(ctx, vs, defaultBatchSize)
}

func (w *providerImpl[T]) InsertBatch(ctx context.Context, vs []*T, batchSize int) error {
	return w.db.WithContext(ctx).
		CreateInBatches(vs, batchSize).
		Error
}

func (w *providerImpl[T]) Update(ctx context.Context, id int64, v *T) error {
	err := w.db.WithContext(ctx).Model(v).
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Omit("id").
		Updates(v).
		Limit(1).
		Error
	if err != nil {
		return err
	}
	return nil
}

func (w *providerImpl[T]) UpdateFields(ctx context.Context, id int64, fields map[string]any) (*T, error) {
	var m T
	err := w.db.WithContext(ctx).Model(&m).
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Omit("id").
		Updates(fields).
		Limit(1).
		Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type WithDeletableCheck interface {
	Deletable(ctx context.Context) bool
}

// Delete if it's soft delete how to locate and restore it on insert conflict?
func (w *providerImpl[T]) Delete(ctx context.Context, id int64) error {
	var m T
	res, err := w.FindOne(ctx, id)
	if err != nil {
		return err
	}
	var deleter any = res
	if d, ok := deleter.(WithDeletableCheck); ok {
		if !d.Deletable(ctx) {
			return ErrNotDeletable
		}
	}
	return w.db.WithContext(ctx).
		Delete(&m, id).
		Limit(1).
		Error
}

func (w *providerImpl[T]) DeleteMany(ctx context.Context, ids []int64) error {
	var m T
	return w.db.WithContext(ctx).
		Delete(&m, ids).
		Limit(len(ids)).
		Error
}
