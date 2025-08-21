package router

import (
	"github.com/gin-gonic/gin"
	"github.com/n1207n/cache-query-aggregator/internal/handler"
)

func SetupPostRoutes(apiGroup *gin.RouterGroup, postHandler *handler.PostHandler) {
	postRoutes := apiGroup.Group("/posts")
	{
		postRoutes.POST("", postHandler.CreatePost)
	}

	// It's common to nest resource routes, e.g., getting posts by a user.
	userSpecificRoutes := apiGroup.Group("/users/:id/")
	{
		userSpecificRoutes.GET("/posts", postHandler.ListPostsByUser)
	}
}
