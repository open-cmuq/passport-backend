package utils

import (
	"sync"
	"time"
)

var (
	pendingRegistrations = make(map[string]PendingUser) // Key: email, Value: PendingUser
	mu                   sync.RWMutex                  // Mutex to handle concurrent access
)

// PendingUser represents a user waiting for OTP verification
type PendingUser struct {
	Name         string
	Email        string
	PasswordHash string
	OTP          string
	OTPExpiresAt time.Time
}

// AddPendingUser adds a user to the pending registrations cache
func AddPendingUser(email string, user PendingUser) {
	mu.Lock()
	defer mu.Unlock()
	pendingRegistrations[email] = user
}

// GetPendingUser retrieves a user from the pending registrations cache
func GetPendingUser(email string) (PendingUser, bool) {
	mu.RLock()
	defer mu.RUnlock()
	user, exists := pendingRegistrations[email]
	return user, exists
}

// DeletePendingUser removes a user from the pending registrations cache
func DeletePendingUser(email string) {
	mu.Lock()
	defer mu.Unlock()
	delete(pendingRegistrations, email)
}

// CleanupExpiredRegistrations removes expired pending registrations from the cache
func CleanupExpiredRegistrations() {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	for email, user := range pendingRegistrations {
		if now.After(user.OTPExpiresAt) {
			delete(pendingRegistrations, email)
		}
	}
}
