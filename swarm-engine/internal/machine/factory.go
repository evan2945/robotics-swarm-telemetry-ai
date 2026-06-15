package machine

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadCauldronRegistry parses the JSON blueprint config into active memory.
func LoadCauldronRegistry(path string) (CauldronRegistry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open registry file: %w", err)
	}
	defer file.Close()

	var registry CauldronRegistry
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&registry); err != nil {
		return nil, fmt.Errorf("failed to decode registry JSON: %w", err)
	}

	return registry, nil
}
