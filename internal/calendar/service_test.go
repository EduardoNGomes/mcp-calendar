package calendar

import (
	"testing"
	"time"

	calendarapi "google.golang.org/api/calendar/v3"
)

func TestAPIEventIncludesAttendees(t *testing.T) {
	start := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	input := CreateEvent{
		Summary:   "Project sync",
		Start:     start,
		End:       start.Add(30 * time.Minute),
		TimeZone:  "America/Sao_Paulo",
		Attendees: []string{"guest@example.com"},
	}

	event := apiEvent(input)
	if len(event.Attendees) != 1 || event.Attendees[0].Email != "guest@example.com" {
		t.Fatalf("attendees = %v; want guest@example.com", event.Attendees)
	}
}

func TestEventFromAPIIncludesMeetingDetails(t *testing.T) {
	event := eventFromAPI(&calendarapi.Event{
		Id:          "event-id",
		HangoutLink: "https://meet.google.com/abc-defg-hij",
		Attendees: []*calendarapi.EventAttendee{
			{Email: "guest@example.com"},
		},
	})

	if event.MeetLink != "https://meet.google.com/abc-defg-hij" {
		t.Fatalf("meet link = %q; want Google Meet URL", event.MeetLink)
	}
	if len(event.Attendees) != 1 || event.Attendees[0] != "guest@example.com" {
		t.Fatalf("attendees = %v; want guest@example.com", event.Attendees)
	}
}

func TestNewConferenceDataUsesUniqueGoogleMeetRequest(t *testing.T) {
	first, err := newConferenceData()
	if err != nil {
		t.Fatal(err)
	}
	second, err := newConferenceData()
	if err != nil {
		t.Fatal(err)
	}

	if first.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
		t.Fatalf("conference type = %q; want hangoutsMeet", first.CreateRequest.ConferenceSolutionKey.Type)
	}
	if first.CreateRequest.RequestId == second.CreateRequest.RequestId {
		t.Fatal("conference request IDs must be unique")
	}
}
