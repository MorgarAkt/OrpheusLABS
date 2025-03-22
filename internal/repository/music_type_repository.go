package repository

import (
	"github.com/morgarakt/aurify/internal/models"
	"gorm.io/gorm"
)

type MusicTypeRepository interface {
	Create(musicType *models.MusicType) error
	Delete(musicType *models.MusicType) error
	GetByID(id any, musicType *models.MusicType) error
	GetByName(name string) (*models.MusicType, error)
	GetAll() ([]models.MusicType, error)
}

type musicTypeRepo struct {
	*GenericRepository[models.MusicType]
	db *gorm.DB
}

func NewMusicTypeRepository(db *gorm.DB) MusicTypeRepository {
	return &musicTypeRepo{
		GenericRepository: NewGenericRepository[models.MusicType](db),
		db:                db,
	}
}

func (r *musicTypeRepo) GetByName(name string) (*models.MusicType, error) {
	var musicType models.MusicType
	if err := r.db.Where("name = ?", name).First(&musicType).Error; err != nil {
		return nil, err
	}
	return &musicType, nil
}

func (r *musicTypeRepo) GetAll() ([]models.MusicType, error) {
	var musicTypes []models.MusicType
	if err := r.db.Find(&musicTypes).Error; err != nil {
		return nil, err
	}
	return musicTypes, nil
}
