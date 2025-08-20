package dto

import (
	"news-api/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HelloGetResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type HelloPostResponse struct {
	Message   string `json:"message"`
	FullName  string `json:"full_name"`
	Timestamp string `json:"timestamp"`
}

type NewsArticleResponse struct {
	ID              primitive.ObjectID `json:"id"`
	Title           string             `json:"title"`
	Description     string             `json:"description"`
	URL             string             `json:"url"`
	PublicationDate time.Time          `json:"publication_date"`
	SourceName      string             `json:"source_name"`
	Category        []string           `json:"category"`
	RelevanceScore  float64            `json:"relevance_score"`
	Location        models.Location    `json:"location"`
	LLMSummary      string             `json:"llm_summary"`
}

func NewNewsArticleResponse(article models.Article) NewsArticleResponse {
	llmSummary := article.LLMSummary
	if llmSummary == "" {
		llmSummary = "" // Ensure it's an empty string if not present
	}

	return NewsArticleResponse{
		ID:              article.ID,
		Title:           article.Title,
		Description:     article.Description,
		URL:             article.URL,
		PublicationDate: article.PublicationDate,
		SourceName:      article.SourceName,
		Category:        article.Category,
		RelevanceScore:  article.RelevanceScore,
		Location:        article.Location, // This will now work because models.Location is used
		LLMSummary:      llmSummary,
	}
}
