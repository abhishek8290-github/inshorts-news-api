package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Article struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title           string             `bson:"title" json:"title"`
	Description     string             `bson:"description" json:"description"`
	URL             string             `bson:"url" json:"url"`
	PublicationDate time.Time          `bson:"publication_date" json:"publication_date"`
	SourceName      string             `bson:"source_name" json:"source_name"`
	Category        []string           `bson:"category" json:"category"`
	RelevanceScore  float64            `bson:"relevance_score" json:"relevance_score"`

	// Geospatial location in GeoJSON format
	Location Location `bson:"location" json:"location"`

	// Optional enrichment fields
	LLMSummary      string    `bson:"llm_summary" json:"llm_summary"`
	VectorEmbedding []float64 `bson:"vector_embedding,omitempty" json:"vector_embedding,omitempty"`
}

// Location represents the GeoJSON location structure
type Location struct {
	Type        string    `bson:"type" json:"type"`               // always "Point"
	Coordinates []float64 `bson:"coordinates" json:"coordinates"` // [lon, lat]
}
