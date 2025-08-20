package main

import (
	"log"
	"news-api/internal/database"
	"news-api/internal/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}
	// Create Gin router
	r := gin.Default()

	database.Connect()

	// Setup all routes
	routes.SetupRoutes(r)

	r.GET("/test-db", func(c *gin.Context) {
		if database.Client == nil {
			c.JSON(500, gin.H{"error": "Database not connected"})
			return
		}

		c.JSON(200, gin.H{
			"message":  "Database connected successfully!",
			"database": "news_db",
		})
	})

	// Start server
	r.Run(":8080")
}
