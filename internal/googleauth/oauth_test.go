package googleauth

import "testing"

func TestValidateDesktopCredentials(t *testing.T) {
	desktop := []byte(`{"installed":{"client_id":"id"}}`)
	if err := validateDesktopCredentials(desktop); err != nil {
		t.Fatalf("desktop credentials rejected: %v", err)
	}
}

func TestValidateDesktopCredentialsRejectsWebClient(t *testing.T) {
	web := []byte(`{"web":{"client_id":"id"}}`)
	if err := validateDesktopCredentials(web); err == nil {
		t.Fatal("expected web credentials to be rejected")
	}
}
