package dto

import "news-api/internal/models"

type SearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit,omitempty"`
}

type CategoryRequest struct {
	Category string `json:"category" binding:"required"`
	Limit    int    `json:"limit,omitempty"`
}

type SourceRequest struct {
	Source string `json:"source" binding:"required"`
	Limit  int    `json:"limit,omitempty"`
}

type NearbyRequest struct {
	Lat    float64 `json:"lat" binding:"required"`
	Lon    float64 `json:"lon" binding:"required"`
	Radius float64 `json:"radius,omitempty"` // km
	Limit  int     `json:"limit,omitempty"`
}

type NewsResponse struct {
	Articles []models.Article `json:"articles"`
	Count    int              `json:"count"`
	Query    string           `json:"query,omitempty"`
}

// type AddNewsRequest = models.Article

type AddNewsRequest struct {
	Title           string   `json:"title" binding:"required"`
	Description     string   `json:"description" binding:"required"`
	URL             string   `json:"url" binding:"required"`
	PublicationDate string   `json:"publication_date" binding:"required"`
	SourceName      string   `json:"source_name" binding:"required"`
	Category        []string `json:"category" binding:"required"`
	RelevanceScore  float64  `json:"relevance_score" binding:"required"`
	Latitude        float64  `json:"latitude" binding:"required"`
	Longitude       float64  `json:"longitude" binding:"required"`
	LLMSummary      string   `json:"llm_summary,omitempty"`
}
