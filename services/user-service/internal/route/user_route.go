package route

import (
	"auron/user-service/internal/handler"
	"auron/user-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(router *gin.Engine, userHandler *handler.UserHandler) {
	api := router.Group("/")

	api.POST("/register", userHandler.Register)
	api.POST("/login", userHandler.Login)
	api.POST("/refresh", userHandler.RefreshToken)

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/logout", userHandler.Logout)
		protected.GET("/me", userHandler.GetProfile)
		protected.PUT("/me", userHandler.UpdateProfile)
		protected.POST("/me/addresses", userHandler.AddAddress)
		protected.GET("/me/addresses", userHandler.GetAddresses)
		protected.PUT("/me/addresses/:id", userHandler.UpdateAddress)
		protected.DELETE("/me/addresses/:id", userHandler.DeleteAddress)
	}
}
