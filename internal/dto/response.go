package dto

type HelloGetResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type HelloPostResponse struct {
	Message   string `json:"message"`
	FullName  string `json:"full_name"`
	Timestamp string `json:"timestamp"`
}
