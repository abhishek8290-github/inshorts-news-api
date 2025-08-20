package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"news-api/internal/dto"
	"news-api/internal/models"
	"news-api/internal/services"
	"news-api/internal/utils"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/api/option"
)

func GetEmbeddingsfromText(c *gin.Context) {
	// Parse input JSON { "text": "..." }
	var req struct {
		Text string `json:"text"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Call local FastAPI service
	embedURL := "http://localhost:8001/embed"
	// embedURL := "http://embed-service:8001/embed"

	payload, _ := json.Marshal(req)

	resp, err := http.Post(embedURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call embedding service"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": "Embedding service failed", "details": string(body)})
		return
	}

	// Parse embedding result { "embedding": [...] }
	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse embedding response"})
		return
	}

	// Return embedding to client
	c.JSON(http.StatusOK, gin.H{
		"embedding":  result.Embedding,
		"dimensions": len(result.Embedding),
	})
}

func CreateNewsEntry(c *gin.Context) {
	// 1. Parse and validate request
	var req dto.AddNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid input: "+err.Error())
		return
	}

	// 2. Call service layer
	article, err := services.AddNewsEntry(&req)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to add news entry: "+err.Error())
		return
	}

	// 3. Return success response
	utils.SuccessResponse(c, article)
}
func CreateNewsEntryList(c *gin.Context) {
	// 1. Parse and validate request
	var req []dto.AddNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid input: "+err.Error())
		return
	}

	// 2. Call service layer
	var newsPointers []*dto.AddNewsRequest
	for i := range req {
		newsPointers = append(newsPointers, &req[i])
	}
	article, err := services.AddNewsEntryList(newsPointers)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to add news entry: "+err.Error())
		return
	}

	// 3. Return success response
	utils.SuccessResponse(c, article)
}

func getPaginationParams(c *gin.Context) (page, pageSize int64, err error) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")

	page, err = strconv.ParseInt(pageStr, 10, 64)
	if err != nil || page <= 0 {
		utils.ErrorResponse(c, 400, "Invalid page number")
		return 0, 0, fmt.Errorf("invalid page number")
	}

	pageSize, err = strconv.ParseInt(pageSizeStr, 10, 64)
	if err != nil || pageSize <= 0 {
		utils.ErrorResponse(c, 400, "Invalid page size")
		return 0, 0, fmt.Errorf("invalid page size")
	}
	return page, pageSize, nil
}

func GetCategoryNews(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		utils.ErrorResponse(c, 400, "Category parameter is missing")
		return
	}

	page, pageSize, err := getPaginationParams(c)
	if err != nil {
		return // Error response already handled by getPaginationParams
	}

	articles, err := services.FindNews(primitive.M{"category": category}, page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve news by category: "+err.Error())
		return
	}

	utils.SuccessResponse(c, articles)
}

func GetNewsByScore(c *gin.Context) {
	scoreStr := c.Param("score")
	if scoreStr == "" {
		utils.ErrorResponse(c, 400, "Score parameter is missing")
		return
	}

	score, err := strconv.ParseFloat(scoreStr, 64)
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid score value")
		return
	}

	page, pageSize, err := getPaginationParams(c)
	if err != nil {
		return
	}

	filter := primitive.M{"relevance_score": primitive.M{"$gte": score}}
	articles, err := services.FindNews(filter, page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve news by score: "+err.Error())
		return
	}

	utils.SuccessResponse(c, articles)
}

func SearchNews(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.ErrorResponse(c, 400, "Search query parameter 'q' is missing")
		return
	}

	page, pageSize, err := getPaginationParams(c)
	if err != nil {
		return
	}

	filter := primitive.M{
		"$or": []primitive.M{
			{"title": primitive.Regex{Pattern: query, Options: "i"}},
			{"description": primitive.Regex{Pattern: query, Options: "i"}},
		},
	}
	articles, err := services.FindNews(filter, page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to search news: "+err.Error())
		return
	}

	utils.SuccessResponse(c, articles)
}

func GetNewsBySource(c *gin.Context) {
	source := c.Param("source")
	if source == "" {
		utils.ErrorResponse(c, 400, "Source parameter is missing")
		return
	}

	page, pageSize, err := getPaginationParams(c)
	if err != nil {
		return
	}

	filter := primitive.M{"source_name": source}
	articles, err := services.FindNews(filter, page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve news by source: "+err.Error())
		return
	}

	utils.SuccessResponse(c, articles)
}

func GetNewsNearby(c *gin.Context) {
	latitudeStr := c.Query("lat")
	longitudeStr := c.Query("lon")
	radiusStr := c.Query("radius")

	if latitudeStr == "" || longitudeStr == "" || radiusStr == "" {
		utils.ErrorResponse(c, 400, "Latitude, Longitude, or Radius parameter is missing")
		return
	}

	latitude, err := strconv.ParseFloat(latitudeStr, 64)
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid latitude value")
		return
	}
	longitude, err := strconv.ParseFloat(longitudeStr, 64)
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid longitude value")
		return
	}
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil || radius <= 0 {
		utils.ErrorResponse(c, 400, "Invalid radius value")
		return
	}

	page, pageSize, err := getPaginationParams(c)
	if err != nil {
		return
	}

	// MongoDB geospatial query for articles within a circle
	filter := primitive.M{
		"location": primitive.M{
			"$geoWithin": primitive.M{
				"$centerSphere": []interface{}{
					[]float64{longitude, latitude},
					radius / 6378.1, // Convert km to radians (Earth's radius in km)
				},
			},
		},
	}

	articles, err := services.FindNews(filter, page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve nearby news: "+err.Error())
		return
	}

	utils.SuccessResponse(c, articles)
}

type GeminiResponse struct {
	Intent   string   `json:"intent"`
	Entities []string `json:"entities"`
}

func SmartNewsRouter(c *gin.Context) {
	userQuery := c.Query("q")
	if userQuery == "" {
		utils.ErrorResponse(c, 400, "Query parameter 'q' is missing")
		return
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to create Gemini client: "+err.Error())
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")

	prompt := fmt.Sprintf(`
You are a query router for a news API.
Extract:
- intent: one of ["category","source","score","search","nearby"]
- entities: list of objects, each with:
   { "type": "category|source|keyword|location|score", "value": "..." }

Important:
- If the user mentions multiple keywords joined with "and" or "or", split them into separate entities.
- Example: "Bangladesh and India from News18" → 
  [
    { "type": "keyword", "value": "Bangladesh" },
    { "type": "keyword", "value": "India" },
    { "type": "source", "value": "News18" }
  ]

Return only valid JSON.
User query: "%s"`, userQuery)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get response from Gemini: "+err.Error())
		return
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		utils.ErrorResponse(c, 500, "Gemini returned no content")
		return
	}

	geminiText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	geminiText = strings.TrimSpace(geminiText)
	geminiText = strings.TrimPrefix(geminiText, "```json\n")
	geminiText = strings.TrimPrefix(geminiText, "```\n")
	geminiText = strings.TrimSuffix(geminiText, "\n```")

	var geminiResponse struct {
		Intent   string `json:"intent"`
		Entities []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"entities"`
	}

	err = json.Unmarshal([]byte(geminiText), &geminiResponse)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to parse Gemini response: "+err.Error())
		return
	}

	page, pageSize, err := getPaginationParams(c)
	if err != nil {
		return
	}

	var filter primitive.M
	var articles []models.Article

	switch geminiResponse.Intent {
	case "category":
		for _, e := range geminiResponse.Entities {
			if e.Type == "category" {
				filter = primitive.M{"category": e.Value}
				break
			}
		}

	case "source":
		for _, e := range geminiResponse.Entities {
			if e.Type == "source" {
				filter = primitive.M{"source_name": primitive.Regex{Pattern: "^" + e.Value + "$", Options: "i"}}
				break
			}
		}

	case "score":
		for _, e := range geminiResponse.Entities {
			if e.Type == "score" {
				score, parseErr := strconv.ParseFloat(e.Value, 64)
				if parseErr != nil {
					utils.ErrorResponse(c, 400, "Invalid score value")
					return
				}
				filter = primitive.M{"relevance_score": primitive.M{"$gte": score}}
				break
			}
		}

	case "search":
		var orClauses []primitive.M
		var andClauses []primitive.M

		for _, e := range geminiResponse.Entities {
			switch e.Type {
			case "keyword":
				orClauses = append(orClauses,
					primitive.M{"title": primitive.Regex{Pattern: e.Value, Options: "i"}},
					primitive.M{"description": primitive.Regex{Pattern: e.Value, Options: "i"}},
				)
			case "source":
				andClauses = append(andClauses, primitive.M{"source_name": e.Value})
			case "category":
				andClauses = append(andClauses, primitive.M{"category": e.Value})
			}
		}

		if len(orClauses) > 0 {
			andClauses = append(andClauses, primitive.M{"$or": orClauses})
		}
		if len(andClauses) > 0 {
			filter = primitive.M{"$and": andClauses}
		} else {
			// fallback to raw user query
			filter = primitive.M{
				"$or": []primitive.M{
					{"title": primitive.Regex{Pattern: userQuery, Options: "i"}},
					{"description": primitive.Regex{Pattern: userQuery, Options: "i"}},
				},
			}
		}

	case "nearby":
		latStr := c.Query("lat")
		lonStr := c.Query("lon")
		radiusStr := c.Query("radius")

		if latStr == "" || lonStr == "" || radiusStr == "" {
			utils.ErrorResponse(c, 400, "Latitude, Longitude, or Radius query parameters are missing for nearby intent")
			return
		}

		latitude, parseErr := strconv.ParseFloat(latStr, 64)
		if parseErr != nil {
			utils.ErrorResponse(c, 400, "Invalid latitude value")
			return
		}
		longitude, parseErr := strconv.ParseFloat(lonStr, 64)
		if parseErr != nil {
			utils.ErrorResponse(c, 400, "Invalid longitude value")
			return
		}
		radius, parseErr := strconv.ParseFloat(radiusStr, 64)
		if parseErr != nil || radius <= 0 {
			utils.ErrorResponse(c, 400, "Invalid radius value")
			return
		}

		filter = primitive.M{
			"location": primitive.M{
				"$geoWithin": primitive.M{
					"$centerSphere": []interface{}{
						[]float64{longitude, latitude},
						radius / 6378.1,
					},
				},
			},
		}

	default:
		utils.ErrorResponse(c, 400, "Unknown intent from Gemini: "+geminiResponse.Intent)
		return
	}

	articles, err = services.FindNews(filter, page, pageSize)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve news: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"articles": articles,
		"meta": gin.H{
			"intent":         geminiResponse.Intent,
			"entities":       geminiResponse.Entities,
			"original_query": userQuery,
		},
	})
}
