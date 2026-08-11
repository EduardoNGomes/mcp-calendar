package main

import (
	"context"
	"log"
	"os"

	"github.com/egomes/google-calendar-mcp-tool/internal/mcp"
)

func main() {
	credentialsPath := envOrDefault("GOOGLE_CREDENTIALS_FILE", "google_credentials.json")
	tokenPath := envOrDefault("GOOGLE_TOKEN_FILE", "data/token.json")

	if err := mcp.Run(context.Background(), credentialsPath, tokenPath); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
