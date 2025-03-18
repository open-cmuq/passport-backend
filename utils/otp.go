package utils

import (
	"math/rand"
  "fmt"
)

// GenerateOTP generates a 6-digit OTP
func GenerateOTP() string {
	otp := rand.Intn(1000000) // Generates a number between 0 and 999999
	return fmt.Sprintf("%06d", otp) // Ensures the OTP is always 6 digits
}
