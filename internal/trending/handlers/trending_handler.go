package trending_handler

import (
	"context"
	"fmt"
	"news-api/internal/database"
	"news-api/internal/services"
	"news-api/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func CreateUserEvent(c *gin.Context) {
	var event struct {
		UserID    string  `json:"user_id"`
		ArticleID string  `json:"article_id"`
		EventType string  `json:"event_type"` // view, click, share
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	if err := c.ShouldBindJSON(&event); err != nil {
		utils.ErrorResponse(c, 400, "Invalid input: "+err.Error())
		return
	}

	// Create MongoDB document
	userEvent := bson.M{
		"user_id":    event.UserID,
		"article_id": event.ArticleID,
		"event_type": event.EventType,
		"timestamp":  time.Now(),
		"location": bson.M{
			"type":        "Point",
			"coordinates": []float64{event.Longitude, event.Latitude},
		},
	}

	// Insert into MongoDB
	db := database.GetDB()
	_, err := db.Collection("user_events").InsertOne(context.Background(), userEvent)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to create event: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "Event created successfully"})
}

func GetTrendingNews(c *gin.Context) {
	window := c.Query("window")

	var duration time.Duration
	switch window {
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "week":
		duration = 7 * 24 * time.Hour
	default:
		utils.ErrorResponse(c, 400, "Invalid window. Use 6h, 24h, or week")
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if err != nil || limit <= 0 {
		utils.ErrorResponse(c, 400, "Invalid limit parameter")
		return
	}

	cacheKey := fmt.Sprintf("trending:articles:%s", window)

	trendingArticles, err := services.GetTrendingFromCache(cacheKey, duration, limit)
	if err != nil {
		utils.ErrorResponse(c, 500, "Something Went Wrong: "+err.Error())
		return // Add return here to prevent further execution on error
	}

	utils.SuccessResponse(c, gin.H{
		"window":   window,
		"articles": trendingArticles,
	})
}
