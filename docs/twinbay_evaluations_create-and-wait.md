## twinbay evaluations create-and-wait

Evaluate a test run and wait

### Synopsis

Capture and grade a test run, then wait for the verdict. Only passed exits successfully; failed, inconclusive, and error are failures. Naming another evaluator regrades the latest ready capture for the same phase. Use --async to return the queued evaluation immediately.

```
twinbay evaluations create-and-wait [flags]
```

### Examples

```
  twinbay evaluations create-and-wait --test-run-id b04bd7fb-3f6c-4de8-9c5a-918909419dcc --phase final
  twinbay evaluations create-and-wait --test-run-id b04bd7fb-3f6c-4de8-9c5a-918909419dcc --phase final --evaluator-id 1177bd8d-986f-44a2-aec2-1518dc46d6b0
```

### Options

```
      --async                    Return the operation handle without waiting for a terminal response
      --body string              Request body as JSON (advanced; replaces intent arguments). Can also be provided via stdin; @path reads a file, @- reads stdin to EOF. Use --schema to print the exact JSON Schema.
  -e, --evaluator-id string      Different ready evaluator used to regrade the latest ready capture for this run and phase.
  -h, --help                     help for create-and-wait
  -i, --idempotency-key string   Replaying the same key and payload returns the same evaluation; reusing it with a different payload answers 409.
  -p, --phase string             Checkpoint grades without closing the run; final closes and grades it. (options: checkpoint, final) [required]
      --poll-interval string     Override the initial polling interval (positive Go duration, for example 500ms or 2s)
      --poll-timeout string      Override the overall polling deadline (positive Go duration, at least the effective poll interval)
      --schema                   Print the exact JSON Schema of the request body and exit
  -t, --test-run-id string       Test run whose attributed traffic should be graded. [required]
```

### Options inherited from parent commands

```
      --access-token string           Access token issued by WorkOS AuthKit.
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

* [twinbay evaluations](twinbay_evaluations.md)	 - Operations for evaluations

### Machine interface

* `twinbay evaluations create-and-wait --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `twinbay evaluations create-and-wait --schema` — the exact JSON Schema of the request body (all `$ref`s bundled)
* `twinbay evaluations create-and-wait --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
