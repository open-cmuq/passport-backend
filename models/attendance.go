package models

import (
	"time"
)

type Attendance struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null" json:"user_id"`
	EventID     uint      `gorm:"not null" json:"event_id"`
	ScannedTime time.Time `gorm:"not null" json:"scanned_time"`

	// Relationships
	User  User  `gorm:"foreignKey:UserID" json:"user"`
	Event Event `gorm:"foreignKey:EventID" json:"event"`
}
