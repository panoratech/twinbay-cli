package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/panoratech/twinbay-cli/internal/client"
	"github.com/panoratech/twinbay-cli/internal/flagutil"
	"github.com/panoratech/twinbay-cli/internal/output"
	"github.com/panoratech/twinbay-cli/internal/sdk"
	"github.com/panoratech/twinbay-cli/internal/sdk/models/sdkerrors"
	"github.com/panoratech/twinbay-cli/internal/usage"
	"github.com/spf13/cobra"
)

// initRawCmds adds get/post/put/patch/delete for API paths without a
// dedicated command, mirroring `stripe get /v1/...`.
func initRawCmds(root *cobra.Command) {
	examples := map[string]string{
		http.MethodGet:    "  twinbay get /environments -f limit=5\n  twinbay get /tests/7ef34f0b-0ae6-4e26-ae92-f49d239d4d7d/versions",
		http.MethodPost:   "  twinbay post /scenarios -f name=checkout -f 'twins:=[\"stripe\"]'\n  twinbay post /scenarios --body @scenario.json",
		http.MethodPut:    "  twinbay put /twins/<twin-id>/records/customers/cus_123 -f 'fields:={\"email\":\"a@example.com\"}'",
		http.MethodPatch:  "  twinbay patch /organizations/current -f name=Acme",
		http.MethodDelete: "  twinbay delete /scenarios/0d7e9c1a-3b1f-4c55-9b0e-7f3a2c1d9e44 --confirm",
	}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		method := method
		name := strings.ToLower(method)
		cmd := &cobra.Command{
			Use:     name + " <path>",
			Short:   "Send a " + method + " request to an API path",
			Long:    "Send a " + method + " request to a Twinbay API path, using the same authentication, server, header, timeout and output handling as every other command.",
			Example: examples[method],
			Args:    exactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runRawCmd(cmd, method, args[0])
			},
		}
		if method == http.MethodGet || method == http.MethodDelete {
			cmd.Flags().StringArrayP("field", "f", nil, "Query parameter as key=value (repeatable)")
		} else {
			cmd.Flags().StringArrayP("field", "f", nil, "Body field as key=value (string) or key:=<json> (repeatable)")
			cmd.Flags().String("body", "", "Request body as JSON; @path reads a file, @- reads stdin")
		}
		usage.MarkDynamic(cmd)
		root.AddCommand(cmd)
	}
}

func runRawCmd(cmd *cobra.Command, method, path string) error {
	if !strings.HasPrefix(path, "/") {
		return flagutil.WithCLIValidation(fmt.Errorf("path %q must start with /", path))
	}
	if err := output.ValidateGlobalServerIndex(cmd, len(sdk.ServerList)); err != nil {
		return err
	}
	data, _ := cmd.Flags().GetStringArray("field")
	var body []byte
	if method == http.MethodGet || method == http.MethodDelete {
		query := url.Values{}
		for _, pair := range data {
			key, value, ok := strings.Cut(pair, "=")
			if !ok {
				return flagutil.WithCLIValidation(fmt.Errorf("invalid --field %q: expected key=value", pair))
			}
			query.Add(key, value)
		}
		if len(query) > 0 {
			separator := "?"
			if strings.Contains(path, "?") {
				separator = "&"
			}
			path += separator + query.Encode()
		}
	} else {
		var err error
		if body, err = rawBody(cmd, data); err != nil {
			return err
		}
	}

	res, err := client.Raw(cmd, method, path, body)
	if err != nil {
		return output.Error(cmd, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if res.StatusCode >= 400 {
		return output.Error(cmd, sdkerrors.NewSDKDefaultError("API error occurred", res.StatusCode, string(raw), res))
	}
	if flagutil.DidDryRunRequest(cmd) || len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		_, err = cmd.OutOrStdout().Write(raw)
		return err
	}
	return output.Result(cmd, value)
}

// rawBody merges --body with -f fields; -f wins on conflicting keys.
func rawBody(cmd *cobra.Command, data []string) ([]byte, error) {
	text, _ := cmd.Flags().GetString("body")
	text, err := flagutil.ResolveBodyFlagValue(cmd, "body", text)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		if text == "" {
			return nil, nil
		}
		if !json.Valid([]byte(text)) {
			return nil, flagutil.WithCLIValidation(fmt.Errorf("--body is not valid JSON"))
		}
		return []byte(text), nil
	}
	fields := map[string]any{}
	if text != "" {
		if err := json.Unmarshal([]byte(text), &fields); err != nil {
			return nil, flagutil.WithCLIValidation(fmt.Errorf("--body must be a JSON object when combined with --field: %w", err))
		}
	}
	for _, pair := range data {
		if key, value, ok := strings.Cut(pair, ":="); ok && !strings.Contains(key, "=") {
			var decoded any
			if err := json.Unmarshal([]byte(value), &decoded); err != nil {
				return nil, flagutil.WithCLIValidation(fmt.Errorf("invalid --field %q: %s is not valid JSON", pair, value))
			}
			fields[key] = decoded
			continue
		}
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, flagutil.WithCLIValidation(fmt.Errorf("invalid --field %q: expected key=value or key:=<json>", pair))
		}
		fields[key] = value
	}
	return json.Marshal(fields)
}
