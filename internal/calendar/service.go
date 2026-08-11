package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/egomes/google-calendar-mcp-tool/internal/googleauth"
	calendarapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Event struct {
	ID          string   `json:"id"`
	Summary     string   `json:"summary"`
	Description string   `json:"description,omitempty"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	HTMLLink    string   `json:"htmlLink,omitempty"`
	Recurrence  []string `json:"recurrence,omitempty"`
}

type CreateEvent struct {
	CalendarID  string
	Summary     string
	Description string
	Start       time.Time
	End         time.Time
	TimeZone    string
	Recurrence  []string
}

type Service struct {
	auth *googleauth.Service
}

func New(auth *googleauth.Service) *Service {
	return &Service{auth: auth}
}

func (s *Service) ListEvents(
	ctx context.Context,
	calendarID string,
	start time.Time,
	end time.Time,
	maxResults int64,
) ([]Event, error) {
	api, err := s.api(ctx)
	if err != nil {
		return nil, err
	}

	result, err := api.Events.List(calendarID).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(maxResults).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("list Calendar events: %w", err)
	}

	events := make([]Event, 0, len(result.Items))
	for _, item := range result.Items {
		events = append(events, eventFromAPI(item))
	}
	return events, nil
}

func (s *Service) CreateEvent(ctx context.Context, input CreateEvent) (Event, error) {
	api, err := s.api(ctx)
	if err != nil {
		return Event{}, err
	}

	event := &calendarapi.Event{
		Summary:     input.Summary,
		Description: input.Description,
		Start: &calendarapi.EventDateTime{
			DateTime: input.Start.Format(time.RFC3339),
			TimeZone: input.TimeZone,
		},
		End: &calendarapi.EventDateTime{
			DateTime: input.End.Format(time.RFC3339),
			TimeZone: input.TimeZone,
		},
		Recurrence: input.Recurrence,
	}

	created, err := api.Events.Insert(input.CalendarID, event).Context(ctx).Do()
	if err != nil {
		return Event{}, fmt.Errorf("create Calendar event: %w", err)
	}

	return eventFromAPI(created), nil
}

func (s *Service) api(ctx context.Context) (*calendarapi.Service, error) {
	client, err := s.auth.Client(ctx)
	if err != nil {
		return nil, err
	}

	api, err := calendarapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("create Calendar client: %w", err)
	}
	return api, nil
}

func eventFromAPI(event *calendarapi.Event) Event {
	return Event{
		ID:          event.Id,
		Summary:     event.Summary,
		Description: event.Description,
		Start:       eventTime(event.Start),
		End:         eventTime(event.End),
		HTMLLink:    event.HtmlLink,
		Recurrence:  event.Recurrence,
	}
}

func eventTime(value *calendarapi.EventDateTime) string {
	if value == nil {
		return ""
	}
	if value.DateTime != "" {
		return value.DateTime
	}
	return value.Date
}
