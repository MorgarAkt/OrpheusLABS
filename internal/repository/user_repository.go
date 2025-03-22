package repository

import (
	"github.com/morgarakt/aurify/internal/models"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	Delete(user *models.User) error
	GetByID(id any, user *models.User) error
	GetByEmail(email string) (*models.User, error)
}

type userRepo struct {
	*GenericRepository[models.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{
		GenericRepository: NewGenericRepository[models.User](db),
		db:                db,
	}
}

func (r *userRepo) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
