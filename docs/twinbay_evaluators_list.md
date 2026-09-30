## twinbay evaluators list

List evaluators

### Synopsis

Lists every compilation for a test, optionally filtered by version.

```
twinbay evaluators list [flags]
```

### Examples

```
  twinbay evaluators list --test b1ca31c4-0107-480e-a3ad-308c55b0b0e7
```

### Options

```
  -a, --all                   Automatically paginate and fetch all results (streams NDJSON for JSON output)
  -h, --help                  help for list
      --max-pages int         Maximum number of pages to fetch when using --all (0 = no limit)
  -p, --page int              Page number (default 1)
  -s, --size int              Page size (default 50)
      --test string           [required]
      --test-version string   integer value
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
      --map string[="tree"]           Print the commands below this one instead of running it: tree (default), paths or json
      --no-interactive                Disable all interactive features (auto-prompting, explorer auto-launch, TUI forms)
      --organization-api-key string   An organization API key. Create one at https://console.twinbay.ai/api-keys (alias --api-key; env TWINBAY_API_KEY)
  -o, --output-format string          Specify the output format. Options: pretty, json, yaml, table, toon. (default "pretty")
      --raw-output                    Write --jq string results as raw text instead of JSON strings (like jq -r); non-string results stay JSON
      --server string                 Select a server by index (for indexed servers) or name (for named servers)
      --server-url string             Override the default server URL
      --timeout string                HTTP request timeout (e.g., 30s, 5m, 100ms)
      --usage                         Print the CLI Usage schema in KDL format
```

### SEE ALSO

* [twinbay evaluators](twinbay_evaluators.md)	 - Operations for evaluators

### Machine interface

* `twinbay evaluators list --usage` — this command's flags, defaults and env vars as machine-readable KDL
* `twinbay evaluators list --dry-run` — preview the request without OS-keychain access or a network call (human preview on stderr)
* `--dry-run --output-format json` (or a caller-explicit `--jq`) writes one preview object per request as NDJSON on stdout; jq is not applied to previews
* `--output-format json` or `--jq <expr>` for machine-readable live output; in agent mode errors are a JSON envelope on stderr

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
