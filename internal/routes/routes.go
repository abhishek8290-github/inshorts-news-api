package routes

import (
	newsHandlers "news-api/internal/handlers"
	"news-api/internal/middleware"
	trendingHandlers "news-api/internal/trending/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Apply global middleware
	r.Use(middleware.Logger())

	// API v1 group
	v1 := r.Group("/api/v1")
	{
	}

	newsRouterV1 := v1.Group("/news")
	{
		newsRouterV1.POST("/", newsHandlers.CreateNewsEntry)
		newsRouterV1.POST("/list", newsHandlers.CreateNewsEntryList)

		newsRouterV1.GET("/category/:category", newsHandlers.GetCategoryNews)
		newsRouterV1.GET("/score/:score", newsHandlers.GetNewsByScore)
		newsRouterV1.GET("/source/:source", newsHandlers.GetNewsBySource)
		newsRouterV1.GET("/search", newsHandlers.SmartNewsRouter)
		newsRouterV1.GET("/nearby", newsHandlers.GetNewsNearby)
		newsRouterV1.GET("/categories", newsHandlers.GetCategories)
		newsRouterV1.GET("/sources", newsHandlers.GetSourceNames)

		newsRouterV1.POST("/events", trendingHandlers.CreateUserEvent)

		newsRouterV1.GET("/trending", trendingHandlers.GetTrendingNews)

	}

	// Health check
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
}
