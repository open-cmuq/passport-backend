package models

import (
	"time"
  "golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusBanned   UserStatus = "banned"
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleStaff   UserRole = "staff"
	RoleStudent UserRole = "student"
)

type User struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Name             string         `gorm:"size:255;not null" json:"name"`
	Email            string         `gorm:"size:255;unique;not null" json:"email"`
	Password         string         `gorm:"size:255" json:"-"` // Exclude password from JSON
	GoogleID         string         `gorm:"size:255" json:"-"` // Exclude Google ID from JSON
  RefreshToken     string         `gorm:"size:512" json:"-"` // Refresh token
	RefreshTokenExp  time.Time      `json:"-"`                // Refresh token expiration time
	GradYear         int            `gorm:"not null" json:"grad_year"`
	CurrentPoints    int            `gorm:"default:0" json:"current_points"`
	AwardsEarned     []Award        `gorm:"many2many:user_badges" json:"badges"` // Many-to-many relationship
	RegistrationDate time.Time      `gorm:"not null" json:"registration_date"`
	Status           string         `gorm:"size:50;check:status IN ('active', 'inactive', 'banned');default:'active'" json:"status"`
	Role             string         `gorm:"size:50;check:role IN ('admin', 'staff', 'student');default:'student'" json:"role"`
	Department       string         `gorm:"size:255" json:"department"`
	Title            string         `gorm:"size:255" json:"title"`
	Biography        string         `gorm:"type:text" json:"biography"`
	OTP              string         `gorm:"size:6" json:"-"` // OTP for email verification
	OTPExpiresAt     time.Time      `json:"-"`              // OTP expiration time
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// HashPassword hashes the user's password
func (u *User) HashPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword checks if the provided password matches the hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
