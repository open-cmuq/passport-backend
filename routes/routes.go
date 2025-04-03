package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/open-cmuq/passport-backend/controllers"
	"github.com/open-cmuq/passport-backend/middleware"
)

func SetupRoutes(router *gin.Engine) {
	router.POST("/login", controllers.Login)
	router.POST("/register", controllers.Register)
	router.POST("/verify-otp", controllers.VerifyOTP)
	router.POST("/refresh-token", controllers.RefreshToken)

	userRoutes := router.Group("/users")
	userRoutes.Use(middleware.AuthMiddleware())
	{
		userRoutes.GET("/", controllers.GetUsers)
		// userRoutes.POST("/", controllers.CreateUser)
		userRoutes.GET("/:id", controllers.GetUserByID)
		userRoutes.PATCH("/:id", middleware.OwnershipMiddleware(), controllers.UpdateUser)
		userRoutes.DELETE("/:id", middleware.AdminOnlyMiddleware(), controllers.DeleteUser)
	}

	// Event routes
	// TODO enable the admin before production
	eventRoutes := router.Group("/events")
	eventRoutes.Use(middleware.AuthMiddleware())
	{
		eventRoutes.GET("/", controllers.GetEvents)
		eventRoutes.POST("/", controllers.CreateEvent)
		eventRoutes.GET("/:eventId", controllers.GetEvent)
		eventRoutes.PATCH("/:eventId", controllers.UpdateEvent)
		eventRoutes.DELETE("/:eventId", controllers.DeleteEvent)
		eventRoutes.GET("/:eventId/attendees", controllers.GetEventAttendees)
		eventRoutes.POST("/:eventId/attendances", controllers.AddAttendances)
		eventRoutes.DELETE("/:eventId/attendances", controllers.DeleteAttendances)
	}
}
