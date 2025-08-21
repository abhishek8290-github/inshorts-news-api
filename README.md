# Inshorts News API (Go + Gin + MongoDB + Redis)

Production-ready REST API for ingesting, querying, and trending news articles. Built with Go (Gin), MongoDB (with geospatial and vector search), Redis caching, and scheduled jobs for precomputing trending results. Includes an intelligent search router powered by Google Gemini.

- Module: `news-api`
- Go: 1.23
- HTTP Port: 8080 (fixed in code)

## Features

- News ingestion (single and bulk) with:
  - URL de-duplication
  - Vector embeddings via external service (for semantic search)
  - LLM summary generation via external service
  - GeoJSON location storage
- Query APIs:
  - Category / Source / Score filters
  - Nearby geospatial query
  - Categories and Sources discovery
  - Smart search router (uses Gemini to classify intent and route/query)
  - Optional vector-based semantic search merge
- Trending:
  - User event ingestion (view/click/share) with location
  - Aggregation of trending by time window (6h, 24h, week)
  - Redis caching with hourly pre-warming (cron)
- Observability:
  - Structured JSON responses
  - Request logging middleware
  - Health check and DB test endpoints
- Containerization with Docker

---

## Architecture Overview

- main.go
  - Loads environment
  - Initializes MongoDB and Redis
  - Starts a cron scheduler (hourly) to precompute trending caches
  - Mounts routes on Gin
- internal/routes/routes.go
  - Defines `/api/v1/news/*` endpoints (and `/ping`)
- internal/handlers
  - news_handler.go: HTTP handlers for news APIs and smart router
- internal/services
  - news_service.go: business logic (embedding, summarization, Mongo queries, vector search)
  - trending_service.go: event aggregation, cache read/write, cron precompute
- internal/database
  - connection.go: MongoDB client and database accessor
  - redis.go: Redis client
- internal/models
  - news.go: Article and Location models
  - trending.go: UserEvent and TrendingArticle models
- internal/dto
  - request.go, news.go, response.go: DTOs
- internal/middleware/logger.go: request logging
- internal/utils/response.go: JSON response helpers
- Dockerfile: multi-stage build

---

## Prerequisites

- Go 1.23+
- MongoDB Atlas (recommended) or MongoDB 7.0+ with:
  - 2dsphere index on `location`
  - Atlas Search (Vector Search) index named `vector_index` on `vector_embedding`
- Redis (e.g., Redis Cloud)
- External microservice for embeddings and summarization running at:
  - POST http://localhost:8001/embed  → `{"embedding": []float64}`
  - POST http://localhost:8001/summarize → `{"summary": string, "title": string, "status": "success"}`
- (Optional for smart router) Google Gemini API key

---

## Environment Variables

Create a `.env` file in project root (Dockerfile copies it into the image; do not do this in production):

Required:
- MONGODB_URI: Mongo connection string
- DATABASE_NAME: Database name (e.g., `news_db`)
- REDIS_ENDPOINT: host:port
- REDIS_PASSWORD: Redis auth password (DB=0 used)
- GEMINI_API_KEY: Required for smart router (handlers/news_handler.go)

Optional:
- PORT: present in `.env` but not used (server binds on :8080 in code)
- OPENAI_API_KEY: not used by current code
- COLLECTION_NAME: present in `.env` but the code uses `news_articles` collection name directly

Example:
```
MONGODB_URI=mongodb+srv://user:pass@cluster/...
DATABASE_NAME=news_db
PORT=8080
GEMINI_API_KEY=your_gemini_key
REDIS_ENDPOINT=your-redis-endpoint:port
REDIS_PASSWORD=your-redis-password
```

Security note: The repository `.env` contains real-looking secrets. Rotate them immediately and avoid committing secrets in future.

---

## MongoDB Indexes

1) Geospatial index for nearby query:
```
use news_db
db.news_articles.createIndex({ location: "2dsphere" })
```

2) Atlas Vector Search index for semantic search:
- Create an Atlas Search index (name: `vector_index`) on collection `news_articles`:
Example index definition (Atlas Vector Search):
```json
{
  "fields": [
    {
      "type": "vector",
      "path": "vector_embedding",
      "numDimensions": 768,
      "similarity": "cosine"
    }
  ]
}
```
- The code uses `$vectorSearch` with `"index": "vector_index"` in `FindNewsByVectorEmbedding`.

3) Recommended:
- `news_articles` on `url` unique (application de-duplicates; DB unique index further enforces):
```
db.news_articles.createIndex({ url: 1 }, { unique: true })
```
- `user_events` on `timestamp`, `article_id`:
```
db.user_events.createIndex({ timestamp: -1 })
db.user_events.createIndex({ article_id: 1 })
```

---

## Running Locally

1) Install dependencies and run:
```
go mod download
go run .
```
Server listens on http://localhost:8080

2) Ensure dependencies are running/accessible:
- MongoDB (Atlas or local) with required indexes
- Redis reachable via REDIS_ENDPOINT
- Embedding/Summarization service listening on http://localhost:8001

3) Health checks:
- GET /ping → `{"message": "pong"}`
- GET /test-db → validates Mongo connectivity

---

## Docker

Build:
```
docker build -t inshorts-news-api:latest .
```

Run (ensure `.env` is present in project root; note: Dockerfile copies it into the image):
```
docker run --rm -p 8080:8080 --name news-api inshorts-news-api:latest
```

Production note:
- Do NOT bake `.env` into the image. Change the Dockerfile to rely on `-e` or `--env-file`:
```
docker run --rm -p 8080:8080 --env-file .env inshorts-news-api:latest
```

---

## Data Models

Article (Mongo: `news_articles`):
```
{
  _id: ObjectId,
  title: string,
  description: string,
  url: string,
  publication_date: ISODate,
  source_name: string,
  category: [string],
  relevance_score: number,
  location: { type: "Point", coordinates: [lon, lat] },
  llm_summary: string,
  vector_embedding: [number]  // optional
}
```

UserEvent (Mongo: `user_events`):
```
{
  _id: ObjectId,
  user_id: string,
  article_id: string,      // hex ObjectId stored as string
  event_type: "view"|"click"|"share",
  timestamp: ISODate,
  location: { type: "Point", coordinates: [lon, lat] },
  metadata: { [k: string]: string } // optional
}
```

Response wrapper (utils/response.go):
- Success: `{"success": true, "data": any}`
- Error: `{"success": false, "error": "message"}`

Pagination (where supported):
- Query params: `page` (default 1), `pageSize` (default 10)

---

## API Reference

Base URL: `http://localhost:8080/api/v1`

Health:
- GET `/ping` → `{ "message": "pong" }`
- GET `/test-db` → `{ "message": "Database connected successfully!", "database": "news_db" }`

News (prefix `/api/v1/news`):

Ingest
- POST `/`  
  Body:
  ```
  {
    "title": "string",
    "description": "string",
    "url": "https://...",
    "publication_date": "2025-08-20T12:34:56Z",
    "source_name": "News18",
    "category": ["world","business"],
    "relevance_score": 0.85,
    "latitude": 28.6139,
    "longitude": 77.2090
  }
  ```
  Behavior:
  - De-duplicates by `url`
  - Calls `/embed` and `/summarize` on external service
  - Stores GeoJSON location (lon,lat)
  - Returns article

- POST `/list` (bulk)  
  Body: array of the same objects as above  
  Behavior: skips duplicates; embeds and summarizes per item

Discovery
- GET `/categories` → `[]string`
- GET `/sources` → `[]string`

Filter
- GET `/category/:category?page=&pageSize=` → articles by exact category
- GET `/source/:source? page=&pageSize=` → articles by exact source (case-insensitive exact match)
- GET `/score/:score? page=&pageSize=` → articles with `relevance_score >= score`
- GET `/nearby?lat=..&lon=..&radius=..&page=&pageSize=`  
  - `radius` in kilometers; uses `$geoWithin: $centerSphere` with Earth radius 6378.1 km

Search (Smart Router)
- GET `/search?q=...&page=&pageSize=[...]`  
  Uses Gemini to parse intent into one of:
  - `category` | `source` | `score` | `search` | `nearby` | `vector_search`
  - Builds Mongo filters from extracted entities
  - Fallbacks to regex search on title/description
  - Also attempts vector search (`$vectorSearch`) and merges deduplicated results

Trending
- POST `/events`  
  Body:
  ```
  {
    "user_id": "u123",
    "article_id": "64e....",   // hex ObjectID string
    "event_type": "view" | "click" | "share",
    "latitude": 28.61,
    "longitude": 77.20
  }
  ```
  Creates a `user_events` record.

- GET `/trending?window=6h|24h|week&limit=10`  
  Reads from Redis if cached; otherwise computes and caches.  
  Trending score = views*1 + clicks*2 + shares*3 (within the chosen window).

---

## cURL Examples

Ingest one:
```
curl -X POST http://localhost:8080/api/v1/news/ \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Sample Title",
    "description": "Short description",
    "url": "https://example.com/article-1",
    "publication_date": "2025-08-20T12:34:56Z",
    "source_name": "News18",
    "category": ["world"],
    "relevance_score": 0.9,
    "latitude": 28.61,
    "longitude": 77.20
  }'
```

Bulk ingest:
```
curl -X POST http://localhost:8080/api/v1/news/list \
  -H "Content-Type: application/json" \
  -d '[{ ... }, { ... }]'
```

Category:
```
curl "http://localhost:8080/api/v1/news/category/world?page=1&pageSize=10"
```

Nearby:
```
curl "http://localhost:8080/api/v1/news/nearby?lat=28.61&lon=77.20&radius=25&page=1&pageSize=10"
```

Search (smart):
```
curl "http://localhost:8080/api/v1/news/search?q=India from News18 last week"
```

Event:
```
curl -X POST http://localhost:8080/api/v1/news/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "u123",
    "article_id": "64e0f2bb7e9cbb7f8f8a1111",
    "event_type": "click",
    "latitude": 28.61,
    "longitude": 77.20
  }'
```

Trending:
```
curl "http://localhost:8080/api/v1/news/trending?window=24h&limit=10"
```

---

## Scheduled Jobs (Cron)

- A cron (`robfig/cron`) runs hourly:
  - Calls `services.ScheduleGlobalTrendingCalculations()`
  - Precomputes and stores trending lists in Redis for keys:
    - `trending:articles:6h`, `trending:articles:24h`, `trending:articles:week`
  - TTL for cached entries: 1 hour

---

## External Embedding/Summarization Service

The API expects a local service on port 8001:

- POST `/embed`
  - Request: `{"text": "some content"}`
  - Response: `{"embedding": [0.1, 0.2, ...]}`

- POST `/summarize`
  - Request: `{"url": "https://..."}`
  - Response: `{"summary": "text", "title": "text", "status": "success"}`
  - For YouTube URLs, the API skips summarization and stores empty string.

You can stub this service in development if needed.

---

## Development Notes

- Logging: see `internal/middleware/logger.go`
- Responses: wrap via `utils.SuccessResponse`/`ErrorResponse`
- The smart router uses `GEMINI_API_KEY` and `google.golang.org/api` + `genai` client
- Server port is currently hardcoded to 8080 in `main.go`

---

## Roadmap / Improvements

- Parameterize server port via env
- Proper config package with validation
- Rate limiting and auth
- Better error codes and error wrapping
- Postman / OpenAPI spec
- Do not copy `.env` into Docker images; use runtime envs

---
