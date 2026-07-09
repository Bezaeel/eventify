// Package config reads process configuration from the environment.
//
// Nothing here is eventify-specific: api, outbox and subscribers each build
// their own config struct from these helpers. Secrets are read from the
// environment, never embedded in source.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// String returns the value of key, or def when unset.
func String(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// MustString returns the value of key, or an error when unset. Use for values
// with no safe default — database passwords, JWT secrets.
func MustString(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

// Int returns the value of key parsed as an int, or def when unset or invalid.
func Int(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// Duration returns the value of key parsed as a duration, or def when unset or
// invalid.
func Duration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
