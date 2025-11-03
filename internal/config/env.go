package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// RequiredString returns the value of the environment variable identified by key.
// It returns an error if the variable is unset or contains only whitespace.
func RequiredString(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

// String returns the value of the environment variable identified by key or defaultValue when it is unset.
func String(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value != "" {
		return value
	}
	return defaultValue
}

// Int returns the integer value of the environment variable identified by key or defaultValue when it is unset or malformed.
func Int(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

// SplitAndTrim splits a comma separated string and removes empty items.
func SplitAndTrim(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
