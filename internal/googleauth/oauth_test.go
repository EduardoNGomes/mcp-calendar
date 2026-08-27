package googleauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

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

func TestIsAuthenticatedRefreshesExpiredToken(t *testing.T) {
	ctx := tokenContext(func(r *http.Request) (int, string) {
		if got := r.FormValue("refresh_token"); got != "refresh-token" {
			t.Fatalf("refresh_token = %q; want refresh-token", got)
		}
		return http.StatusOK, `{"access_token":"new-access-token","token_type":"Bearer","expires_in":3600}`
	})

	service, tokenPath := testService(t)
	if err := SaveToken(tokenPath, expiredToken()); err != nil {
		t.Fatal(err)
	}

	authenticated, err := service.IsAuthenticated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !authenticated {
		t.Fatal("expected refreshed token to be authenticated")
	}

	token, err := LoadToken(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "new-access-token" {
		t.Fatalf("saved access token = %q; want new-access-token", token.AccessToken)
	}
	if token.RefreshToken != "refresh-token" {
		t.Fatalf("saved refresh token = %q; want refresh-token", token.RefreshToken)
	}
}

func TestIsAuthenticatedRemovesRevokedToken(t *testing.T) {
	ctx := tokenContext(func(r *http.Request) (int, string) {
		return http.StatusBadRequest, `{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`
	})

	service, tokenPath := testService(t)
	if err := SaveToken(tokenPath, expiredToken()); err != nil {
		t.Fatal(err)
	}

	authenticated, err := service.IsAuthenticated(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if authenticated {
		t.Fatal("revoked token must not be authenticated")
	}
	if _, err := os.Stat(tokenPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("token file still exists after invalid_grant: %v", err)
	}
}

func TestClientRemovesTokenWhenRefreshIsRejected(t *testing.T) {
	ctx := tokenContext(func(r *http.Request) (int, string) {
		return http.StatusBadRequest, `{"error":"invalid_grant"}`
	})

	service, tokenPath := testService(t)
	if err := SaveToken(tokenPath, expiredToken()); err != nil {
		t.Fatal(err)
	}
	client, err := service.Client(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Get("https://example.invalid")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("client error = %v; want ErrNotAuthenticated", err)
	}
	if _, err := os.Stat(tokenPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("token file still exists after invalid_grant: %v", err)
	}
}

func TestIsAuthenticatedKeepsTokenAfterTransientRefreshFailure(t *testing.T) {
	ctx := tokenContext(func(r *http.Request) (int, string) {
		return http.StatusServiceUnavailable, "temporary failure"
	})

	service, tokenPath := testService(t)
	if err := SaveToken(tokenPath, expiredToken()); err != nil {
		t.Fatal(err)
	}

	if _, err := service.IsAuthenticated(ctx); err == nil {
		t.Fatal("expected transient refresh error")
	}
	if _, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("token file should be kept after transient failure: %v", err)
	}
}

func testService(t *testing.T) (*Service, string) {
	t.Helper()
	tokenPath := filepath.Join(t.TempDir(), "token.json")
	return &Service{
		config: &oauth2.Config{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL:  "https://oauth.example/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		tokenPath: tokenPath,
	}, tokenPath
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func tokenContext(response func(*http.Request) (int, string)) context.Context {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := response(r)
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})}
	return context.WithValue(context.Background(), oauth2.HTTPClient, client)
}

func expiredToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  "expired-access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	}
}
