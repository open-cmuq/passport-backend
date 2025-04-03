package controllers

import (
	"net/http"

	"github.com/open-cmuq/passport-backend/database"
	"github.com/open-cmuq/passport-backend/models"

	"github.com/gin-gonic/gin"
)

// Get all users
func GetUsers(c *gin.Context) {
	var users []models.User
	database.DB.Find(&users)
	c.JSON(http.StatusOK, users)
}

// Get single user by ID
func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateUser updates a user's information
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Define input struct with optional fields (pointers)
	var input struct {
		Name       *string `json:"name"`
		GradYear   *int    `json:"grad_year"`
		Title      *string `json:"title"`
		Biography  *string `json:"biography"`
		Department *string `json:"department"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update only provided fields
	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.GradYear != nil {
		user.GradYear = *input.GradYear
	}
	if input.Title != nil {
		user.Title = *input.Title
	}
	if input.Biography != nil {
		user.Biography = *input.Biography
	}
	if input.Department != nil {
		user.Department = *input.Department
	}

	// Save updates
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Delete user
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
