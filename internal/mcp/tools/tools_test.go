package tools

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server"}, nil)
	Register(server, nil, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"auth_status", "authenticate", "create_event", "delete_events", "list_events", "respond_to_event"}
	if len(result.Tools) != len(want) {
		t.Fatalf("tool count = %d; want %d", len(result.Tools), len(want))
	}
	for index, name := range want {
		if result.Tools[index].Name != name {
			t.Fatalf("tool %d = %q; want %q", index, result.Tools[index].Name, name)
		}
	}
}

func TestRequiredEventRange(t *testing.T) {
	start, end, err := requiredEventRange(
		"2026-08-12T06:00:00-03:00",
		"2026-08-12T07:00:00-03:00",
	)
	if err != nil {
		t.Fatal(err)
	}
	if end.Sub(start) != time.Hour {
		t.Fatalf("duration = %s; want 1h", end.Sub(start))
	}
}

func TestRequiredEventRangeRejectsInvalidRange(t *testing.T) {
	_, _, err := requiredEventRange(
		"2026-08-12T07:00:00-03:00",
		"2026-08-12T06:00:00-03:00",
	)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidAttendees(t *testing.T) {
	attendees, err := validAttendees([]string{"guest@example.com", " second@example.com "})
	if err != nil {
		t.Fatal(err)
	}

	if len(attendees) != 2 || attendees[1] != "second@example.com" {
		t.Fatalf("attendees = %v; want normalized email addresses", attendees)
	}
}

func TestValidAttendeesRejectsInvalidEmail(t *testing.T) {
	_, err := validAttendees([]string{"not-an-email"})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestRespondToEventRequiresEventID(t *testing.T) {
	handler := respondToEvent(nil)
	_, _, err := handler(context.Background(), nil, RespondToEventInput{
		ResponseStatus: "accepted",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestRespondToEventRejectsInvalidStatus(t *testing.T) {
	handler := respondToEvent(nil)
	_, _, err := handler(context.Background(), nil, RespondToEventInput{
		EventID:        "event-id",
		ResponseStatus: "maybe",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDeleteEventsRequiresEventID(t *testing.T) {
	handler := deleteEvents(nil)
	_, _, err := handler(context.Background(), nil, DeleteEventsInput{})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidEventIDsNormalizesAndRemovesDuplicates(t *testing.T) {
	eventIDs, err := validEventIDs([]string{" first ", "second", "first"})
	if err != nil {
		t.Fatal(err)
	}

	if len(eventIDs) != 2 || eventIDs[0] != "first" || eventIDs[1] != "second" {
		t.Fatalf("event IDs = %v; want [first second]", eventIDs)
	}
}

func TestValidEventIDsRejectsEmptyID(t *testing.T) {
	_, err := validEventIDs([]string{"event-id", " "})
	if err == nil {
		t.Fatal("expected an error")
	}
}
