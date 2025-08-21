package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var Database *mongo.Database

func Connect() {
	os.Getenv("PORT")
	mongoURI := os.Getenv("MONGODB_URI")
	databaseName := os.Getenv("DATABASE_NAME")

	secondsToCancel := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), secondsToCancel)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	Client = client
	Database = client.Database(databaseName)

	fmt.Println("✅ Connected to MongoDB!")
}

func GetCollection(collectionName string) *mongo.Collection {
	return Database.Collection(collectionName)
}

func GetDB() *mongo.Database {
	return Database
}
