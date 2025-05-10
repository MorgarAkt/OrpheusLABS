package repository

import "gorm.io/gorm"

type Repository struct {
	User      UserRepository
	Music     MusicRepository
	MusicType MusicTypeRepository
	ModelType ModelTypeRepository
	UserLikes UserLikesRepository
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		User:      NewUserRepository(db),
		Music:     NewMusicRepository(db),
		MusicType: NewMusicTypeRepository(db),
		ModelType: NewModelTypeRepository(db),
		UserLikes: NewUserLikesRepository(db),
	}
}
