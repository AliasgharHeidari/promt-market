package utils

import "os"

// Getenv returns the value of the environment variable or the fallback if
// the variable is unset or empty.
func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}