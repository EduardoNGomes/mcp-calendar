package tools

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	calendarservice "github.com/egomes/google-calendar-mcp-tool/internal/calendar"
	"github.com/egomes/google-calendar-mcp-tool/internal/googleauth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultTimeZone = "America/Sao_Paulo"

type AuthenticateOutput struct {
	Authenticated    bool   `json:"authenticated"`
	AuthorizationURL string `json:"authorizationUrl,omitempty"`
	BrowserOpened    bool   `json:"browserOpened"`
	Message          string `json:"message"`
}

type AuthStatusOutput struct {
	Authenticated bool `json:"authenticated"`
}

type ListEventsInput struct {
	CalendarID string `json:"calendarId,omitempty" jsonschema:"calendar identifier; defaults to primary"`
	Start      string `json:"start,omitempty" jsonschema:"range start in RFC3339; defaults to now"`
	End        string `json:"end,omitempty" jsonschema:"range end in RFC3339; defaults to seven days after start"`
	MaxResults int64  `json:"maxResults,omitempty" jsonschema:"maximum number of events from 1 to 100; defaults to 20"`
}

type ListEventsOutput struct {
	Events []calendarservice.Event `json:"events"`
}

type CreateEventInput struct {
	CalendarID  string   `json:"calendarId,omitempty" jsonschema:"calendar identifier; defaults to primary"`
	Summary     string   `json:"summary" jsonschema:"event title"`
	Description string   `json:"description,omitempty" jsonschema:"optional event description"`
	Start       string   `json:"start" jsonschema:"event start in RFC3339"`
	End         string   `json:"end" jsonschema:"event end in RFC3339"`
	TimeZone    string   `json:"timeZone,omitempty" jsonschema:"IANA timezone; defaults to America/Sao_Paulo"`
	Recurrence  []string `json:"recurrence,omitempty" jsonschema:"optional RFC5545 recurrence lines, for example RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR"`
	Attendees   []string `json:"attendees,omitempty" jsonschema:"guest email addresses; Google Calendar sends an invitation to each guest"`
	CreateMeet  bool     `json:"createMeet,omitempty" jsonschema:"create a Google Meet conference for the event"`
}

type CreateEventOutput struct {
	Event calendarservice.Event `json:"event"`
}

func Register(
	server *mcp.Server,
	auth *googleauth.Service,
	calendar *calendarservice.Service,
) {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}
	nonDestructive := false
	additive := &mcp.ToolAnnotations{
		DestructiveHint: &nonDestructive,
		ReadOnlyHint:    false,
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "authenticate",
		Description: "Start Google Calendar OAuth login and return the authorization URL",
		Annotations: additive,
	}, authenticate(auth))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "auth_status",
		Description: "Check whether Google Calendar authentication is available",
		Annotations: readOnly,
	}, authStatus(auth))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_events",
		Description: "List Google Calendar events in a time range",
		Annotations: readOnly,
	}, listEvents(calendar))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_event",
		Description: "Create a single or recurring Google Calendar event",
		Annotations: additive,
	}, createEvent(calendar))
}

func authenticate(auth *googleauth.Service) mcp.ToolHandlerFor[struct{}, AuthenticateOutput] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		input struct{},
	) (*mcp.CallToolResult, AuthenticateOutput, error) {
		authorization, err := auth.StartAuthorization()
		if err != nil {
			return nil, AuthenticateOutput{}, err
		}

		message := "Abra a URL de autorização e conclua o login no Google."
		if authorization.Authenticated {
			message = "Google Calendar já está autenticado."
		} else if authorization.BrowserOpened {
			message = "O navegador foi aberto. Conclua o login no Google."
		}

		return nil, AuthenticateOutput{
			Authenticated:    authorization.Authenticated,
			AuthorizationURL: authorization.AuthorizationURL,
			BrowserOpened:    authorization.BrowserOpened,
			Message:          message,
		}, nil
	}
}

func authStatus(auth *googleauth.Service) mcp.ToolHandlerFor[struct{}, AuthStatusOutput] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		input struct{},
	) (*mcp.CallToolResult, AuthStatusOutput, error) {
		return nil, AuthStatusOutput{Authenticated: auth.IsAuthenticated()}, nil
	}
}

func listEvents(calendar *calendarservice.Service) mcp.ToolHandlerFor[ListEventsInput, ListEventsOutput] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		input ListEventsInput,
	) (*mcp.CallToolResult, ListEventsOutput, error) {
		start, end, err := eventRange(input.Start, input.End)
		if err != nil {
			return nil, ListEventsOutput{}, err
		}

		calendarID := defaultString(input.CalendarID, "primary")
		maxResults := input.MaxResults
		if maxResults == 0 {
			maxResults = 20
		}
		if maxResults < 1 || maxResults > 100 {
			return nil, ListEventsOutput{}, errors.New("maxResults must be between 1 and 100")
		}

		events, err := calendar.ListEvents(ctx, calendarID, start, end, maxResults)
		if err != nil {
			return nil, ListEventsOutput{}, err
		}
		return nil, ListEventsOutput{Events: events}, nil
	}
}

func createEvent(calendar *calendarservice.Service) mcp.ToolHandlerFor[CreateEventInput, CreateEventOutput] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		input CreateEventInput,
	) (*mcp.CallToolResult, CreateEventOutput, error) {
		if input.Summary == "" {
			return nil, CreateEventOutput{}, errors.New("summary is required")
		}

		start, end, err := requiredEventRange(input.Start, input.End)
		if err != nil {
			return nil, CreateEventOutput{}, err
		}

		attendees, err := validAttendees(input.Attendees)
		if err != nil {
			return nil, CreateEventOutput{}, err
		}

		created, err := calendar.CreateEvent(ctx, calendarservice.CreateEvent{
			CalendarID:  defaultString(input.CalendarID, "primary"),
			Summary:     input.Summary,
			Description: input.Description,
			Start:       start,
			End:         end,
			TimeZone:    defaultString(input.TimeZone, defaultTimeZone),
			Recurrence:  input.Recurrence,
			Attendees:   attendees,
			CreateMeet:  input.CreateMeet,
		})
		if err != nil {
			return nil, CreateEventOutput{}, err
		}

		return nil, CreateEventOutput{Event: created}, nil
	}
}

func validAttendees(values []string) ([]string, error) {
	attendees := make([]string, 0, len(values))
	for _, value := range values {
		email := strings.TrimSpace(value)
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email {
			return nil, fmt.Errorf("invalid attendee email: %q", value)
		}
		attendees = append(attendees, email)
	}
	return attendees, nil
}

func eventRange(startValue, endValue string) (time.Time, time.Time, error) {
	start := time.Now()
	var err error
	if startValue != "" {
		start, err = time.Parse(time.RFC3339, startValue)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start: %w", err)
		}
	}

	end := start.Add(7 * 24 * time.Hour)
	if endValue != "" {
		end, err = time.Parse(time.RFC3339, endValue)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end: %w", err)
		}
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, errors.New("end must be after start")
	}
	return start, end, nil
}

func requiredEventRange(startValue, endValue string) (time.Time, time.Time, error) {
	if startValue == "" || endValue == "" {
		return time.Time{}, time.Time{}, errors.New("start and end are required")
	}
	return eventRange(startValue, endValue)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
