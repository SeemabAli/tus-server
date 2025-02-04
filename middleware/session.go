package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type ValidateData struct {
	Email        string `json:"email"`
	SessionToken string `json:"sessionKey"`
}

type ValidateResponse struct {
	IsValid  bool   `json:"isValid"`
	UserType string `json:"userType"`
}

// ValidateSessionAndPerm checks if the session is valid and returns the user's permission level
// Return values: 0 = Invalid session, 1 = Regular user, 2 = Influencer
func ValidateSessionAndPerm(sessionToken, email string) int {

	const influencerType = "Influencer"
	const validSessionCode = 1
	const influencerSessionCode = 2
	const invalidSessionCode = 0

	// Prepare request payload
	body := ValidateData{
		Email:        email,
		SessionToken: sessionToken,
	}

	// Encode request body to JSON
	bodyWriter := bytes.NewBuffer([]byte{})
	if err := json.NewEncoder(bodyWriter).Encode(body); err != nil {
		log.Printf("Error encoding request body: %v", err)
		return invalidSessionCode
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "https://api.acebeauty.club/api/AceBeauty/isValidUploader", bodyWriter)
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return invalidSessionCode
	}
	req.Header.Add("Content-Type", "application/json")

	// Set up HTTP client with a timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making HTTP request: %v", err)
		return invalidSessionCode
	}
	defer resp.Body.Close() // Ensure the response body is closed

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-200 response: %v", resp.Status)
		return invalidSessionCode
	}

	// Read and decode the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return invalidSessionCode
	}

	// Unmarshal JSON response
	var respData ValidateResponse
	if err := json.Unmarshal(data, &respData); err != nil {
		log.Printf("Error unmarshalling response: %v", err)
		return invalidSessionCode
	}

	// Validate session response
	if !respData.IsValid {
		log.Println("Invalid session")
		return invalidSessionCode
	}

	// Check the user type
	if respData.UserType == influencerType {
		return influencerSessionCode
	}

	return validSessionCode
}
