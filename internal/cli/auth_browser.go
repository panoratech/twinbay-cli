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
	"github.com/panoratech/twinbay-cli/internal/output"
	"github.com/panoratech/twinbay-cli/internal/sdk"
	"github.com/panoratech/twinbay-cli/internal/sdk/models/components"
	"github.com/panoratech/twinbay-cli/internal/usage"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var newAuthManager = authkit.NewManager

// installBrowserAuthCommands keeps the generated auth command intact while
// replacing only the runtime handlers owned by the AuthKit integration.
func installBrowserAuthCommands(authCmd *cobra.Command) {
	if authCmd == nil {
		return
	}
	authCmd.Long = `Manage authentication credentials for twinbay.

Subcommands:
  login   - Sign in through a browser
  switch  - Change the active organization
  whoami  - Display current authentication status
  logout  - End the browser session and clear stored credentials`

	for _, sub := range authCmd.Commands() {
		switch sub.Name() {
		case "login":
			sub.Short = "Sign in through a browser"
			sub.Long = `Sign in to Twinbay through WorkOS AuthKit's device authorization flow.

The CLI displays a verification URL and code, opens the URL when possible, and
stores the resulting renewable session in the OS keychain. Use --no-browser on
headless hosts. An explicit --organization-api-key or --access-token is stored
without opening a browser.`
			sub.RunE = runBrowserLoginCmd
			if sub.Flags().Lookup("no-browser") == nil {
				sub.Flags().Bool("no-browser", false, "Display the verification URL and code without opening a browser")
			}
			usage.MarkDynamic(sub)
		case "whoami":
			sub.Short = "Display the current user, organization, and credential sources"
			sub.Long = `Display the stored browser-session identity and active organization,
along with any directly configured credential sources.`
			sub.RunE = runBrowserWhoamiCmd
			usage.MarkDynamic(sub)
		case "logout":
			sub.Short = "End the browser session and clear stored credentials"
			sub.Long = `End the WorkOS browser session and clear all stored authentication credentials
from both the OS keychain and config file.`
			sub.RunE = runBrowserLogoutCmd
			usage.MarkDynamic(sub)
		}
	}

	for _, sub := range authCmd.Commands() {
		if sub.Name() == "switch" {
			return
		}
	}
	switchCmd := &cobra.Command{
		Use:   "switch [organization]",
		Short: "Change the active organization",
		Long:  "Refresh the stored AuthKit session into another organization. The argument may be a Twinbay organization ID, WorkOS organization ID, or exact name.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runAuthSwitchCmd,
	}
	usage.MarkDynamic(switchCmd)
	authCmd.AddCommand(switchCmd)
}

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

func runBrowserWhoamiCmd(cmd *cobra.Command, _ []string) error {
	if usage.UsageRequested(cmd) {
		return usage.EmitSchema(cmd, cmd.OutOrStdout())
	}

	manager := newAuthManager()
	session, sessionErr := manager.Session(cmd.Context())
	if sessionErr != nil && !errors.Is(sessionErr, authkit.ErrNoSession) {
		return sessionErr
	}

	if output.IsMachineMode(cmd) {
		info := map[string]any{
			"config_file":        config.GetConfigPath(),
			"environment_prefix": "CLI_TWINBAY_",
		}
		credentials := map[string]any{}
		for _, name := range []string{"access-token", "organization-api-key"} {
			value, source := config.ResolveSecurityCredential(cmd, name)
			credentials[name] = map[string]any{"source": source, "value": maskSecret(value)}
		}
		info["credentials"] = credentials
		if session != nil {
			info["browser_session"] = map[string]any{
				"source": "keyring", "user_id": session.User.ID, "email": session.User.Email,
				"organization_id": session.OrganizationID,
			}
		} else {
			info["browser_session"] = map[string]any{"source": "unset"}
		}
		return output.LocalResult(cmd, info)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Configuration")
	fmt.Fprintln(out, "=============")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Config file: %s\n", config.GetConfigPath())
	fmt.Fprintln(out, "Environment prefix: CLI_TWINBAY_")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Credentials:")
	if session != nil {
		fmt.Fprintf(out, "  browser session             [keyring] %s", session.User.Email)
		if session.OrganizationID != "" {
			fmt.Fprintf(out, " (organization %s)", session.OrganizationID)
		}
		fmt.Fprintln(out)
	} else {
		fmt.Fprintln(out, "  browser session             [unset  ]")
	}
	for _, name := range []string{"access-token", "organization-api-key"} {
		value, source := config.ResolveSecurityCredential(cmd, name)
		fmt.Fprintf(out, "  --%-25s [%-7s] %s\n", name, source, maskSecret(value))
	}
	return nil
}
