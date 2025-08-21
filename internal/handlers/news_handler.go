package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"news-api/internal/dto"
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

func GetCategories(c *gin.Context) {
	categories, err := services.GetAllCategories()
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve categories: "+err.Error())
		return
	}
	utils.SuccessResponse(c, categories)
}

func GetSourceNames(c *gin.Context) {
	sourceNames, err := services.GetAllSourceNames()
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve source names: "+err.Error())
		return
	}
	utils.SuccessResponse(c, sourceNames)
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

	emeddings, err := services.GetEmbeddingsfromText(userQuery)
	fmt.Println("Hello there ")
	fmt.Println(emeddings)

	model := client.GenerativeModel("gemini-2.5-flash")

	prompt := fmt.Sprintf(`
You are a query router for a news API.
Extract:
- intent: one of ["category","source","score","search","nearby","vector_search"]
- entities: list of objects, each with:
   { "type": "category|source|keyword|score", "value": "..." }

Important:
- If the user mentions multiple keywords joined with "and" or "or", split them into separate entities.
- Remember the User can do the spelling mistakes these are the source _name [
  'Freepressjournal',      'X',                  'Tribuneindia',
  'News18',                'Hindustan Times',    'Sports Tiger',
  'Moneycontrol',          'PTI',                'Zhejiang University',
  'dw.com',                'Cricket.com',        'ANI',
  'ANI News',              'Hindustantimes',     'The Print',
  'Chandigarhcitynews',    'Free Press Journal', 'Ascendants',
  'ET Now',                'The Indian Express', 'LatestLY',
  'Youtube',               'Times Now',          'Instagram',
  'Sports Info',           'RT International',   'The Chenab Times',
  'Reuters',               'NDTV',               'ESPNcricinfo',
  'Cricfit',               'NewsBytes',          'Logistics Outlook',
  'The South First',       'Latestly',           'MoneyControl ',
  'Briefly',               'RT',                 'ABP News',
  'X (Formerly Twitter)',  'YouTube ',           'ABP Live',
  'News Karnataka',        'Aalto',              'Sportskeeda',
  'Financial Express',     'Science',            'Anadolu Ajansi',
  'The Tribune',           'NewsX World',        '30 Stades',
  'Investment Guru India', 'CricTracker',        'Republic World',
  'Trak.in',               'NDTV Profit',        'DW',
  'Curlytales',            'Mid-day',            'NASA',
  'ABP',                   'Wisden',             'E4M',
  'UNICEF',                'Factly',             'Pokerbaazi',
  'BreezyScroll',          'Hub News',           'Aninews',
  'Abplive',               'Defence XP',         'ESPN',
  'Northeast Now',         'The CSR Journal',    'Medical Dialogues',
  'The Siasat Daily',      'Bollywood Hungama',  'ABP ',
  'It Voice',              'Linkedin',           'Indian Express',
  'JACC',                  'Republic TV',        'Theprint',
  'Boom Live',             'The Core',           'TASS',
  'PTI ',                  'GWR',                'Utah.edu'
]

and these are our categories [
  'General',
  'politics',
  'national',
  'world',
  'sports',
  'entertainment',
  'science',
  'technology',
  'IPL_2025',
  'IPL',
  'business',
  'hatke',
  'city',
  'crime',
  'startup',
  'miscellaneous',
  'cricket',
  'Health___Fitness',
  'fashion',
  'Israel-Hamas_War',
  'facts',
  'FINANCE',
  'education',
  'travel',
  'Russia-Ukraine_Conflict',
  'EXPLAINERS',
  'bollywood',
  'automobile',
  'DEFENCE',
  'Feel_Good_Stories'
]

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
	var articles []dto.NewsArticleResponse
	// var err error // Declare err here to avoid redeclaration in switch cases

	switch geminiResponse.Intent {
	case "category":
		for _, e := range geminiResponse.Entities {
			if e.Type == "category" {
				filter = primitive.M{"category": e.Value}
				break
			}
		}
		articles, err = services.FindNews(filter, page, pageSize)

	case "source":
		for _, e := range geminiResponse.Entities {
			if e.Type == "source" {
				filter = primitive.M{"source_name": primitive.Regex{Pattern: "^" + e.Value + "$", Options: "i"}}
				break
			}
		}
		articles, err = services.FindNews(filter, page, pageSize)

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
		articles, err = services.FindNews(filter, page, pageSize)

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
		articles, err = services.FindNews(filter, page, pageSize)

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
		articles, err = services.FindNews(filter, page, pageSize)

	default:
		utils.ErrorResponse(c, 400, "Unknown intent from Gemini: "+geminiResponse.Intent)
		return
	}

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve news: "+err.Error())
		return
	}

	vectorArticles, err := SearchNewsByVectorEmbedding(c, userQuery)
	fmt.Println(vectorArticles, "Found Vectors ")
	if err == nil {
		articles = deduplicateArticles(articles, vectorArticles)
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

// Add this helper function
func deduplicateArticles(articles1, articles2 []dto.NewsArticleResponse) []dto.NewsArticleResponse {
	seen := make(map[string]bool)
	var result []dto.NewsArticleResponse

	// Add first set
	for _, article := range articles1 {
		idStr := article.ID.Hex()
		if !seen[idStr] {
			seen[idStr] = true
			result = append(result, article)
		}
	}

	// Add second set (only if not already seen)
	for _, article := range articles2 {
		idStr := article.ID.Hex()
		if !seen[idStr] {
			seen[idStr] = true
			result = append(result, article)
		}
	}

	return result
}

func SearchNewsByVectorEmbedding(c *gin.Context, userQuery string) ([]dto.NewsArticleResponse, error) {
	embedding, err := services.GetEmbeddingsfromText(userQuery)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get embedding for search query: "+err.Error())
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	articles, err := services.FindNewsByVectorEmbedding(embedding, 1, 10)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to retrieve news by vector embedding: "+err.Error())
		return nil, fmt.Errorf("failed to find news by vector embedding: %w", err)
	}

	return articles, nil
}

func GetEmbeddingsHandler(c *gin.Context) {
	text := c.Query("text")
	if text == "" {
		utils.ErrorResponse(c, 400, "Text parameter is missing")
		return
	}

	embedding, err := services.GetEmbeddingsfromText(text)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get embedding: "+err.Error())
		return
	}

	utils.SuccessResponse(c, embedding)
}
