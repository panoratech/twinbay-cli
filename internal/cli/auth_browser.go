package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/panoratech/twinbay-cli/internal/authkit"
	"github.com/panoratech/twinbay-cli/internal/config"
	"github.com/panoratech/twinbay-cli/internal/flagutil"
	"github.com/panoratech/twinbay-cli/internal/interactive"
	"github.com/panoratech/twinbay-cli/internal/sdk"
	"github.com/panoratech/twinbay-cli/internal/sdk/models/components"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var newAuthManager = authkit.NewManager

func runBrowserLoginCmd(cmd *cobra.Command, _ []string) error {
	if dryRunLocalNoop(cmd, "auth login changes local credentials only (no Twinbay API request); nothing was changed.") {
		return nil
	}
	if handled, err := storeExplicitLoginCredential(cmd); handled {
		return err
	}
	if interactive.Resolve(cmd).FormMode() == interactive.FormOff {
		return flagutil.WithCLIValidation(errors.New("browser login is interactive; remove --no-interactive or provide --organization-api-key"))
	}
	if !config.KeyringAvailable() {
		return errors.New("OS keychain is unavailable; browser sessions cannot be stored securely")
	}

	manager := newAuthManager()
	device, err := manager.Client.AuthorizeDevice(cmd.Context())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Open %s\n", device.VerificationURI)
	fmt.Fprintf(out, "Enter code: %s\n", device.UserCode)

	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	browserURL := device.VerificationURIComplete
	if browserURL == "" {
		browserURL = device.VerificationURI
	}
	if !noBrowser {
		name, args := authkit.BrowserCommand(browserURL)
		if err := exec.CommandContext(cmd.Context(), name, args...).Start(); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Could not open a browser; use the URL and code shown above.")
		}
	}
	fmt.Fprintln(out, "Waiting for approval…")
	tokens, err := manager.Client.Poll(cmd.Context(), device)
	if err != nil {
		return err
	}
	if err := manager.Save(tokens); err != nil {
		return err
	}
	fmt.Fprintf(out, "Signed in as %s", tokens.User.Email)
	if tokens.OrganizationID != "" {
		fmt.Fprintf(out, " (organization %s)", tokens.OrganizationID)
	}
	fmt.Fprintln(out)
	return nil
}

func storeExplicitLoginCredential(cmd *cobra.Command) (bool, error) {
	for _, name := range []string{"organization-api-key", "access-token"} {
		flag := cmd.Flags().Lookup(name)
		if flag == nil || !flag.Changed {
			continue
		}
		value, _ := cmd.Flags().GetString(name)
		if value == "" {
			return true, flagutil.WithCLIValidation(fmt.Errorf("--%s cannot be empty", name))
		}
		cfg := config.GetConfig()
		if cfg == nil {
			cfg = &config.Config{}
		}
		var fallback *string
		if name == "organization-api-key" {
			fallback = &cfg.Security.OrganizationAPIKey
		} else {
			fallback = &cfg.Security.AccessToken
		}
		keyring := config.StoreSecret(name, value, fallback) == nil
		if err := config.SaveConfig(cfg); err != nil {
			return true, err
		}
		if keyring {
			fmt.Fprintln(cmd.OutOrStdout(), "Credential stored in OS keychain.")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Credential stored in %s.\n", config.GetConfigPath())
		}
		return true, nil
	}
	return false, nil
}

func runAuthSwitchCmd(cmd *cobra.Command, args []string) error {
	manager := newAuthManager()
	session, err := manager.Session(cmd.Context())
	if err != nil {
		return err
	}
	security := components.Security{AccessToken: &session.AccessToken}
	opts := []sdk.SDKOption{sdk.WithSecurity(security)}
	if serverURL, _ := flagutil.GetStringFlag(cmd, "server-url"); serverURL != "" {
		if err := flagutil.ValidateServerURL(serverURL); err != nil {
			return err
		}
		opts = append(opts, sdk.WithServerURL(serverURL))
	}
	api := sdk.New(opts...)
	response, err := api.Users.Retrieve(cmd.Context())
	if err != nil {
		return fmt.Errorf("list organizations: %w", err)
	}
	if response.MeResponse == nil {
		return errors.New("list organizations: Twinbay returned an empty response")
	}
	memberships := response.MeResponse.Memberships
	if len(memberships) == 0 {
		return errors.New("this account does not belong to an organization")
	}

	selected := ""
	if len(args) == 1 {
		selected = args[0]
	} else {
		if interactive.Resolve(cmd).FormMode() == interactive.FormOff {
			return flagutil.WithCLIValidation(errors.New("organization is required with --no-interactive"))
		}
		options := make([]huh.Option[string], 0, len(memberships))
		for _, membership := range memberships {
			org := membership.Organization
			options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", org.Name, org.ID), org.ID))
		}
		if err := huh.NewSelect[string]().Title("Active organization").Options(options...).Value(&selected).Run(); err != nil {
			return fmt.Errorf("select organization: %w", err)
		}
	}

	var chosen *components.OrganizationResponse
	for i := range memberships {
		org := &memberships[i].Organization
		if selected == org.ID || (org.WorkosOrganizationID != nil && selected == *org.WorkosOrganizationID) || strings.EqualFold(selected, org.Name) {
			if chosen != nil {
				return flagutil.WithCLIValidation(fmt.Errorf("organization name %q is ambiguous; use an ID", selected))
			}
			chosen = org
		}
	}
	if chosen == nil {
		return flagutil.WithCLIValidation(fmt.Errorf("organization %q was not found in this account", selected))
	}
	if chosen.WorkosOrganizationID == nil || *chosen.WorkosOrganizationID == "" {
		return errors.New("selected organization has no WorkOS organization ID")
	}
	if _, err := manager.Switch(cmd.Context(), *chosen.WorkosOrganizationID); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Active organization: %s (%s)\n", chosen.Name, chosen.ID)
	return nil
}

func runBrowserLogoutCmd(cmd *cobra.Command, _ []string) error {
	if dryRunLocalNoop(cmd, "auth logout removes local credentials only (no Twinbay API request); nothing was changed.") {
		return nil
	}
	manager := newAuthManager()
	remoteErr := manager.Logout(cmd.Context())

	cfg := config.GetConfig()
	if cfg == nil {
		cfg = &config.Config{}
	}
	if config.KeyringAvailable() {
		for _, key := range []string{"access-token", "organization-api-key"} {
			if err := config.DeleteKeyringValue(key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
				return fmt.Errorf("clear %s: %w", key, err)
			}
		}
	}
	cfg.Security.AccessToken = ""
	cfg.Security.OrganizationAPIKey = ""
	if err := config.SaveConfig(cfg); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Signed out and cleared stored credentials.")
	return remoteErr
}
