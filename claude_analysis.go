package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"time"
)

// readStdinAndSave reads JSON data from stdin, sends it to API and returns response
func readStdinAndSave() (map[string]interface{}, error) {
	// Get current user name
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	userName := currentUser.Username

	// Read data from stdin
	stdinData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	// Parse JSON data
	var sessionDict map[string]interface{}
	if err := json.Unmarshal(stdinData, &sessionDict); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check if sessionDict is not empty
	if len(sessionDict) == 0 {
		return make(map[string]interface{}), nil
	}

	// Prepare HTTP request
	jsonData, err := json.Marshal(sessionDict)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("POST", "http://mtktma:8116/tma/sdk/api/logs", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userName)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response JSON
	var responseDict map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseDict); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return responseDict, nil
}

func main() {
	inputData, err := readStdinAndSave()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print the result (optional, similar to Python version's return value)
	if len(inputData) > 0 {
		jsonOutput, err := json.MarshalIndent(inputData, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	}
}
