package models

import (
	"time"

	"github.com/google/uuid"
)

type Music struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Title        string
	Mp3FilePath  string
	MidiFilePath string
	CoverArtPath string
	LikesCount   int        `gorm:"default:0"`
	IsPublic     bool       `gorm:"default:false"`
	UserID       *uuid.UUID `gorm:"type:uuid"`
	User         User
	MusicTypeID  uuid.UUID `gorm:"type:uuid"`
	MusicType    MusicType
	ModelTypeID  uuid.UUID `gorm:"type:uuid"`
	ModelType    ModelType
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
