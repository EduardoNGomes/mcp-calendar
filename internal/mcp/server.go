package mcp

import (
	"context"
	"log"

	"github.com/egomes/google-calendar-mcp-tool/internal/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Server() {

	server := mcp.NewServer(&mcp.Implementation{Name: "Google Calendar", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, tools.SayHi)

	// Run the server over stdin/stdout, until the client disconnects.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
