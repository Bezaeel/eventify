package domain

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Id        uuid.UUID `json:"id" gorm:"primaryKey;column:id"`
	Name      string    `json:"name" gorm:"column:name"`
	Location  string    `json:"location" gorm:"column:location"`
	Date      time.Time `json:"date" gorm:"column:date"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	CreatedBy uuid.UUID `json:"created_by" gorm:"column:created_by"`
	Creator   User      `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:ID"`
}
