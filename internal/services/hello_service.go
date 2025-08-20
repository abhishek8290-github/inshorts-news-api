package services

import (
	"fmt"
	"news-api/internal/dto"
	"time"
)

func GetHelloMessage() *dto.HelloGetResponse {
	return &dto.HelloGetResponse{
		Message: "Hello from Go Backend!",
		Status:  "Active",
	}
}

func ProcessHelloPost(req *dto.HelloPostRequest) *dto.HelloPostResponse {
	fullName := fmt.Sprintf("%s %s", req.FirstName, req.LastName)

	return &dto.HelloPostResponse{
		Message:   fmt.Sprintf("Hello, %s! Welcome to our API.", fullName),
		FullName:  fullName,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
