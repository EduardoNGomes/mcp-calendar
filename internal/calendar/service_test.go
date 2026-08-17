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
			{Email: "guest@example.com", Self: true, ResponseStatus: "accepted"},
		},
	})

	if event.MeetLink != "https://meet.google.com/abc-defg-hij" {
		t.Fatalf("meet link = %q; want Google Meet URL", event.MeetLink)
	}
	if len(event.Attendees) != 1 || event.Attendees[0] != "guest@example.com" {
		t.Fatalf("attendees = %v; want guest@example.com", event.Attendees)
	}
	if event.ResponseStatus != "accepted" {
		t.Fatalf("response status = %q; want accepted", event.ResponseStatus)
	}
}

func TestParseResponseStatus(t *testing.T) {
	for _, value := range []string{"accepted", "declined", "tentative"} {
		status, err := ParseResponseStatus(value)
		if err != nil {
			t.Fatalf("ParseResponseStatus(%q): %v", value, err)
		}
		if string(status) != value {
			t.Fatalf("status = %q; want %q", status, value)
		}
	}
}

func TestParseResponseStatusRejectsInvalidValue(t *testing.T) {
	_, err := ParseResponseStatus("maybe")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestSelfAttendee(t *testing.T) {
	want := &calendarapi.EventAttendee{Email: "me@example.com", Self: true}
	event := &calendarapi.Event{
		Attendees: []*calendarapi.EventAttendee{
			{Email: "guest@example.com"},
			want,
		},
	}

	got, err := selfAttendee(event)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("attendee = %v; want %v", got, want)
	}
}

func TestSelfAttendeeRejectsOrganizerEvent(t *testing.T) {
	_, err := selfAttendee(&calendarapi.Event{})
	if err == nil {
		t.Fatal("expected an error")
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
