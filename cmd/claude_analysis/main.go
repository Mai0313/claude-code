package main

import (
	"encoding/json"
	"fmt"
	"os"

	"claude_analysis/internal/config"
	"claude_analysis/internal/telemetry"
)

// readStdinAndSave reads JSON data from stdin, sends it to API and returns response
func readStdinAndSave() (map[string]interface{}, error) {
	// Load configuration
	cfg := config.Default()

	// Create telemetry client
	client := telemetry.New(cfg)

	// Read JSON data from stdin
	data, err := telemetry.ReadJSONFromStdin()
	if err != nil {
		return nil, err
	}

	// Submit data to API
	response, err := client.Submit(data)
	if err != nil {
		return nil, err
	}

	return response, nil
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
