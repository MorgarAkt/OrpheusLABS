package repository

import (
	"github.com/morgarakt/aurify/internal/models"
	"gorm.io/gorm"
)

type ModelTypeRepository interface {
	Create(modelType *models.ModelType) error
	Delete(modelType *models.ModelType) error
	GetByID(id any, modelType *models.ModelType) error
	GetByName(name string) (*models.ModelType, error)
	GetAll() ([]models.ModelType, error)
}

type modelTypeRepo struct {
	*GenericRepository[models.ModelType]
	db *gorm.DB
}

func NewModelTypeRepository(db *gorm.DB) ModelTypeRepository {
	return &modelTypeRepo{
		GenericRepository: NewGenericRepository[models.ModelType](db),
		db:                db,
	}
}

func (r *modelTypeRepo) GetByName(name string) (*models.ModelType, error) {
	var modelType models.ModelType
	if err := r.db.Where("name = ?", name).First(&modelType).Error; err != nil {
		return nil, err
	}
	return &modelType, nil
}

func (r *modelTypeRepo) GetAll() ([]models.ModelType, error) {
	var modelTypes []models.ModelType
	if err := r.db.Find(&modelTypes).Error; err != nil {
		return nil, err
	}
	return modelTypes, nil
}
