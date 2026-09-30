package authkit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDeviceAuthorizationPollsAndRotatesOnSlowDown(t *testing.T) {
	t.Parallel()
	var polls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/authorize/device" {
			assertForm(t, r, url.Values{"client_id": {"client_test"}})
			writeJSON(w, http.StatusOK, map[string]any{
				"device_code": "secret-device-code", "user_code": "ABCD-EFGH",
				"verification_uri": "https://auth.example/device", "verification_uri_complete": "https://auth.example/device?user_code=ABCD-EFGH",
				"expires_in": 60, "interval": 2,
			})
			return
		}
		polls++
		assertForm(t, r, url.Values{
			"grant_type": {deviceGrant}, "device_code": {"secret-device-code"}, "client_id": {"client_test"},
		})
		switch polls {
		case 1:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "authorization_pending"})
		case 2:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slow_down"})
		default:
			writeJSON(w, http.StatusOK, tokenPayload("access", "refresh", "org_test"))
		}
	}))
	defer server.Close()

	var sleeps []time.Duration
	client := &Client{
		ClientID: "client_test", BaseURL: server.URL, HTTPClient: server.Client(),
		Sleep: func(_ context.Context, duration time.Duration) error {
			sleeps = append(sleeps, duration)
			return nil
		},
	}
	device, err := client.AuthorizeDevice(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := client.Poll(context.Background(), device)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.RefreshToken != "refresh" || tokens.OrganizationID != "org_test" {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
	if want := []time.Duration{2 * time.Second, 3 * time.Second}; !reflect.DeepEqual(sleeps, want) {
		t.Fatalf("sleep intervals = %v, want %v", sleeps, want)
	}
}

func TestDeviceAuthorizationTerminalOutcomes(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		code string
		want string
	}{
		{"access_denied", "denied"},
		{"expired_token", "expired"},
		{"invalid_request", "invalid request"},
	} {
		t.Run(test.code, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": test.code})
			}))
			defer server.Close()
			client := &Client{ClientID: "client_test", BaseURL: server.URL, HTTPClient: server.Client(), Sleep: sleepContext}
			_, err := client.Poll(context.Background(), &DeviceAuthorization{DeviceCode: "device", ExpiresIn: 10, Interval: 1})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestDeviceAuthorizationCancellation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "authorization_pending"})
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		ClientID: "client_test", BaseURL: server.URL, HTTPClient: server.Client(),
		Sleep: func(context.Context, time.Duration) error { cancel(); return context.Canceled },
	}
	_, err := client.Poll(ctx, &DeviceAuthorization{DeviceCode: "device", ExpiresIn: 10, Interval: 1})
	if err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("error = %v, want cancellation", err)
	}
}

func TestManagerRefreshesAndStoresRotatedSession(t *testing.T) {
	t.Parallel()
	store := &memoryStore{session: &Session{
		AccessToken: "invalid-expired-token", RefreshToken: "old-refresh", OrganizationID: "org_current",
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertForm(t, r, url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {"old-refresh"}, "client_id": {"client_test"}, "organization_id": {"org_current"},
		})
		writeJSON(w, http.StatusOK, tokenPayload(testJWT(time.Now().Add(time.Hour), "session_new"), "new-refresh", "org_current"))
	}))
	defer server.Close()
	manager := &Manager{
		Client: &Client{ClientID: "client_test", BaseURL: server.URL, HTTPClient: server.Client()},
		Store:  store, Now: time.Now,
	}
	session, err := manager.Session(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.RefreshToken != "new-refresh" || store.session.RefreshToken != "new-refresh" || store.saves != 1 {
		t.Fatalf("rotated session was not stored: session=%#v store=%#v", session, store)
	}
}

func TestManagerDeletesInvalidGrant(t *testing.T) {
	t.Parallel()
	store := &memoryStore{session: &Session{AccessToken: "expired", RefreshToken: "invalid"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
	}))
	defer server.Close()
	manager := &Manager{Client: &Client{ClientID: "client_test", BaseURL: server.URL, HTTPClient: server.Client()}, Store: store}
	_, err := manager.Session(context.Background())
	if err == nil || !strings.Contains(err.Error(), "expired") || store.deletes != 1 {
		t.Fatalf("error=%v deletes=%d", err, store.deletes)
	}
}

func TestManagerSwitchesOrganizationAndStoresRotatedSession(t *testing.T) {
	t.Parallel()
	store := &memoryStore{session: &Session{
		AccessToken: testJWT(time.Now().Add(time.Hour), "session_old"), RefreshToken: "old-refresh", OrganizationID: "org_old",
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertForm(t, r, url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {"old-refresh"}, "client_id": {"client_test"}, "organization_id": {"org_new"},
		})
		writeJSON(w, http.StatusOK, tokenPayload(testJWT(time.Now().Add(time.Hour), "session_new"), "new-refresh", "org_new"))
	}))
	defer server.Close()
	manager := &Manager{
		Client: &Client{ClientID: "client_test", BaseURL: server.URL, HTTPClient: server.Client()},
		Store:  store,
	}
	session, err := manager.Switch(context.Background(), "org_new")
	if err != nil {
		t.Fatal(err)
	}
	if session.OrganizationID != "org_new" || session.RefreshToken != "new-refresh" || store.saves != 1 {
		t.Fatalf("switched session was not stored: session=%#v store=%#v", session, store)
	}
}

func TestManagerLogoutTerminatesRemoteSessionAndDeletesLocalSession(t *testing.T) {
	t.Parallel()
	store := &memoryStore{session: &Session{AccessToken: testJWT(time.Now().Add(time.Hour), "session_test"), RefreshToken: "refresh"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("session_id"); got != "session_test" {
			t.Errorf("session_id = %q", got)
		}
		w.Header().Set("Location", "https://auth.example/signed-out")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	manager := &Manager{Client: &Client{BaseURL: server.URL, HTTPClient: server.Client()}, Store: store}
	if err := manager.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.deletes != 1 || store.session != nil {
		t.Fatalf("local session was not deleted: %#v", store)
	}
}

func TestManagerLogoutDeletesUnreadableLocalSession(t *testing.T) {
	t.Parallel()
	store := &loadErrorStore{memoryStore: memoryStore{session: &Session{AccessToken: "corrupt", RefreshToken: "corrupt"}}}
	manager := &Manager{Client: &Client{}, Store: store}
	err := manager.Logout(context.Background())
	if err == nil || !strings.Contains(err.Error(), "local session was cleared") {
		t.Fatalf("error = %v, want local-cleanup outcome", err)
	}
	if store.deletes != 1 || store.session != nil {
		t.Fatalf("unreadable local session was not deleted: %#v", store)
	}
}

type memoryStore struct {
	session *Session
	saves   int
	deletes int
}

type loadErrorStore struct{ memoryStore }

func (*loadErrorStore) Load() (*Session, error) {
	return nil, errors.New("corrupt keychain record")
}

func (s *memoryStore) Load() (*Session, error) {
	if s.session == nil {
		return nil, ErrNoSession
	}
	copy := *s.session
	return &copy, nil
}

func (s *memoryStore) Save(session *Session) error {
	copy := *session
	s.session = &copy
	s.saves++
	return nil
}

func (s *memoryStore) Delete() error {
	s.session = nil
	s.deletes++
	return nil
}

func tokenPayload(access, refresh, organization string) map[string]any {
	return map[string]any{
		"user":         map[string]string{"id": "user_test", "email": "person@example.com"},
		"access_token": access, "refresh_token": refresh, "organization_id": organization,
	}
}

func testJWT(expiry time.Time, sessionID string) string {
	payload, _ := json.Marshal(map[string]any{"exp": expiry.Unix(), "sid": sessionID})
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func assertForm(t *testing.T, r *http.Request, want url.Values) {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.PostForm, want) {
		t.Errorf("form = %v, want %v", r.PostForm, want)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(fmt.Sprintf("encode test response: %v", err))
	}
}
