## twinbay evaluator-versions retrieve

Retrieve an evaluator version

### Synopsis

Compilation is asynchronous: poll until `status` is `ready`, when `rules` holds the frozen matchers, or `error`, when `error_message` says why. Only a ready evaluator can start a test run.

```
twinbay evaluator-versions retrieve [evaluator-version-id] [flags]
```

### Examples

```
  twinbay evaluator-versions retrieve --evaluator-version-id ae52bf88-47c9-422f-ad08-9de23eb2fd7a
```

### Options

```
  -e, --evaluator-version-id string   string value (or pass it as the [evaluator-version-id] argument)
  -h, --help                          help for retrieve
```

### Options inherited from parent commands

```
      --agent-mode                    Enable structured errors and default TOON output for AI coding agents. Automatically enabled when a known agent environment is detected (CLAUDECODE, CURSOR_AGENT, etc.). Use --agent-mode=false to disable.
      --color string                  Control colored output: auto (color when output is a TTY), always, or never. Respects NO_COLOR and FORCE_COLOR env vars. (default "auto")
  -d, --debug                         Log request and response diagnostics to stderr
      --dry-run                       Preview API requests without sending them (no network, no OS keychain). Human preview on stderr; with -o json or --jq, one JSON object per request on stdout. Local mutation commands (auth login, auth logout and configure) make no request: they skip prompts and writes and report a no-op (stderr, or one JSON object on stdout in the machine form)
  -H, --header stringArray            Set a custom HTTP request header (format: "Key: Value"). Can be specified multiple times.
      --include-headers               Include HTTP response headers in the output
      --interactive                   Prompt for missing inputs and open guided configure/auth forms (forms fall back to line prompts on stdin off-TTY) (default true)
  -q, --jq string                     Filter and transform output using a jq expression (e.g., '.name', '.items[] | .id')
      --no-interactive                Disable all interactive features (auto-prompting, explorer auto-launch, TUI forms)
      --organization-api-key string   An organization API key. Create one at https://console.twinbay.ai/api-keys
  -o, --output-format string          Specify the output format. Options: pretty, json, yaml, table, toon. (default "pretty")
      --raw-output                    Write --jq string results as raw text instead of JSON strings (like jq -r); non-string results stay JSON
      --server string                 Select a server by index (for indexed servers) or name (for named servers)
      --server-url string             Override the default server URL
      --timeout string                HTTP request timeout (e.g., 30s, 5m, 100ms)
      --usage                         Print the CLI Usage schema in KDL format
```

### SEE ALSO

* [twinbay evaluator-versions](twinbay_evaluator-versions.md)	 - Operations for evaluator-versions

### Machine interface

* `twinbay evaluator-versions retrieve --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `twinbay evaluator-versions retrieve --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
