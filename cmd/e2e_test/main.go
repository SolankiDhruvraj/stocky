package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const baseURL = "http://localhost:8080"

func main() {
	// Wait for server to start
	time.Sleep(2 * time.Second)

	// 1. Health Check
	checkEndpoint("GET", "/health", nil, 200)

	// 2. Create Reward
	userID := "e2e-user"
	rewardID := createReward(userID)
	fmt.Printf("Created Reward ID: %s\n", rewardID)

	// 3. Get Today's Stocks
	checkEndpoint("GET", "/today-stocks/"+userID, nil, 200)

	// 4. Get Stats
	checkEndpoint("GET", "/stats/"+userID, nil, 200)

	// 5. Get Portfolio
	checkEndpoint("GET", "/portfolio/"+userID, nil, 200)

	// 6. Get Historical INR
	checkEndpoint("GET", "/historical-inr/"+userID, nil, 200)

	// 7. Revert Reward
	revertReward(rewardID)

	// 8. Verify Portfolio 
	checkEndpoint("GET", "/portfolio/"+userID, nil, 200)

	fmt.Println("ALL TESTS PASSED")
}

func checkEndpoint(method, path string, body interface{}, expectedStatus int) {
	fmt.Printf("Testing %s %s...\n", method, path)
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, _ := http.NewRequest(method, baseURL+path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		log.Fatalf("Expected status %d, got %d. Body: %s", expectedStatus, resp.StatusCode, string(respBody))
	}
	fmt.Printf("Response: %s\n", string(respBody))
}

func createReward(userID string) string {
	fmt.Println("Creating reward...")
	reqBody := map[string]interface{}{
		"user_id":         userID,
		"symbol":          "INFY",
		"quantity":        "10.0",
		"timestamp":       time.Now().Format(time.RFC3339),
		"idempotency_key": fmt.Sprintf("e2e-key-%d", time.Now().UnixNano()),
		"source":          "e2e-test",
	}
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(baseURL+"/reward", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Fatalf("Create reward failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Create reward failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)
	return res["reward_id"]
}

func revertReward(rewardID string) {
	fmt.Printf("Reverting reward %s...\n", rewardID)
	req, _ := http.NewRequest("POST", baseURL+"/reward/"+rewardID+"/revert", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Revert failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Revert failed with status %d: %s", resp.StatusCode, string(body))
	}
	fmt.Println("Revert successful")
}
