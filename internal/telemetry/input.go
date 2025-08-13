package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReadJSONFromStdin reads and parses JSON data from standard input
func ReadJSONFromStdin() (map[string]interface{}, error) {
	// Read data from stdin
	stdinData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	// Parse JSON data
	var data map[string]interface{}
	if err := json.Unmarshal(stdinData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data, nil
}
