package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/yourprojectname/internal/handler"
)

// SetupUserRoutes configures the routes for user-related actions within a given router group.
func SetupUserRoutes(apiGroup *gin.RouterGroup, userHandler *handler.UserHandler) {
	userRoutes := apiGroup.Group("/users")
	{
		userRoutes.POST("", userHandler.CreateUser)
		userRoutes.GET("/:id", userHandler.GetUserByID)
		// Future routes like ListUsers, UpdateUser, DeleteUser would go here.
		// userRoutes.GET("", userHandler.ListUsers)
		// userRoutes.PUT("/:id", userHandler.UpdateUser)
		// userRoutes.DELETE("/:id", userHandler.DeleteUser)
	}
}
