package controllers

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/open-cmuq/passport-backend/database"
	"github.com/open-cmuq/passport-backend/models"
	"github.com/open-cmuq/passport-backend/utils"
)

// VerifyOTP handles OTP verification
func VerifyOTP(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Retrieve the pending user
	pendingUser, exists := utils.GetPendingUser(input.Email)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "No pending registration found for this email"})
		return
	}

	// Check if the OTP matches and is not expired
	if pendingUser.OTP != input.OTP || time.Now().After(pendingUser.OTPExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Start a transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Create user within the transaction
	user := models.User{
		Name:     pendingUser.Name,
		Email:    pendingUser.Email,
		Password: pendingUser.PasswordHash,
	}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate access token
	accessToken, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	// Generate refresh token
	refreshToken, refreshTokenExp, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Save the refresh token to the database
	user.RefreshToken = refreshToken
	user.RefreshTokenExp = refreshTokenExp
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refresh token"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	// Remove the user from the pending registrations cache
	utils.DeletePendingUser(input.Email)

	// Return the tokens
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Login handles password-based login
func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !user.CheckPassword(input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate access token
	accessToken, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	var refreshToken string
	var refreshTokenExp time.Time

	// Check if existing refresh token is still valid
	if user.RefreshToken != "" && user.RefreshTokenExp.After(time.Now()) {
		refreshToken = user.RefreshToken
		refreshTokenExp = user.RefreshTokenExp
	} else {
		// Generate new refresh token if none exists or it's expired
		refreshToken, refreshTokenExp, err = utils.GenerateRefreshToken(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
			return
		}

		// Save the new refresh token to the database
		user.RefreshToken = refreshToken
		user.RefreshTokenExp = refreshTokenExp
		if err := database.DB.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refresh token"})
			return
		}
	}

	// Return the tokens
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Register handles user registration
func Register(c *gin.Context) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate email domain
	validEmail := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@(andrew\.cmu\.edu|qatar\.cmu\.edu|cmu\.edu)$`)
	if !validEmail.MatchString(input.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email must be from @andrew.cmu.edu, @qatar.cmu.edu, or @cmu.edu"})
		return
	}

	// Check if a user with the same email already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash the password
	var user models.User
	if err := user.HashPassword(input.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate and send OTP (only in production)
	if gin.Mode() == gin.ReleaseMode {
		otp := utils.GenerateOTP()
		otpExpiresAt := time.Now().Add(10 * time.Minute) // OTP expires in 10 minutes

		// Store the user's data in the pending registrations cache
		pendingUser := utils.PendingUser{
			Name:         input.Name,
			Email:        input.Email,
			PasswordHash: user.Password,
			OTP:          otp,
			OTPExpiresAt: otpExpiresAt,
		}
		utils.AddPendingUser(input.Email, pendingUser)

		// Send OTP via email (mock implementation for now)
		// In production, integrate with an email service like SendGrid or AWS SES
		go func(email, otp string) {
			// Mock email sending
			println("Sending OTP to", email, ":", otp)
		}(input.Email, otp)

		c.JSON(http.StatusOK, gin.H{"message": "OTP sent for verification"})
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// In development mode, directly create the user
	user.Name = input.Name
	user.Email = input.Email
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate access token
	accessToken, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	// Generate refresh token
	refreshToken, refreshTokenExp, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Save the refresh token to the database
	user.RefreshToken = refreshToken
	user.RefreshTokenExp = refreshTokenExp
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refresh token"})
		return
	}

	tx.Commit()

	// Return the tokens
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// RefreshToken handles access token refresh
func RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the refresh token
	claims, err := utils.ValidateToken(input.RefreshToken, "refresh") // Only allow refresh tokens
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Find the user
	var user models.User
	if err := database.DB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Check if the refresh token matches and is not expired
	if user.RefreshToken != input.RefreshToken || time.Now().After(user.RefreshTokenExp) {
		fmt.Print(time.Now())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	// Generate a new access token
	accessToken, err := utils.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	// Return the new access token
	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}
