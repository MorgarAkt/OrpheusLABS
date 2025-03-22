package repository

import (
	"github.com/morgarakt/aurify/internal/models"
	"gorm.io/gorm"
)

type MusicRepository interface {
	Create(music *models.Music) error
	Delete(music *models.Music) error
	GetByID(id any, music *models.Music) error
	GetAll() ([]models.Music, error)
}

type musicRepo struct {
	*GenericRepository[models.Music]
	db *gorm.DB
}

func NewMusicRepository(db *gorm.DB) MusicRepository {
	return &musicRepo{
		GenericRepository: NewGenericRepository[models.Music](db),
		db:                db,
	}
}

func (r *musicRepo) GetAll() ([]models.Music, error) {
	var musicList []models.Music
	if err := r.db.Preload("User").Preload("MusicType").Preload("ModelType").Find(&musicList).Error; err != nil {
		return nil, err
	}
	return musicList, nil
}

func (r *musicRepo) GetByCreatorId() ([]models.Music, error) {
	var musicList []models.Music
	return musicList, nil
}
