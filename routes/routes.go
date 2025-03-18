package routes

import (
	"github.com/open-cmuq/passport-backend/controllers"
  "github.com/open-cmuq/passport-backend/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
  router.POST("/login", controllers.Login)
	router.POST("/register", controllers.Register)
  router.POST("/verify-otp", controllers.VerifyOTP)

  userRoutes := router.Group("/users")
	userRoutes.Use(middleware.AuthMiddleware())
	{
		userRoutes.GET("/", controllers.GetUsers)
		//userRoutes.POST("/", controllers.CreateUser)
		userRoutes.GET("/:id", controllers.GetUserByID)
		userRoutes.PUT("/:id",middleware.OwnershipMiddleware(), controllers.UpdateUser)
		userRoutes.DELETE("/:id", middleware.AdminOnlyMiddleware(), controllers.DeleteUser)
	}
}
