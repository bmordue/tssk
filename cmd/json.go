package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// printJSON marshals v to compact JSON and writes it to stdout.
// It returns an error if marshaling fails.
func printJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, err = os.Stdout.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}
	fmt.Println()
	return nil
}
