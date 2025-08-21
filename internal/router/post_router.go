package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/yourprojectname/internal/handler"
)

func SetupPostRoutes(apiGroup *gin.RouterGroup, postHandler *handler.PostHandler) {
	postRoutes := apiGroup.Group("/posts")
	{
		postRoutes.POST("", postHandler.CreatePost)
	}

	// It's common to nest resource routes, e.g., getting posts by a user.
	userSpecificRoutes := apiGroup.Group("/users/:userId")
	{
		userSpecificRoutes.GET("/posts", postHandler.ListPostsByUser)
	}
}
