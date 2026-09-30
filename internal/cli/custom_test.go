package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/panoratech/twinbay-cli/internal/config"
	"github.com/panoratech/twinbay-cli/internal/output"
)

// runCLI executes the real command tree in-process, the way main does.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	config.Reset()
	config.ResetKeyring()
	config.SetKeyringBackend(availableKeyring{})
	output.ResetAgentMode()
	t.Cleanup(func() {
		config.Reset()
		config.ResetKeyring()
		output.ResetAgentMode()
	})
	root, err := NewRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(strings.NewReader(""))
	err = ExecuteRoot(context.Background(), root, append([]string{"--no-interactive", "--agent-mode=false"}, args...))
	return out.String(), errOut.String(), err
}

func isolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TWINBAY_API_KEY", "")
	t.Setenv("CLI_TWINBAY_ORGANIZATION_API_KEY", "")
	t.Setenv("CLI_TWINBAY_ACCESS_TOKEN", "")
}

func TestDryRunSucceedsForNon200Operations(t *testing.T) {
	isolateHome(t)
	id := "0d7e9c1a-3b1f-4c55-9b0e-7f3a2c1d9e44"
	for _, args := range [][]string{
		{"organizations", "create", "--name", "Acme"},                                      // 201
		{"api-keys", "create", "--name", "ci"},                                             // 201
		{"scenarios", "create", "--body", `{"name":"s","twins":["stripe"]}`},               // 201
		{"tests", "create", "--body", `{"name":"t","task_description":"d","outcomes":[]}`}, // 201
		{"test-runs", "create", "--environment-id", id, "--evaluator-id", id},              // 201
		{"environments", "create", "--twins", `["stripe"]`},                                // 202
		{"twins", "start", id}, // 202
		{"twins", "stop", id},  // 202
		{"seeds", "create", "--body", `{"name":"s","twin":"zendesk","prompt":"p"}`}, // 202
		{"evaluators", "create", "--test-id", id, "--twins", `["stripe"]`},          // 202
		{"evaluations", "create", "--test-run-id", id, "--phase", "final"},          // 202
		{"environments", "exports", "create", id},                                   // 202
		{"api-keys", "revoke", id},                                                  // 204
		{"scenarios", "delete", id},                                                 // 204
		{"seeds", "delete", id},                                                     // 204
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			_, stderr, err := runCLI(t, append(args, "--dry-run")...)
			if err != nil {
				t.Fatalf("exit error %v\n%s", err, stderr)
			}
			if !strings.Contains(stderr, "[DRY-RUN] Would send:") {
				t.Fatalf("no preview:\n%s", stderr)
			}
		})
	}
}

func dryRunURL(t *testing.T, args ...string) string {
	t.Helper()
	stdout, stderr, err := runCLI(t, append(args, "--dry-run", "-o", "json")...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, stderr)
	}
	var preview struct {
		Request struct{ Method, URL string } `json:"request"`
	}
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, stdout)
	}
	return preview.Request.Method + " " + preview.Request.URL
}

func TestNestedSubresourcesTakePositionalIDs(t *testing.T) {
	isolateHome(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"environments", "logs", "list", "E"}, "GET https://api.twinbay.ai/environments/E/logs?page=1&size=50"},
		{[]string{"environments", "logs", "retrieve", "E", "R"}, "GET https://api.twinbay.ai/environments/E/logs/R"},
		{[]string{"environments", "exports", "download", "E", "X"}, "GET https://api.twinbay.ai/environments/E/exports/X/download"},
		{[]string{"tests", "versions", "retrieve", "T", "3"}, "GET https://api.twinbay.ai/tests/T/versions/3"},
		{[]string{"tests", "versions", "retrieve", "--test-id", "T", "--version", "3"}, "GET https://api.twinbay.ai/tests/T/versions/3"},
		{[]string{"twins", "records", "update", "W", "customers", "cus_1", "--fields", "{}"}, "PUT https://api.twinbay.ai/twins/W/records/customers/cus_1"},
		// Deprecated paths and flag names keep working.
		{[]string{"environment-logs", "retrieve", "--environment-id", "E", "--request-id", "R"}, "GET https://api.twinbay.ai/environments/E/logs/R"},
		{[]string{"tests", "retrieve-version", "--test-id", "T", "--version-param", "3"}, "GET https://api.twinbay.ai/tests/T/versions/3"},
	} {
		if got := dryRunURL(t, tc.args...); got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.args, got, tc.want)
		}
	}

	if _, _, err := runCLI(t, "twins", "records", "list", "W", "customers", "extra"); err == nil {
		t.Error("extra positional argument was accepted")
	}
	stdout, _, _ := runCLI(t, "tests", "versions", "retrieve", "--help")
	if !strings.Contains(stdout, "--version int") || strings.Contains(stdout, "version-param") {
		t.Errorf("version flag is not named --version:\n%s", stdout)
	}
	stdout, _, _ = runCLI(t, "environments", "exports", "create", "--help")
	if strings.Contains(stdout, "body-param") {
		t.Errorf("--body-param is still shown:\n%s", stdout)
	}
}

func TestDestructiveCommandsRequireConfirmation(t *testing.T) {
	isolateHome(t)
	for _, args := range [][]string{{"scenarios", "delete", "S"}, {"api-keys", "revoke", "K"}, {"delete", "/scenarios/S"}} {
		_, _, err := runCLI(t, args...)
		if err == nil || !strings.Contains(err.Error(), "--confirm") {
			t.Errorf("%v ran without confirmation: %v", args, err)
		}
		if _, stderr, err := runCLI(t, append(args, "--confirm", "--dry-run")...); err != nil {
			t.Errorf("%v --confirm: %v\n%s", args, err, stderr)
		}
	}
}

func TestRawCommandsAndAPIKeyAliases(t *testing.T) {
	isolateHome(t)
	type seen struct{ method, uri, key, body string }
	var got seen
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = seen{r.Method, r.URL.RequestURI(), r.Header.Get("X-API-Key"), string(body)}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	stdout, stderr, err := runCLI(t, "--server-url", server.URL, "--api-key", "flag-key", "get", "/environments", "-f", "size=5", "--jq", ".ok")
	if err != nil || strings.TrimSpace(stdout) != "true" {
		t.Fatalf("get: %v %q %s", err, stdout, stderr)
	}
	if want := (seen{"GET", "/environments?size=5", "flag-key", ""}); got != want {
		t.Errorf("get sent %+v, want %+v", got, want)
	}

	t.Setenv("TWINBAY_API_KEY", "env-key")
	if _, stderr, err := runCLI(t, "--server-url", server.URL, "post", "/scenarios", "-f", "name=s", "-f", `twins:=["stripe"]`); err != nil {
		t.Fatalf("post: %v %s", err, stderr)
	}
	if want := (seen{"POST", "/scenarios", "env-key", `{"name":"s","twins":["stripe"]}`}); got != want {
		t.Errorf("post sent %+v, want %+v", got, want)
	}
}

func TestConfigSetListUnset(t *testing.T) {
	isolateHome(t)
	if _, stderr, err := runCLI(t, "config", "set", "timeout", "30s"); err != nil {
		t.Fatalf("set: %v %s", err, stderr)
	}
	if _, _, err := runCLI(t, "config", "set", "output-format", "xml"); err == nil {
		t.Error("invalid output-format was accepted")
	}
	stdout, _, err := runCLI(t, "config", "list", "-o", "json")
	if err != nil || !strings.Contains(stdout, `"key":"timeout","source":"config","value":"30s"`) {
		t.Fatalf("list: %v %s", err, stdout)
	}
	if _, _, err := runCLI(t, "config", "unset", "timeout"); err != nil {
		t.Fatal(err)
	}
	stdout, _, _ = runCLI(t, "config", "list", "-o", "json")
	if !strings.Contains(stdout, `"key":"timeout","source":"unset"`) {
		t.Fatalf("timeout still set: %s", stdout)
	}
}

func TestMapFormats(t *testing.T) {
	isolateHome(t)
	stdout, _, err := runCLI(t, "--map=paths")
	if err != nil || !strings.Contains(stdout, "twinbay environments logs retrieve [environment-id] [request-id]\n") {
		t.Fatalf("paths: %v\n%s", err, stdout)
	}
	if strings.Contains(stdout, "environment-logs") {
		t.Error("deprecated path listed in the map")
	}
	stdout, _, err = runCLI(t, "tests", "--map=json")
	var node commandNode
	if err != nil || json.Unmarshal([]byte(stdout), &node) != nil || node.Path != "twinbay tests" {
		t.Fatalf("json: %v\n%s", err, stdout)
	}
	if stdout, _, _ = runCLI(t, "twins", "--map"); !strings.Contains(stdout, "  records") {
		t.Errorf("tree: %s", stdout)
	}
	if _, _, err = runCLI(t, "--map=xml"); err == nil {
		t.Error("invalid --map accepted")
	}
}
