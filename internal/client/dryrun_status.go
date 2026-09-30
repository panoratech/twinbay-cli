package client

import (
	"net/http"

	"github.com/spf13/cobra"
)

// dryRunSuccessStatus lists operations whose only success response is not 200.
// Generated response dispatch rejects any other status, so a dry-run must
// answer with the documented one. TestDryRunSuccessStatusMatchesSpec keeps
// this in sync with .speakeasy/out.openapi.yaml.
var dryRunSuccessStatus = map[string]int{
	"create_organization":    http.StatusCreated,
	"create_api_key":         http.StatusCreated,
	"create_scenario":        http.StatusCreated,
	"create_test":            http.StatusCreated,
	"create_test_run":        http.StatusCreated,
	"create_environment":     http.StatusAccepted,
	"start_environment_twin": http.StatusAccepted,
	"stop_environment_twin":  http.StatusAccepted,
	"create_seed":            http.StatusAccepted,
	"create_evaluator":       http.StatusAccepted,
	"create_evaluation":      http.StatusAccepted,
	"create_export":          http.StatusAccepted,
	"revoke_api_key":         http.StatusNoContent,
	"delete_scenario":        http.StatusNoContent,
	"delete_seed":            http.StatusNoContent,
}

func dryRunStatus(cmd *cobra.Command) int {
	if cmd != nil {
		if status, ok := dryRunSuccessStatus[cmd.Annotations["speakeasy_operation"]]; ok {
			return status
		}
	}
	return http.StatusOK
}
