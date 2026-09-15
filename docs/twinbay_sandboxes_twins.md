## twinbay sandboxes twins

Operations for twins

### Synopsis

Operations for twins

```
twinbay sandboxes twins [flags]
```

### Options

```
  -h, --help   help for twins
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
      --organization-api-key string   An organization API key, as minted by POST /organizations/current/api-keys.
  -o, --output-format string          Specify the output format. Options: pretty, json, yaml, table, toon. (default "pretty")
      --raw-output                    Write --jq string results as raw text instead of JSON strings (like jq -r); non-string results stay JSON
      --server string                 Select a server by index (for indexed servers) or name (for named servers)
      --server-url string             Override the default server URL
      --timeout string                HTTP request timeout (e.g., 30s, 5m, 100ms)
      --usage                         Print the CLI Usage schema in KDL format
```

### SEE ALSO

* [twinbay sandboxes](twinbay_sandboxes.md)	 - Create and edit isolated provider sandboxes
* [twinbay sandboxes twins advance](twinbay_sandboxes_twins_advance.md)	 - Advance a deterministic twin lifecycle
* [twinbay sandboxes twins credential](twinbay_sandboxes_twins_credential.md)	 - Collect the twin's API key
* [twinbay sandboxes twins records](twinbay_sandboxes_twins_records.md)	 - Operations for records
* [twinbay sandboxes twins start](twinbay_sandboxes_twins_start.md)	 - Start a sandbox twin
* [twinbay sandboxes twins stop](twinbay_sandboxes_twins_stop.md)	 - Stop a sandbox twin

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
