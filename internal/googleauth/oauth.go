package googleauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendarapi "google.golang.org/api/calendar/v3"
)

var ErrNotAuthenticated = errors.New("Google Calendar is not authenticated")

type Authorization struct {
	Authenticated    bool
	AuthorizationURL string
	BrowserOpened    bool
}

type Service struct {
	config    *oauth2.Config
	tokenPath string

	mu       sync.Mutex
	loginURL string
}

func New(credentialsPath, tokenPath string) (*Service, error) {
	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read Google credentials: %w", err)
	}
	if err := validateDesktopCredentials(credentials); err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(credentials, calendarapi.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("parse Google credentials: %w", err)
	}

	return &Service{config: config, tokenPath: tokenPath}, nil
}

func validateDesktopCredentials(credentials []byte) error {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(credentials, &document); err != nil {
		return fmt.Errorf("parse Google credentials: %w", err)
	}
	if len(document["installed"]) == 0 {
		return errors.New("Google credentials must be an OAuth Desktop app client")
	}
	return nil
}

func (s *Service) IsAuthenticated() bool {
	token, err := LoadToken(s.tokenPath)
	return err == nil && (token.Valid() || token.RefreshToken != "")
}

func (s *Service) Client(ctx context.Context) (*http.Client, error) {
	token, err := LoadToken(s.tokenPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotAuthenticated
		}
		return nil, err
	}

	return s.config.Client(ctx, token), nil
}

func (s *Service) StartAuthorization() (Authorization, error) {
	if s.IsAuthenticated() {
		return Authorization{Authenticated: true}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loginURL != "" {
		return Authorization{AuthorizationURL: s.loginURL}, nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Authorization{}, fmt.Errorf("start OAuth callback listener: %w", err)
	}

	config := *s.config
	config.RedirectURL = "http://" + listener.Addr().String() + "/callback"

	state, err := randomState()
	if err != nil {
		_ = listener.Close()
		return Authorization{}, err
	}

	s.loginURL = config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	server.Handler = s.callbackHandler(&config, state, server)

	go s.serveCallback(server, listener)
	opened := openBrowser(s.loginURL)

	return Authorization{
		AuthorizationURL: s.loginURL,
		BrowserOpened:    opened,
	}, nil
}

func (s *Service) callbackHandler(
	config *oauth2.Config,
	state string,
	server *http.Server,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid OAuth state", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			return
		}

		token, err := config.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "failed to exchange authorization code", http.StatusBadGateway)
			return
		}

		if err := SaveToken(s.tokenPath, token); err != nil {
			http.Error(w, "failed to save OAuth token", http.StatusInternalServerError)
			return
		}

		_, _ = w.Write([]byte("Autorização concluída. Você pode fechar esta aba."))
		go func() {
			_ = server.Shutdown(context.Background())
		}()
	})

	return mux
}

func (s *Service) serveCallback(server *http.Server, listener net.Listener) {
	timer := time.AfterFunc(5*time.Minute, func() {
		_ = server.Shutdown(context.Background())
	})

	err := server.Serve(listener)
	timer.Stop()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("OAuth callback server failed: %v", err)
	}

	s.mu.Lock()
	s.loginURL = ""
	s.mu.Unlock()
}

func randomState() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate OAuth state: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func openBrowser(url string) bool {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}

	return command.Start() == nil
}
