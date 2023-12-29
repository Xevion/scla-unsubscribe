package main

type Entry struct {
	Id         string
	Name       string
	JobTitle   string // Subtitle
	Department string
	College    string // Subtitle
	Phone      string
}

type ConfirmationResponse struct {
	FormId              string `json:"formId"`
	FollowUpUrl         string `json:"followUpUrl"`
	DeliveryType        string `json:"deliveryType"`
	FollowUpStreamValue string `json:"followUpStreamValue"`
	AliId               string `json:"aliId"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}
