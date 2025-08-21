package main

import (
	"log"
	"news-api/internal/database"
	"news-api/internal/routes"
	"news-api/internal/services" // Import services package

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3" // Import cron package
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}
	// Create Gin router
	r := gin.Default()

	database.Connect()
	database.InitRedis() // Initialize Redis client

	// Initialize and start cron scheduler
	c := cron.New()
	// Schedule the global trending calculations to run every hour
	c.AddFunc("@hourly", func() {
		log.Println("Running scheduled global trending calculations...")
		services.ScheduleGlobalTrendingCalculations()
	})
	c.Start()

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
