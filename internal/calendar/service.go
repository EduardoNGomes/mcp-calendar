package calendar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/egomes/google-calendar-mcp-tool/internal/googleauth"
	calendarapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Event struct {
	ID             string   `json:"id"`
	Summary        string   `json:"summary"`
	Description    string   `json:"description,omitempty"`
	Start          string   `json:"start"`
	End            string   `json:"end"`
	HTMLLink       string   `json:"htmlLink,omitempty"`
	MeetLink       string   `json:"meetLink,omitempty"`
	Attendees      []string `json:"attendees,omitempty"`
	ResponseStatus string   `json:"responseStatus,omitempty"`
	Recurrence     []string `json:"recurrence,omitempty"`
}

type ResponseStatus string

const (
	ResponseAccepted  ResponseStatus = "accepted"
	ResponseDeclined  ResponseStatus = "declined"
	ResponseTentative ResponseStatus = "tentative"
)

type CreateEvent struct {
	CalendarID  string
	Summary     string
	Description string
	Start       time.Time
	End         time.Time
	TimeZone    string
	Recurrence  []string
	Attendees   []string
	CreateMeet  bool
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

	event := apiEvent(input)
	insert := api.Events.Insert(input.CalendarID, event).Context(ctx)

	if len(input.Attendees) > 0 {
		insert.SendUpdates("all")
	}

	if input.CreateMeet {
		conference, err := newConferenceData()
		if err != nil {
			return Event{}, err
		}
		event.ConferenceData = conference
		insert.ConferenceDataVersion(1)
	}

	created, err := insert.Do()
	if err != nil {
		return Event{}, fmt.Errorf("create Calendar event: %w", err)
	}

	return eventFromAPI(created), nil
}

func (s *Service) RespondToEvent(
	ctx context.Context,
	calendarID string,
	eventID string,
	status ResponseStatus,
) (Event, error) {
	if _, err := ParseResponseStatus(string(status)); err != nil {
		return Event{}, err
	}

	api, err := s.api(ctx)
	if err != nil {
		return Event{}, err
	}

	event, err := api.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return Event{}, fmt.Errorf("get Calendar event: %w", err)
	}

	attendee, err := selfAttendee(event)
	if err != nil {
		return Event{}, err
	}
	attendee.ResponseStatus = string(status)

	update := api.Events.Update(calendarID, eventID, event).
		SendUpdates("all").
		Context(ctx)
	if event.Etag != "" {
		update.Header().Set("If-Match", event.Etag)
	}

	updated, err := update.Do()
	if err != nil {
		return Event{}, fmt.Errorf("respond to Calendar event: %w", err)
	}

	return eventFromAPI(updated), nil
}

func apiEvent(input CreateEvent) *calendarapi.Event {
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

	for _, email := range input.Attendees {
		event.Attendees = append(event.Attendees, &calendarapi.EventAttendee{Email: email})
	}

	return event
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
	attendees := make([]string, 0, len(event.Attendees))
	responseStatus := ""
	for _, attendee := range event.Attendees {
		if attendee == nil {
			continue
		}
		attendees = append(attendees, attendee.Email)
		if attendee.Self {
			responseStatus = attendee.ResponseStatus
		}
	}

	return Event{
		ID:             event.Id,
		Summary:        event.Summary,
		Description:    event.Description,
		Start:          eventTime(event.Start),
		End:            eventTime(event.End),
		HTMLLink:       event.HtmlLink,
		MeetLink:       event.HangoutLink,
		Attendees:      attendees,
		ResponseStatus: responseStatus,
		Recurrence:     event.Recurrence,
	}
}

func ParseResponseStatus(value string) (ResponseStatus, error) {
	status := ResponseStatus(value)
	switch status {
	case ResponseAccepted, ResponseDeclined, ResponseTentative:
		return status, nil
	default:
		return "", errors.New("responseStatus must be accepted, declined, or tentative")
	}
}

func selfAttendee(event *calendarapi.Event) (*calendarapi.EventAttendee, error) {
	for _, attendee := range event.Attendees {
		if attendee != nil && attendee.Self {
			return attendee, nil
		}
	}
	return nil, errors.New("authenticated user is not an attendee of this event")
}

func conferenceRequestID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate conference request ID: %w", err)
	}
	return hex.EncodeToString(value), nil
}

func newConferenceData() (*calendarapi.ConferenceData, error) {
	requestID, err := conferenceRequestID()
	if err != nil {
		return nil, err
	}

	return &calendarapi.ConferenceData{
		CreateRequest: &calendarapi.CreateConferenceRequest{
			RequestId: requestID,
			ConferenceSolutionKey: &calendarapi.ConferenceSolutionKey{
				Type: "hangoutsMeet",
			},
		},
	}, nil
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
