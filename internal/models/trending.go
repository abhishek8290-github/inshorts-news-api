package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserEvent represents a user interaction with a news article.
type UserEvent struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID    string             `bson:"user_id" json:"user_id"`                       // or IP hash for anonymous
	ArticleID string             `bson:"article_id" json:"article_id"`                 // ID of the news article
	EventType string             `bson:"event_type" json:"event_type"`                 // "view", "click", "share"
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`                   // When the event occurred
	Location  Location           `bson:"location" json:"location"`                     // GeoJSON Point for user location
	Metadata  map[string]string  `bson:"metadata,omitempty" json:"metadata,omitempty"` // Additional event data
}

// TrendingArticle represents an article's trending score within a cache entry.
type TrendingArticle struct {
	ArticleID        string  `bson:"article_id" json:"article_id"`
	TrendingScore    float64 `bson:"trending_score" json:"trending_score"`
	InteractionCount int     `bson:"interaction_count" json:"interaction_count"`
	RecentActivity   int     `bson:"recent_activity" json:"recent_activity"`
	Title            string  `bson:"title,omitempty" json:"title,omitempty"` // Enriched data
	Description      string  `bson:"description,omitempty" json:"description,omitempty"`
	URL              string  `bson:"url,omitempty" json:"url,omitempty"`
	SourceName       string  `bson:"source_name,omitempty" json:"source_name,omitempty"`
	Category         string  `bson:"category,omitempty" json:"category,omitempty"`
}

// TrendingCache represents a cached set of trending articles for a specific geo-cluster.
type TrendingCache struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	GeoCluster   string             `bson:"geo_cluster" json:"geo_cluster"`     // e.g., "37.42_-122.08_10km"
	Articles     []TrendingArticle  `bson:"articles" json:"articles"`           // List of trending articles
	CalculatedAt time.Time          `bson:"calculated_at" json:"calculated_at"` // When the cache was calculated
	ExpiresAt    time.Time          `bson:"expires_at" json:"expires_at"`       // When the cache expires (TTL)
	RadiusKm     float64            `bson:"radius_km" json:"radius_km"`         // Radius used for calculation
}
