package models

import (
	"time"
)

type Event struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:255;not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	Location        string    `gorm:"size:255" json:"location"`
	DateTime        time.Time `gorm:"not null" json:"date_time"`
	OrganizerID     uint      `gorm:"not null" json:"organizer_id"` // ID of the user who organized the event
	PointsAllocation int      `gorm:"default:0" json:"points_allocation"`
  ImageURL     string    `gorm:"size:512" json:"icon_url"`

	// Relationships
	Organizer User       `gorm:"foreignKey:OrganizerID" json:"organizer"`
	Attendees []Attendance `gorm:"foreignKey:EventID" json:"attendees"`
	Awards    []Award     `gorm:"many2many:event_awards" json:"awards"` // Many-to-many relationship with awards
}
