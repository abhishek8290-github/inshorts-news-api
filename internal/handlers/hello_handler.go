package handlers

import (
	"news-api/internal/dto"
	"news-api/internal/services"
	"news-api/internal/utils"

	"github.com/gin-gonic/gin"
)

// GET /hello
func GetHello(c *gin.Context) {
	// Call service layer
	response := services.GetHelloMessage()

	// Return success response
	utils.SuccessResponse(c, response)
}

// POST /hello
func PostHello(c *gin.Context) {
	// 1. Parse and validate request
	var req dto.HelloPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid input: "+err.Error())
		return
	}

	// 2. Call service layer
	response := services.ProcessHelloPost(&req)

	// 3. Return success response
	utils.SuccessResponse(c, response)
}
