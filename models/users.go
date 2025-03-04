package models

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

type UserStatus string

const (
	RoleAdmin  UserRole = "admin"
	RoleStaff  UserRole = "staff"
	RoleStudent UserRole = "student"

	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusBanned   UserStatus = "banned"
)

type User struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Email          string         `gorm:"size:255;unique;not null" json:"email"`
  Password       string         `gorm:"size:255" json:"-"` // Exclude password from JSON
	GoogleID       string         `gorm:"size:255" json:"-"` // Exclude Google ID from JSON
	GradYear       int            `gorm:"not null" json:"grad_year"`
	CurrentPoints  int            `gorm:"default:0" json:"current_points"`
  AwardsEarned  []Award         `gorm:"many2many:user_badges" json:"badges"` // Many-to-many relationship
	RegistrationDate time.Time     `gorm:"not null" json:"registration_date"`
	Status         UserStatus     `gorm:"type:enum('active','inactive','banned');default:'active'" json:"status"`
	Role           UserRole       `gorm:"type:enum('admin','staff','student');default:'student'" json:"role"`
	Department     string         `gorm:"size:255" json:"department"`
	Title          string         `gorm:"size:255" json:"title"`
	Biography      string         `gorm:"type:text" json:"biography"`

	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
