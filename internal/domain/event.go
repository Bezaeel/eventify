package domain

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Id          uuid.UUID  `json:"id" gorm:"primaryKey;column:id"`
	Name        string     `json:"name" gorm:"column:name"`
	Description string     `json:"description" gorm:"column:description"`
	Location    string     `json:"location" gorm:"column:location"`
	Date        time.Time  `json:"date" gorm:"column:date"`
	Organizer   string     `json:"organizer" gorm:"column:organizer"`
	Category    string     `json:"category" gorm:"column:category"`
	Tags        []string   `json:"tags" gorm:"column:tags;serializer:json"`
	Capacity    int        `json:"capacity" gorm:"column:capacity"`
	UpdatedAt   *time.Time `json:"updated_at" gorm:"column:updated_at"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at"`
	CreatedBy   uuid.UUID  `json:"created_by" gorm:"column:created_by"`
	Creator     User       `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:ID"`
}
