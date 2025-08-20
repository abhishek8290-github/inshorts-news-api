package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(req *gin.Context) {
		start := time.Now()

		// Process request
		req.Next()

		// Log after request
		duration := time.Since(start)
		fmt.Printf("[%s] %s %s - %v - %d\n",
			start.Format("2006-01-02 15:04:05"),
			req.Request.Method,
			req.Request.URL.Path,
			duration,
			req.Writer.Status(),
		)
	}
}
