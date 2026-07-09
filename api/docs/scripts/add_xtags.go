package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

func main() {
	// Load swagger.json
	data, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		fmt.Println("Error reading swagger.json:", err)
		os.Exit(1)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		fmt.Println("Error parsing swagger.json:", err)
		os.Exit(1)
	}

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		fmt.Println("No paths found in swagger.json")
		os.Exit(1)
	}

	// Regex to extract version from path, e.g. /api/v1/
	re := regexp.MustCompile(`/api/(v[0-9]+)/`)

	for path, ops := range paths {
		matches := re.FindStringSubmatch(path)
		if len(matches) == 2 {
			version := matches[1]
			operations, ok := ops.(map[string]interface{})
			if !ok {
				continue
			}
			for _, op := range operations {
				opMap, ok := op.(map[string]interface{})
				if !ok {
					continue
				}
				// Add x-tags only if not present
				if _, exists := opMap["x-tags"]; !exists {
					opMap["x-tags"] = []string{version}
				}
			}
		}
	}

	// Save the modified swagger.json
	out, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling modified swagger.json:", err)
		os.Exit(1)
	}
	if err := os.WriteFile("docs/swagger.json", out, 0644); err != nil {
		fmt.Println("Error writing swagger.json:", err)
		os.Exit(1)
	}

	fmt.Println("x-tags injected successfully.")
}
