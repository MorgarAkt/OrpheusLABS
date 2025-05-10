package models

import (
	"time"

	"github.com/google/uuid"
)

// UserLikesMusic, kullanıcıların hangi müzikleri beğendiğini gösteren ara tablo.
type UserLikesMusic struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	MusicID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Music     Music     `gorm:"foreignKey:MusicID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt time.Time
}
