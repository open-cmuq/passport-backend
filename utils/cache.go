package utils

import (
	"sync"
	"time"
)

var (
	// Pending registrations for new users
	pendingRegistrations = make(map[string]PendingUser) // Key: email, Value: PendingUser
	muRegistrations      sync.RWMutex                   // Mutex for pending registrations

	// Pending password resets for existing users
	pendingResets = make(map[string]PendingReset) // Key: email, Value: PendingReset
	muResets      sync.RWMutex                    // Mutex for pending resets
)

// PendingUser represents a user waiting for OTP verification during registration
type PendingUser struct {
	Name         string    // User's name
	Email        string    // User's email
	PasswordHash string    // Hashed password
	OTP          string    // One-time password
	OTPExpiresAt time.Time // When the OTP expires
	LastOTPSent  time.Time // When the last OTP was sent
	Attempts     int       // Number of OTP attempts
	CreatedAt    time.Time // When the registration was initiated
}

// PendingReset represents a password reset request waiting for OTP verification
type PendingReset struct {
	Email        string    // User's email
	OTP          string    // One-time password
	OTPExpiresAt time.Time // When the OTP expires
	LastOTPSent  time.Time // When the last OTP was sent
	Attempts     int       // Number of OTP attempts
	CreatedAt    time.Time // When the reset was initiated
}

// Registration Functions
func AddPendingUser(email string, user PendingUser) {
	muRegistrations.Lock()
	defer muRegistrations.Unlock()
	pendingRegistrations[email] = user
}

func GetPendingUser(email string) (PendingUser, bool) {
	muRegistrations.RLock()
	defer muRegistrations.RUnlock()
	user, exists := pendingRegistrations[email]
	return user, exists
}

func DeletePendingUser(email string) {
	muRegistrations.Lock()
	defer muRegistrations.Unlock()
	delete(pendingRegistrations, email)
}

func IncrementPendingUserAttempts(email string) {
	muRegistrations.Lock()
	defer muRegistrations.Unlock()
	if user, exists := pendingRegistrations[email]; exists {
		user.Attempts++
		pendingRegistrations[email] = user
	}
}

// Reset Functions
func AddPendingReset(email string, reset PendingReset) {
	muResets.Lock()
	defer muResets.Unlock()
	pendingResets[email] = reset
}

func GetPendingReset(email string) (PendingReset, bool) {
	muResets.RLock()
	defer muResets.RUnlock()
	reset, exists := pendingResets[email]
	return reset, exists
}

func DeletePendingReset(email string) {
	muResets.Lock()
	defer muResets.Unlock()
	delete(pendingResets, email)
}

func IncrementPendingResetAttempts(email string) {
	muResets.Lock()
	defer muResets.Unlock()
	if reset, exists := pendingResets[email]; exists {
		reset.Attempts++
		pendingResets[email] = reset
	}
}

// Cleanup Functions
func CleanupExpiredRegistrations() {
	muRegistrations.Lock()
	defer muRegistrations.Unlock()
	now := time.Now()
	for email, user := range pendingRegistrations {
		if now.After(user.OTPExpiresAt) {
			delete(pendingRegistrations, email)
		}
	}
}

func CleanupExpiredResets() {
	muResets.Lock()
	defer muResets.Unlock()
	now := time.Now()
	for email, reset := range pendingResets {
		if now.After(reset.OTPExpiresAt) {
			delete(pendingResets, email)
		}
	}
}

func CleanupExpiredCache() {
	CleanupExpiredRegistrations()
	CleanupExpiredResets()
}
