package models

import (
	"time"

	"github.com/google/uuid"
)

type Music struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Title        string
	FilePath     string
	LikesCount   int `gorm:"default:0"`
	ListensCount int `gorm:"default:0"`
	UserID       uuid.UUID
	User         User
	MusicTypeID  uuid.UUID
	MusicType    MusicType
	ModelTypeID  uuid.UUID
	ModelType    ModelType
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
