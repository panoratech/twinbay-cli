## twinbay

Twinbay: Backend API

### Synopsis

Twinbay: Backend API

```
twinbay [flags]
```

### Options

```
      --agent-mode                    Enable structured errors and default TOON output for AI coding agents. Automatically enabled when a known agent environment is detected (CLAUDECODE, CURSOR_AGENT, etc.). Use --agent-mode=false to disable.
      --color string                  Control colored output: auto (color when output is a TTY), always, or never. Respects NO_COLOR and FORCE_COLOR env vars. (default "auto")
  -d, --debug                         Log request and response diagnostics to stderr
      --dry-run                       Preview API requests without sending them (no network, no OS keychain). Human preview on stderr; with -o json or --jq, one JSON object per request on stdout. Local mutation commands (auth login, auth logout and configure) make no request: they skip prompts and writes and report a no-op (stderr, or one JSON object on stdout in the machine form)
  -H, --header stringArray            Set a custom HTTP request header (format: "Key: Value"). Can be specified multiple times.
  -h, --help                          help for twinbay
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

* [twinbay api-keys](twinbay_api-keys.md)	 - Long-lived credentials for callers that cannot hold an AuthKit session — agents, SDKs, CI
* [twinbay auth](twinbay_auth.md)	 - Manage authentication credentials
* [twinbay configure](twinbay_configure.md)	 - Configure authentication credentials and preferences
* [twinbay environment-logs](twinbay_environment-logs.md)	 - Operations for environment-logs
* [twinbay environment-records](twinbay_environment-records.md)	 - Operations for environment-records
* [twinbay environment-templates](twinbay_environment-templates.md)	 - Operations for environment-templates
* [twinbay environment-twins](twinbay_environment-twins.md)	 - Operations for environment-twins
* [twinbay environments](twinbay_environments.md)	 - Create and edit isolated provider environments
* [twinbay explore](twinbay_explore.md)	 - Interactively browse and run commands
* [twinbay organizations](twinbay_organizations.md)	 - Organizations the caller belongs to
* [twinbay twins](twinbay_twins.md)	 - Browse the digital twins available for new environments
* [twinbay users](twinbay_users.md)	 - The current user
* [twinbay version](twinbay_version.md)	 - Print the CLI version
* [twinbay whoami](twinbay_whoami.md)	 - Display current authentication configuration

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
