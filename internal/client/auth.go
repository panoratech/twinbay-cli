package client

import (
	"context"
	"errors"

	"github.com/panoratech/twinbay-cli/internal/authkit"
	"github.com/panoratech/twinbay-cli/internal/config"
	"github.com/panoratech/twinbay-cli/internal/flagutil"
	"github.com/panoratech/twinbay-cli/internal/sdk/models/components"
	"github.com/spf13/cobra"
)

func buildGlobalSecuritySource(cmd *cobra.Command, allowedSecurityFields []string) func(context.Context) (components.Security, error) {
	manager := authkit.NewManager()
	return func(ctx context.Context) (components.Security, error) {
		// API keys are intentionally checked first: CI and other explicit key
		// callers must never touch the keychain or trigger browser-session work.
		if fieldAllowed("OrganizationAPIKey", allowedSecurityFields) {
			if value, _ := config.ResolveRequestSecurityCredential(cmd, "organization-api-key"); value != "" {
				return components.Security{OrganizationAPIKey: &value}, nil
			}
		}
		if fieldAllowed("AccessToken", allowedSecurityFields) {
			if value, _ := config.ResolveRequestSecurityCredential(cmd, "access-token"); value != "" {
				return components.Security{AccessToken: &value}, nil
			}
			dryRun, _ := flagutil.GetBoolFlag(cmd, "dry-run")
			if dryRun {
				return components.Security{}, nil
			}
			value, err := manager.AccessToken(ctx)
			if errors.Is(err, authkit.ErrNoSession) {
				return components.Security{}, nil
			}
			if err != nil {
				return components.Security{}, err
			}
			return components.Security{AccessToken: &value}, nil
		}
		return components.Security{}, nil
	}
}

func fieldAllowed(field string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if candidate == field {
			return true
		}
	}
	return false
}
