package services

import (
	"context"
	"encoding/json"
	"fmt"
	database "news-api/internal/database"
	"news-api/internal/models"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// enrichWithNewsData fetches full article details for trending articles.
func enrichWithNewsData(trendingResults []bson.M) []models.TrendingArticle {
	db := database.GetDB()
	var enrichedArticles []models.TrendingArticle

	if len(trendingResults) == 0 {
		return enrichedArticles
	}

	// Extract article IDs
	var articleIDs []primitive.ObjectID
	for _, res := range trendingResults {
		articleIDStr, ok := res["_id"].(string)
		if !ok {
			fmt.Printf("Warning: Article ID is not a string: %v\n", res["_id"])
			continue
		}
		objID, err := primitive.ObjectIDFromHex(articleIDStr)
		if err != nil {
			fmt.Printf("Warning: Could not convert article ID '%s' to ObjectID: %v\n", articleIDStr, err)
			continue
		}
		articleIDs = append(articleIDs, objID)
	}

	if len(articleIDs) == 0 {
		return enrichedArticles
	}

	// Fetch articles from news_articles collection
	cursor, err := db.Collection("news_articles").Find(context.Background(), bson.M{"_id": bson.M{"$in": articleIDs}})
	if err != nil {
		fmt.Printf("Failed to fetch news articles for enrichment: %v\n", err)
		return enrichedArticles
	}
	defer cursor.Close(context.Background())

	articleMap := make(map[primitive.ObjectID]models.Article)
	for cursor.Next(context.Background()) {
		var article models.Article
		if err := cursor.Decode(&article); err != nil {
			fmt.Printf("Failed to decode news article during enrichment: %v\n", err)
			continue
		}
		articleMap[article.ID] = article
	}

	// Combine trending scores with article details
	for _, res := range trendingResults {
		if articleIDStr, ok := res["_id"].(string); ok {
			if objID, err := primitive.ObjectIDFromHex(articleIDStr); err == nil {
				if article, found := articleMap[objID]; found {
					trendingArticle := models.TrendingArticle{
						ArticleID:        articleIDStr,
						TrendingScore:    toFloat64(res["trending_score"]),
						InteractionCount: toInt(res["interaction_count"]),
						RecentActivity:   toInt(res["recent_activity"]),
						Title:            article.Title,
						Description:      article.Description,
						URL:              article.URL,
						SourceName:       article.SourceName,
						Category:         article.Category[0], // Assuming single category for simplicity
					}
					enrichedArticles = append(enrichedArticles, trendingArticle)
				}
			}
		}
	}
	return enrichedArticles
}

// CalculateTrendingScoresGlobal calculates trending scores globally within a time window.
func CalculateTrendingScoresGlobal(duration time.Duration, limit int) []models.TrendingArticle {
	db := database.GetDB()
	pipeline := []bson.M{
		{"$match": bson.M{
			"timestamp": bson.M{"$gte": time.Now().Add(-duration)},
		}},
		{
			"$group": bson.M{
				"_id":          "$article_id",
				"total_events": bson.M{"$sum": 1},
				"views": bson.M{
					"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []string{"$event_type", "view"}}, 1, 0}},
				},
				"clicks": bson.M{
					"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []string{"$event_type", "click"}}, 1, 0}},
				},
				"shares": bson.M{
					"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []string{"$event_type", "share"}}, 1, 0}},
				},
			},
		},
		{
			"$addFields": bson.M{
				"trending_score": bson.M{
					"$add": []interface{}{
						bson.M{"$multiply": []interface{}{"$views", 1}},
						bson.M{"$multiply": []interface{}{"$clicks", 2}},
						bson.M{"$multiply": []interface{}{"$shares", 3}},
					},
				},
			},
		},
		{"$sort": bson.M{"trending_score": -1}},
		{"$limit": limit},
	}

	cursor, err := db.Collection("user_events").Aggregate(context.Background(), pipeline)
	if err != nil {
		fmt.Printf("Failed to aggregate trending scores: %v\n", err)
		return []models.TrendingArticle{}
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		fmt.Printf("Failed to decode trending aggregation results: %v\n", err)
		return []models.TrendingArticle{}
	}

	fmt.Print(&results)

	return enrichWithNewsData(results)
}

func toFloat64(v interface{}) float64 {
	switch t := v.(type) {
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case float32:
		return float64(t)
	case float64:
		return t
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	switch t := v.(type) {
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}

func GetTrendingFromCache(cacheKey string, duration time.Duration, limit int) ([]models.TrendingArticle, error) {
	ctx := context.Background()
	val, err := database.Rdb.Get(ctx, cacheKey).Result()

	if err == redis.Nil {
		articles := CalculateTrendingScoresGlobal(duration, limit)

		articlesJSON, err := json.Marshal(articles)
		if err != nil {
			fmt.Printf("Failed to marshal trending articles for caching: %v\n", err)
			return articles, nil
		}

		// A cron Should work fine with this TODO
		err = database.Rdb.Set(ctx, cacheKey, articlesJSON, 1*time.Hour).Err()
		if err != nil {
			fmt.Printf("Failed to cache trending articles in Redis: %v\n", err)
		}

		return articles, nil
	} else if err != nil {
		return nil, err
	}

	var results []models.TrendingArticle
	err = json.Unmarshal([]byte(val), &results)
	return results, err
}

func ScheduleGlobalTrendingCalculations() {
	ctx := context.Background()

	fmt.Println("this is happening !!")

	trendingConfigs := []struct {
		Key      string
		Duration time.Duration
		Limit    int
	}{
		{"trending:articles:6h", 6 * time.Hour, 10},
		{"trending:articles:24h", 24 * time.Hour, 10},
		{"trending:articles:week", 7 * 24 * time.Hour, 10},
	}

	for _, config := range trendingConfigs {
		articles := CalculateTrendingScoresGlobal(config.Duration, config.Limit)
		articlesJSON, err := json.Marshal(articles)
		if err != nil {
			fmt.Printf("Failed to marshal trending articles for key %s: %v\n", config.Key, err)
			continue
		}

		err = database.Rdb.Set(ctx, config.Key, articlesJSON, 1*time.Hour).Err()
		if err != nil {
			fmt.Printf("Failed to cache trending articles for key %s in Redis: %v\n", config.Key, err)
		} else {
			fmt.Printf("Successfully cached trending articles for key: %s\n", config.Key)
		}
	}
}
