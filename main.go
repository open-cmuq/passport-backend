package main

import (
	"log"
	"github.com/open-cmuq/passport-backend/database"
	"github.com/open-cmuq/passport-backend/models"
	"github.com/open-cmuq/passport-backend/routes"
  "github.com/open-cmuq/passport-backend/utils"
  "gorm.io/gorm"
  "time"
	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to database
	database.Connect()
  // Create ENUM types if they don't exist
	createEnumTypes(database.DB)
	// Auto-migrate models
  if err := database.DB.AutoMigrate(&models.User{}, &models.Event{}, &models.Attendance{}, &models.Award{}); err != nil {
    log.Fatalf("Failed to auto-migrate: %v", err)
  }

	// Initialize Gin
  gin.SetMode(gin.DebugMode)
	router := gin.Default()

	// Register routes
	routes.SetupRoutes(router)

  // Start the background cleanup task
	go func() {
		for {
			time.Sleep(5 * time.Minute) // Run every 5 minutes
			utils.CleanupExpiredRegistrations()
			log.Println("Cleaned up expired pending registrations")
		}
	}()
  
	// Start server
	log.Println("Server running on :8080")
	router.Run("0.0.0.0:8080")
}


func createEnumTypes(db *gorm.DB) {
	// Create user_status ENUM type
	if err := db.Exec(`DO $$ BEGIN
		CREATE TYPE user_status AS ENUM ('active', 'inactive', 'banned');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;`).Error; err != nil {
		log.Fatalf("Failed to create user_status ENUM type: %v", err)
	}

	// Create user_role ENUM type
	if err := db.Exec(`DO $$ BEGIN
		CREATE TYPE user_role AS ENUM ('admin', 'staff', 'student');
	EXCEPTION
		WHEN duplicate_object THEN null;
	END $$;`).Error; err != nil {
		log.Fatalf("Failed to create user_role ENUM type: %v", err)
	}
}
