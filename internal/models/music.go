// Modify: OrpheusLABS/internal/models/music.go
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
	LikesCount   int  `gorm:"default:0"`
	ListensCount int  `gorm:"default:0"`
	IsPublic     bool `gorm:"default:false"`
	// UserID alanını pointer (*) olarak değiştirerek NULL olabilir hale getiriyoruz
	UserID      *uuid.UUID `gorm:"type:uuid"` // Foreign key to User (nullable)
	User        User       // Belongs To relationship (GORM bunu UserID üzerinden yönetir)
	MusicTypeID uuid.UUID  // Foreign key to MusicType
	MusicType   MusicType  // Belongs To relationship
	ModelTypeID uuid.UUID  // Foreign key to ModelType
	ModelType   ModelType  // Belongs To relationship
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
