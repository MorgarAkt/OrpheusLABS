package repository

import "gorm.io/gorm"

type GenericRepository[T any] struct {
	db *gorm.DB
}

func NewGenericRepository[T any](db *gorm.DB) *GenericRepository[T] {
	return &GenericRepository[T]{db: db}
}

func (r *GenericRepository[T]) Create(entity *T) error {
	return r.db.Create(entity).Error
}

func (r *GenericRepository[T]) Delete(entity *T) error {
	return r.db.Delete(entity).Error
}

func (r *GenericRepository[T]) GetByID(id any, entity *T) error {
	result := r.db.First(entity, id)
	return result.Error
}
