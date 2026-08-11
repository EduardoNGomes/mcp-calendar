package mcp

import (
	"context"
	"fmt"

	calendarservice "github.com/egomes/google-calendar-mcp-tool/internal/calendar"
	"github.com/egomes/google-calendar-mcp-tool/internal/googleauth"
	"github.com/egomes/google-calendar-mcp-tool/internal/mcp/tools"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func Run(ctx context.Context, credentialsPath, tokenPath string) error {
	auth, err := googleauth.New(credentialsPath, tokenPath)
	if err != nil {
		return err
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "google-calendar",
		Version: "0.1.0",
	}, nil)

	tools.Register(server, auth, calendarservice.New(auth))

	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return fmt.Errorf("run MCP server: %w", err)
	}
	return nil
}
