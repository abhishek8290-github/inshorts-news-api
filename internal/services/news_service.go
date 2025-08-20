package services

import (
	"context"
	"fmt"
	"news-api/internal/database"
	"news-api/internal/dto"
	"news-api/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddNewsEntry(req *dto.AddNewsRequest) (*models.Article, error) {
	collection := database.GetCollection("news_articles") // Assuming "news_articles" is your collection name

	// Convert DTO to Model if necessary, or directly use req if it's already models.Article
	article := models.Article{
		ID:              primitive.NewObjectID(),
		Title:           req.Title,
		Description:     req.Description,
		URL:             req.URL,
		PublicationDate: parseTime(req.PublicationDate),
		SourceName:      req.SourceName,
		Category:        req.Category,
		RelevanceScore:  req.RelevanceScore,
		Location: struct {
			Type        string    `bson:"type" json:"type"`
			Coordinates []float64 `bson:"coordinates" json:"coordinates"`
		}{
			Type:        "Point",
			Coordinates: []float64{req.Longitude, req.Latitude}, // GeoJSON expects [lon, lat]
		},
		LLMSummary:      req.LLMSummary,
		VectorEmbedding: req.VectorEmbedding,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, article)
	if err != nil {
		fmt.Printf("Failed to insert article: %v\n", err)
		return nil, err
	}

	fmt.Println("Article added successfully!")
	return &article, nil
}

func AddNewsEntryList(req []*dto.AddNewsRequest) ([]models.Article, error) {
	collection := database.GetCollection("news_articles") // Assuming "news_articles" is your collection name
	ctx, cancel := context.WithTimeout(context.Background(), 60*10*time.Second)
	defer cancel()

	_articlesAdded := []models.Article{}
	for _, value := range req {
		article := models.Article{
			ID:              primitive.NewObjectID(), // Generate a new ObjectID
			Title:           value.Title,
			Description:     value.Description,
			URL:             value.URL,
			PublicationDate: parseTime(value.PublicationDate), // Assuming a helper function or direct parsing
			SourceName:      value.SourceName,
			Category:        value.Category,
			RelevanceScore:  value.RelevanceScore,
			Location: struct {
				Type        string    `bson:"type" json:"type"`
				Coordinates []float64 `bson:"coordinates" json:"coordinates"`
			}{
				Type:        "Point",
				Coordinates: []float64{value.Longitude, value.Latitude}, // GeoJSON expects [lon, lat]
			},
			LLMSummary:      value.LLMSummary,
			VectorEmbedding: value.VectorEmbedding,
		}

		_articlesAdded = append(_articlesAdded, article)
	}

	// Convert []models.Article to []interface{} for InsertMany
	var articlesToInsert []interface{}
	for _, article := range _articlesAdded {
		articlesToInsert = append(articlesToInsert, article)
	}

	_, err := collection.InsertMany(ctx, articlesToInsert)
	if err != nil {
		fmt.Printf("Failed to insert articles: %v\n", err)
		return nil, err
	}

	fmt.Println("Articles added successfully!")
	return _articlesAdded, nil
}

func FindNews(filter primitive.M, page, pageSize int64) ([]models.Article, error) {
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

	return articles, nil
}

// parseTime parses a string into a time.Time object.
// It attempts to parse common date formats.
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
