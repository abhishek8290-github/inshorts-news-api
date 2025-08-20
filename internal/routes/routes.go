package routes

import (
	"news-api/internal/handlers"
	"news-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Apply global middleware
	r.Use(middleware.Logger())

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/hello", handlers.GetHello)
		v1.POST("/hello", handlers.PostHello)
	}

	newsRouterV1 := v1.Group("/news")
	{
		newsRouterV1.POST("/", handlers.CreateNewsEntry)
		newsRouterV1.POST("/list", handlers.CreateNewsEntryList)

		newsRouterV1.GET("/category/:category", handlers.GetCategoryNews)
		newsRouterV1.GET("/score/:score", handlers.GetNewsByScore)
		newsRouterV1.GET("/source/:source", handlers.GetNewsBySource)
		newsRouterV1.GET("/search", handlers.SmartNewsRouter)
		newsRouterV1.GET("/nearby", handlers.GetNewsNearby)

		// newsRouterV1.GET("/embed", handlers.GetEmbeddingsHandler)
	}

	// Health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
}
