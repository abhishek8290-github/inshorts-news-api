package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"news-api/internal/database"
	"news-api/internal/dto"
	"news-api/internal/models"
	"strings" // Added for string manipulation
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetEmbeddingsfromText(text string) ([]float64, error) {
	embedURL := "http://localhost:8001/embed"
	requestBody, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := http.Post(embedURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	return result.Embedding, nil
}

func GetLLMSummaryFromURL(articleURL string) (string, error) {
	// If the URL contains "youtube", return an empty string
	if strings.Contains(articleURL, "youtube.com") || strings.Contains(articleURL, "youtu.be") {
		return "", nil
	}

	summarizeURL := "http://localhost:8001/summarize"
	requestBody, err := json.Marshal(map[string]string{"url": articleURL})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body for summary: %w", err)
	}

	resp, err := http.Post(summarizeURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to call summarization service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("summarization service failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Summary string `json:"summary"`
		Title   string `json:"title"`
		Status  string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse summarization response: %w", err)
	}

	if result.Status != "success" {
		return "", fmt.Errorf("summarization service returned non-success status: %s", result.Status)
	}

	return result.Summary, nil
}

func AddNewsEntry(req *dto.AddNewsRequest) (*models.Article, error) {
	collection := database.GetCollection("news_articles")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check for duplicate URL
	filter := bson.M{"url": req.URL}
	var existingArticle models.Article
	err := collection.FindOne(ctx, filter).Decode(&existingArticle)
	if err == nil {
		return nil, fmt.Errorf("article with URL '%s' already exists", req.URL)
	}
	if err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to check for existing article: %w", err)
	}

	// Calculate vector embedding
	articleText := req.Title + " " + req.Description
	embedding, err := GetEmbeddingsfromText(articleText)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	// Calculate LLM Summary
	llmSummary, err := GetLLMSummaryFromURL(req.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM summary: %w", err)
	}

	article := models.Article{
		ID:              primitive.NewObjectID(),
		Title:           req.Title,
		Description:     req.Description,
		URL:             req.URL,
		PublicationDate: parseTime(req.PublicationDate),
		SourceName:      req.SourceName,
		Category:        req.Category,
		RelevanceScore:  req.RelevanceScore,
		Location: models.Location{
			Type:        "Point",
			Coordinates: []float64{req.Longitude, req.Latitude},
		},
		LLMSummary:      llmSummary,
		VectorEmbedding: embedding,
	}

	_, err = collection.InsertOne(ctx, article)
	if err != nil {
		fmt.Printf("Failed to insert article: %v\n", err)
		return nil, err
	}

	fmt.Println("Article added successfully!")
	return &article, nil
}

func AddNewsEntryList(req []*dto.AddNewsRequest) ([]models.Article, error) {
	collection := database.GetCollection("news_articles")
	ctx, cancel := context.WithTimeout(context.Background(), 60*10*time.Second)
	defer cancel()

	var articlesToInsert []interface{}
	var articlesAdded []models.Article

	for _, value := range req {
		// Check for duplicate URL
		filter := bson.M{"url": value.URL}
		var existingArticle models.Article
		err := collection.FindOne(ctx, filter).Decode(&existingArticle)
		if err == nil {
			fmt.Printf("Skipping duplicate article with URL: %s\n", value.URL)
			continue // Skip this article if it's a duplicate
		}
		if err != mongo.ErrNoDocuments {
			return nil, fmt.Errorf("failed to check for existing article '%s': %w", value.Title, err)
		}

		// Calculate vector embedding for each article
		articleText := value.Title + " " + value.Description
		embedding, err := GetEmbeddingsfromText(articleText)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedding for article '%s': %w", value.Title, err)
		}

		// Calculate LLM Summary for each article
		llmSummary, err := GetLLMSummaryFromURL(value.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to get LLM summary for article '%s': %w", value.Title, err)
		}

		article := models.Article{
			ID:              primitive.NewObjectID(),
			Title:           value.Title,
			Description:     value.Description,
			URL:             value.URL,
			PublicationDate: parseTime(value.PublicationDate),
			SourceName:      value.SourceName,
			Category:        value.Category,
			RelevanceScore:  value.RelevanceScore,
			Location: models.Location{
				Type:        "Point",
				Coordinates: []float64{value.Longitude, value.Latitude},
			},
			LLMSummary:      llmSummary,
			VectorEmbedding: embedding,
		}

		articlesToInsert = append(articlesToInsert, article)
		articlesAdded = append(articlesAdded, article)
	}

	if len(articlesToInsert) > 0 {
		_, err := collection.InsertMany(ctx, articlesToInsert)
		if err != nil {
			fmt.Printf("Failed to insert articles: %v\n", err)
			return nil, err
		}
		fmt.Println("Articles added successfully!")
	} else {
		fmt.Println("No new articles to add (all were duplicates or invalid).")
	}

	return articlesAdded, nil
}

func FindNews(filter primitive.M, page, pageSize int64) ([]dto.NewsArticleResponse, error) {
	collection := database.GetCollection("news_articles") // Assuming "news_articles" is your collection name
	ctx, cancel := context.WithTimeout(context.Background(), 60*10*time.Second)
	defer cancel()

	findOptions := options.Find()
	findOptions.SetSkip((page - 1) * pageSize)
	findOptions.SetLimit(pageSize)

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		fmt.Printf("Failed to find articles: %v\n", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var articles []models.Article
	if err = cursor.All(ctx, &articles); err != nil {
		fmt.Printf("Failed to decode articles: %v\n", err)
		return nil, err
	}

	var responseArticles []dto.NewsArticleResponse
	for _, article := range articles {
		responseArticles = append(responseArticles, dto.NewNewsArticleResponse(article))
	}

	return responseArticles, nil
}

func FindNewsByVectorEmbedding(embedding []float64, page, pageSize int64) ([]dto.NewsArticleResponse, error) {
	collection := database.GetCollection("news_articles")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{
			"$vectorSearch", bson.D{
				{"queryVector", embedding},
				{"path", "vector_embedding"},
				{"numCandidates", 100},
				{"limit", pageSize},
				{"index", "vector_index"},
			},
		}},
		{{
			"$addFields", bson.D{
				{"score", bson.D{{"$meta", "vectorSearchScore"}}},
			},
		}},
		{{
			"$skip", (page - 1) * pageSize,
		}},
		{{
			"$limit", pageSize,
		}},
		// Remove the restrictive $project stage or include ALL fields
		{{
			"$project", bson.D{
				{"_id", 1},
				{"title", 1},
				{"description", 1},
				{"url", 1},
				{"publication_date", 1},
				{"source_name", 1},
				{"category", 1},
				{"relevance_score", 1},
				{"location", 1},
				{"llm_summary", 1},
				{"vector_embedding", 1}, // Include if needed
				{"score", 1},
			},
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Printf("Failed to perform vector search: %v\n", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var articles []models.Article
	if err = cursor.All(ctx, &articles); err != nil {
		fmt.Printf("Failed to decode articles from vector search: %v\n", err)
		return nil, err
	}

	var responseArticles []dto.NewsArticleResponse
	for _, article := range articles {
		responseArticles = append(responseArticles, dto.NewNewsArticleResponse(article))
	}
	fmt.Println(responseArticles, "responseArticles")

	return responseArticles, nil
}

func parseTime(dateStr string) time.Time {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t
		}
	}
	fmt.Printf("Warning: Could not parse date string '%s' with known layouts. Returning zero time.\n", dateStr)
	return time.Time{} // Return zero time if parsing fails
}

func GetAllCategories() ([]string, error) {
	collection := database.GetCollection("news_articles")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use Distinct to get all unique categories
	categories, err := collection.Distinct(ctx, "category", bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct categories: %w", err)
	}

	var result []string
	for _, category := range categories {
		if catStr, ok := category.(string); ok {
			result = append(result, catStr)
		} else if catArr, ok := category.(primitive.A); ok {
			// Handle cases where category might be an array of strings
			for _, item := range catArr {
				if itemStr, ok := item.(string); ok {
					result = append(result, itemStr)
				}
			}
		}
	}
	return result, nil
}

func GetAllSourceNames() ([]string, error) {
	collection := database.GetCollection("news_articles")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use Distinct to get all unique source names
	sourceNames, err := collection.Distinct(ctx, "source_name", bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct source names: %w", err)
	}

	var result []string
	for _, sourceName := range sourceNames {
		if snStr, ok := sourceName.(string); ok {
			result = append(result, snStr)
		}
	}
	return result, nil
}
