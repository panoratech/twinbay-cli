package client

import (
	"context"
	"testing"

	"github.com/panoratech/twinbay-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func TestAPIKeyEnvironmentTakesPrecedenceWithoutKeyringRead(t *testing.T) {
	config.Reset()
	config.ResetKeyring()
	t.Cleanup(func() {
		config.Reset()
		config.ResetKeyring()
	})
	t.Setenv("CLI_TWINBAY_ORGANIZATION_API_KEY", "api-key")
	if err := config.Init("twinbay", "CLI_TWINBAY"); err != nil {
		t.Fatal(err)
	}
	backend := &countingKeyring{}
	config.SetKeyringBackend(backend)
	cmd := authCommand(t)
	security, err := buildGlobalSecuritySource(cmd, nil)(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if security.OrganizationAPIKey == nil || *security.OrganizationAPIKey != "api-key" {
		t.Fatalf("organization API key was not selected: %#v", security)
	}
	if security.AccessToken != nil {
		t.Fatalf("access token must not be sent with an API key: %#v", security)
	}
	if backend.gets != 0 {
		t.Fatalf("keyring reads = %d, want 0", backend.gets)
	}
}

func TestExplicitAPIKeyFlagWinsOverExplicitAccessToken(t *testing.T) {
	config.Reset()
	config.ResetKeyring()
	t.Cleanup(func() {
		config.Reset()
		config.ResetKeyring()
	})
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Init("twinbay", "CLI_TWINBAY"); err != nil {
		t.Fatal(err)
	}
	config.SetKeyringBackend(&countingKeyring{})
	cmd := authCommand(t)
	if err := cmd.Flags().Set("organization-api-key", "api-key"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("access-token", "bearer-token"); err != nil {
		t.Fatal(err)
	}
	security, err := buildGlobalSecuritySource(cmd, nil)(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if security.OrganizationAPIKey == nil || security.AccessToken != nil {
		t.Fatalf("expected only API key security, got %#v", security)
	}
}

func authCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("organization-api-key", "", "")
	cmd.Flags().String("access-token", "", "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd
}

type countingKeyring struct{ gets int }

func (k *countingKeyring) Get(string, string) (string, error) {
	k.gets++
	return "", keyring.ErrNotFound
}
func (*countingKeyring) Set(string, string, string) error { return nil }
func (*countingKeyring) Delete(string, string) error      { return nil }
