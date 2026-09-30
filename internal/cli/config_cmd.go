package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/panoratech/twinbay-cli/internal/config"
	"github.com/panoratech/twinbay-cli/internal/flagutil"
	"github.com/panoratech/twinbay-cli/internal/output"
	"github.com/panoratech/twinbay-cli/internal/usage"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

type setting struct {
	key    string
	secret bool
	field  func(*config.Config) *string
}

var settings = []setting{
	{"access-token", true, func(c *config.Config) *string { return &c.Security.AccessToken }},
	{"organization-api-key", true, func(c *config.Config) *string { return &c.Security.OrganizationAPIKey }},
	{"output-format", false, func(c *config.Config) *string { return &c.OutputFormat }},
	{"timeout", false, func(c *config.Config) *string { return &c.Timeout }},
}

func lookupSetting(key string) (setting, error) {
	key = string(normalizeFlagName(nil, key))
	for _, s := range settings {
		if s.key == key {
			return s, nil
		}
	}
	keys := make([]string, len(settings))
	for i, s := range settings {
		keys[i] = s.key
	}
	return setting{}, flagutil.WithCLIValidation(fmt.Errorf("unknown setting %q; valid settings: %s", key, strings.Join(keys, ", ")))
}

func initConfigCmd(root *cobra.Command) {
	keys := make([]string, 0, len(settings)+1)
	for _, s := range settings {
		keys = append(keys, s.key)
	}
	keys = append(keys, "api-key")
	cmd := &cobra.Command{
		Use:   "config",
		Short: "List, set and unset CLI settings without prompts",
		Long: `List, set and unset CLI settings without prompts.

Secrets are stored in the OS keychain when available, otherwise in
~/.config/twinbay/config.yaml. Settings: ` + strings.Join(keys, ", ") + `.
For a guided form, use "twinbay configure".`,
		Example: "  twinbay config set api-key \"$TWINBAY_API_KEY\"\n  twinbay config set output-format json\n  twinbay config list -o json\n  twinbay config unset timeout",
		Args:    cobra.NoArgs,
		RunE:    func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List settings with their effective values and sources",
			Args:  cobra.NoArgs,
			RunE:  runConfigList,
		},
		&cobra.Command{
			Use:       "set <key> <value>",
			Short:     "Store a setting",
			Args:      exactArgs(2),
			ValidArgs: keys,
			RunE:      runConfigSet,
		},
		&cobra.Command{
			Use:       "unset <key>",
			Short:     "Remove a stored setting",
			Args:      exactArgs(1),
			ValidArgs: keys,
			RunE:      runConfigUnset,
		},
	)
	walkCommands(cmd, usage.MarkDynamic)
	root.AddCommand(cmd)
}

func runConfigList(cmd *cobra.Command, _ []string) error {
	type entry struct {
		Key    string `json:"key"`
		Value  string `json:"value"`
		Source string `json:"source"`
	}
	entries := make([]entry, 0, len(settings))
	for _, s := range settings {
		var value, source string
		if s.secret {
			value, source = config.ResolveSecurityCredential(cmd, s.key)
			value = maskSecret(value)
		} else if value = config.GetEnvValue(s.key); value != "" {
			source = "env"
		} else if value = config.GetConfigValue(s.key); value != "" {
			source = "config"
		} else {
			source = "unset"
		}
		entries = append(entries, entry{s.key, value, source})
	}
	if output.IsMachineMode(cmd) {
		return output.LocalResult(cmd, entries)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Config file: %s\n\n", config.GetConfigPath())
	for _, e := range entries {
		fmt.Fprintf(out, "%-22s [%-7s] %s\n", e.Key, e.Source, e.Value)
	}
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	s, err := lookupSetting(args[0])
	if err != nil {
		return err
	}
	value := args[1]
	switch s.key {
	case "output-format":
		if !slices.Contains(output.Formats, value) {
			return flagutil.WithCLIValidation(fmt.Errorf("invalid output-format %q; valid formats: %s", value, strings.Join(output.Formats, ", ")))
		}
	case "timeout":
		if d, err := time.ParseDuration(value); err != nil || d <= 0 {
			return flagutil.WithCLIValidation(fmt.Errorf("invalid timeout %q: use a positive duration such as 30s", value))
		}
	}
	if value == "" {
		return flagutil.WithCLIValidation(fmt.Errorf("value for %s cannot be empty; use \"config unset %s\"", s.key, s.key))
	}
	if dryRunLocalNoop(cmd, "config set changes local settings only (no API request); nothing was changed.") {
		return nil
	}
	cfg := loadedConfig()
	where := config.GetConfigPath()
	if s.secret {
		if config.StoreSecret(s.key, value, s.field(cfg)) == nil {
			where = "OS keychain"
		}
	} else {
		*s.field(cfg) = value
	}
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Set %s (%s)\n", s.key, where)
	return nil
}

func runConfigUnset(cmd *cobra.Command, args []string) error {
	s, err := lookupSetting(args[0])
	if err != nil {
		return err
	}
	if dryRunLocalNoop(cmd, "config unset changes local settings only (no API request); nothing was changed.") {
		return nil
	}
	cfg := loadedConfig()
	if s.secret && config.KeyringAvailable() {
		if err := config.DeleteKeyringValue(s.key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("clear %s from OS keychain: %w", s.key, err)
		}
	}
	*s.field(cfg) = ""
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Unset %s\n", s.key)
	return nil
}

func loadedConfig() *config.Config {
	if cfg := config.GetConfig(); cfg != nil {
		return cfg
	}
	return &config.Config{}
}
