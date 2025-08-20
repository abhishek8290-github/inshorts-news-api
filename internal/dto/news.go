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
	ID              string    `json:"id,omitempty"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	URL             string    `json:"url"`
	PublicationDate string    `json:"publication_date"`
	SourceName      string    `json:"source_name"`
	Category        []string  `json:"category"`
	RelevanceScore  float64   `json:"relevance_score"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	VectorEmbedding []float64 `json:"vector_embedding,omitempty"`
	LLMSummary      string    `json:"llm_summary,omitempty"`
}
