package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/models"
	"gorm.io/gorm"
)

type UserLikesRepository interface {
	HasUserLiked(userID, musicID uuid.UUID) (bool, error)
	AddLike(like *models.UserLikesMusic) error
	RemoveLike(userID, musicID uuid.UUID) error
	GetLikesForMusic(musicID uuid.UUID) ([]models.UserLikesMusic, error) // Opsiyonel
}

type userLikesRepo struct {
	db *gorm.DB
}

func NewUserLikesRepository(db *gorm.DB) UserLikesRepository {
	return &userLikesRepo{db: db}
}

func (r *userLikesRepo) HasUserLiked(userID, musicID uuid.UUID) (bool, error) {
	var like models.UserLikesMusic
	err := r.db.Where("user_id = ? AND music_id = ?", userID, musicID).First(&like).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // Kayıt bulunamadı = beğenmemiş
		}
		return false, err // Başka bir veritabanı hatası
	}
	return true, nil // Kayıt bulundu = beğenmiş
}

func (r *userLikesRepo) AddLike(like *models.UserLikesMusic) error {
	return r.db.Create(like).Error
}

func (r *userLikesRepo) RemoveLike(userID, musicID uuid.UUID) error {
	return r.db.Where("user_id = ? AND music_id = ?", userID, musicID).Delete(&models.UserLikesMusic{}).Error
}

func (r *userLikesRepo) GetLikesForMusic(musicID uuid.UUID) ([]models.UserLikesMusic, error) {
	var likes []models.UserLikesMusic
	err := r.db.Where("music_id = ?", musicID).Find(&likes).Error
	return likes, err
}
