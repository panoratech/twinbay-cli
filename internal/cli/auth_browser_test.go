package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/panoratech/twinbay-cli/internal/authkit"
	"github.com/panoratech/twinbay-cli/internal/config"
	"github.com/panoratech/twinbay-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func TestBrowserLoginNoBrowserShowsFallbackAndStoresSession(t *testing.T) {
	setupAuthCommandTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/authorize/device":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code": "device-secret", "user_code": "ABCD-EFGH",
				"verification_uri":          "https://auth.example/device",
				"verification_uri_complete": "https://auth.example/device?user_code=ABCD-EFGH",
				"expires_in":                30, "interval": 1,
			})
		case "/authenticate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"user":         map[string]string{"id": "user_test", "email": "person@example.com"},
				"access_token": "access-secret", "refresh_token": "refresh-secret", "organization_id": "org_test",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &commandStore{}
	originalManager := newAuthManager
	newAuthManager = func() *authkit.Manager {
		return &authkit.Manager{
			Client: &authkit.Client{
				ClientID: "client_test", BaseURL: server.URL, HTTPClient: server.Client(),
				Sleep: func(context.Context, time.Duration) error { return nil },
			},
			Store: store,
		}
	}
	t.Cleanup(func() { newAuthManager = originalManager })

	cmd := browserLoginTestCommand(t)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := runBrowserLoginCmd(cmd, nil); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	for _, expected := range []string{"https://auth.example/device", "ABCD-EFGH", "person@example.com", "org_test"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("output %q does not contain %q", got, expected)
		}
	}
	combinedOutput := got + stderr.String()
	if strings.Contains(combinedOutput, "device-secret") || strings.Contains(combinedOutput, "access-secret") || strings.Contains(combinedOutput, "refresh-secret") {
		t.Fatalf("output leaked a credential: %q", combinedOutput)
	}
	if store.session == nil || store.session.RefreshToken != "refresh-secret" {
		t.Fatalf("session was not stored: %#v", store.session)
	}
}

func TestBrowserLoginRejectsNoninteractiveWithoutCredential(t *testing.T) {
	setupAuthCommandTest(t)
	cmd := browserLoginTestCommand(t)
	if err := cmd.Root().PersistentFlags().Set("no-interactive", "true"); err != nil {
		t.Fatal(err)
	}
	err := runBrowserLoginCmd(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "browser login is interactive") {
		t.Fatalf("error = %v", err)
	}
}

func setupAuthCommandTest(t *testing.T) {
	t.Helper()
	config.Reset()
	config.ResetKeyring()
	output.ResetAgentMode()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Init("twinbay", "CLI_TWINBAY"); err != nil {
		t.Fatal(err)
	}
	config.SetKeyringBackend(availableKeyring{})
	t.Cleanup(func() {
		config.Reset()
		config.ResetKeyring()
		output.ResetAgentMode()
	})
}

func browserLoginTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "twinbay"}
	root.PersistentFlags().Bool("interactive", true, "")
	root.PersistentFlags().Bool("no-interactive", false, "")
	root.PersistentFlags().Bool("agent-mode", false, "")
	root.PersistentFlags().Bool("dry-run", false, "")
	root.PersistentFlags().String("output-format", "pretty", "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().String("organization-api-key", "", "")
	root.PersistentFlags().String("access-token", "", "")
	cmd := &cobra.Command{Use: "login"}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("no-browser", true, "")
	root.AddCommand(cmd)
	if err := root.PersistentFlags().Set("interactive", "true"); err != nil {
		t.Fatal(err)
	}
	if err := root.PersistentFlags().Set("agent-mode", "false"); err != nil {
		t.Fatal(err)
	}
	return cmd
}

type commandStore struct{ session *authkit.Session }

func (s *commandStore) Load() (*authkit.Session, error) {
	if s.session == nil {
		return nil, authkit.ErrNoSession
	}
	return s.session, nil
}
func (s *commandStore) Save(session *authkit.Session) error { s.session = session; return nil }
func (s *commandStore) Delete() error                       { s.session = nil; return nil }

type availableKeyring struct{}

func (availableKeyring) Get(string, string) (string, error) { return "", keyring.ErrNotFound }
func (availableKeyring) Set(string, string, string) error   { return nil }
func (availableKeyring) Delete(string, string) error        { return nil }
