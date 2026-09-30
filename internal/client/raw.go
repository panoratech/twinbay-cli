package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/panoratech/twinbay-cli/internal/flagutil"
	"github.com/panoratech/twinbay-cli/internal/sdk"
	"github.com/panoratech/twinbay-cli/internal/testclient"
	"github.com/spf13/cobra"
)

// Raw sends one request to an API path with the server, authentication,
// header, timeout and diagnostics handling that generated commands use.
func Raw(cmd *cobra.Command, method, path string, body []byte) (*http.Response, error) {
	base := sdk.ServerList[0]
	if serverURL, _ := flagutil.GetStringFlag(cmd, "server-url"); serverURL != "" {
		if err := flagutil.ValidateServerURL(serverURL); err != nil {
			return nil, err
		}
		base = serverURL
	} else if serverFlag, _ := flagutil.GetStringFlag(cmd, "server"); serverFlag != "" {
		if idx, err := strconv.Atoi(serverFlag); err == nil && idx >= 0 && idx < len(sdk.ServerList) {
			base = sdk.ServerList[idx]
		}
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(cmd.Context(), method, strings.TrimRight(base, "/")+path, reader)
	if err != nil {
		return nil, flagutil.WithCLIValidation(err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	headers, _ := flagutil.GetStringArrayFlag(cmd, "header")
	for _, h := range headers {
		k, v, ok := strings.Cut(h, ":")
		if !ok {
			return nil, flagutil.WithCLIValidation(fmt.Errorf("invalid header format %q: expected \"Key: Value\"", h))
		}
		req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}

	security, err := buildGlobalSecuritySource(cmd, nil)(req.Context())
	if err != nil {
		return nil, err
	}
	if token := security.AccessToken; token != nil && *token != "" {
		value := *token
		if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
			value = "Bearer " + value
		}
		req.Header.Set("Authorization", value)
	}
	if key := security.OrganizationAPIKey; key != nil && *key != "" {
		req.Header.Set("X-API-Key", *key)
	}

	httpClient := &http.Client{Transport: newPhaseBoundedTransport(cmd)}
	if timeoutStr := resolveStringFlag(cmd, "timeout"); timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, flagutil.WithCLIValidation(fmt.Errorf("invalid --timeout value %q: %w", timeoutStr, err))
		}
		httpClient.Timeout = timeout
	}
	var client HTTPClient = httpClient
	if testClient := testclient.NewTestHTTPClient(); testClient != nil {
		client = testClient
	}
	return WrapClientForDiagnostics(cmd, client).Do(req)
}
