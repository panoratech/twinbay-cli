# twinbay

Command-line interface for the *Twinbay* API.

[![Built by Speakeasy](https://img.shields.io/badge/Built_by-SPEAKEASY-374151?style=for-the-badge&labelColor=f3f4f6)](https://www.speakeasy.com/?utm_source=github-com/panoratech/twinbay-cli&utm_campaign=cli)
[![License: MIT](https://img.shields.io/badge/LICENSE_//_MIT-3b5bdb?style=for-the-badge&labelColor=eff6ff)](https://opensource.org/licenses/MIT)


<br /><br />
> [!IMPORTANT]
> This CLI is not yet ready for production use. To complete setup please follow the steps outlined in your [workspace](https://app.speakeasy.com/org/panora-technologies-inc/twinbay). Delete this section before > publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary

Twinbay: Backend API
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [twinbay](#twinbay)
  * [CLI Installation](#cli-installation)
  * [Shell Completion](#shell-completion)
  * [CLI Example Usage](#cli-example-usage)
  * [For AI agents](#for-ai-agents)
  * [Authentication](#authentication)
  * [Commands](#commands)
  * [Request Body Input](#request-body-input)
  * [Server Selection](#server-selection)
  * [Output Formats](#output-formats)
  * [Pagination](#pagination)
  * [Error Handling](#error-handling)
  * [Diagnostics](#diagnostics)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start CLI Installation [installation] -->
## CLI Installation

### Quick Install (Linux/macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/panoratech/twinbay-cli/main/scripts/install.sh | bash
```

### Quick Install (Windows PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/panoratech/twinbay-cli/main/scripts/install.ps1 | iex
```

### Go Install

Alternatively, install directly via Go:

```bash
go install github.com/panoratech/twinbay-cli/cmd/twinbay@latest
```

### Manual Download

Download pre-built binaries for your platform from the [releases page](https://github.com/panoratech/twinbay-cli/releases).
<!-- End CLI Installation [installation] -->

<!-- Start Shell Completion [completion] -->
## Shell Completion

Shell completions are available for Bash, Zsh, Fish, and PowerShell.

### Bash

```bash
# Add to ~/.bashrc:
source <(twinbay completion bash)

# Or install permanently:
twinbay completion bash > /etc/bash_completion.d/twinbay
```

### Zsh

```zsh
# Add to ~/.zshrc:
source <(twinbay completion zsh)

# Or install permanently:
twinbay completion zsh > "${fpath[1]}/_twinbay"
```

### Fish

```fish
twinbay completion fish | source

# Or install permanently:
twinbay completion fish > ~/.config/fish/completions/twinbay.fish
```

### PowerShell

```powershell
twinbay completion powershell | Out-String | Invoke-Expression
```
<!-- End Shell Completion [completion] -->

<!-- Start CLI Example Usage [usage] -->
## CLI Example Usage

### Quick start

```bash
# Compile the latest test version
twinbay evaluators create-and-wait --test-id 7ef34f0b-0ae6-4e26-ae92-f49d239d4d7d --twins stripe

# Finish and grade a test run
twinbay evaluations create-and-wait --test-run-id b04bd7fb-3f6c-4de8-9c5a-918909419dcc --phase final

# Generate a support backlog
twinbay seeds create-and-wait --name aging-backlog --twin zendesk "50 open tickets, half aging past 90 days"
```

### Example

```bash
twinbay users retrieve --organization-api-key test_api_key

```
<!-- End CLI Example Usage [usage] -->

<!-- Start For AI agents [agents] -->
## For AI agents

This CLI is built to be driven by AI coding agents as well as people: everything an agent needs is discoverable from the binary itself, and every command can be validated without credentials. Work down this ladder:

| Run | You get |
|-----|---------|
| `twinbay --help`, `twinbay api-keys list --help` | Commands by category, runnable examples, flags |
| `twinbay --usage`, `twinbay api-keys list --usage` | The command surface as machine-readable [KDL](https://kdl.dev): commands, aliases, flags, defaults, env vars, config keys |
| `twinbay evaluators create-and-wait --schema` | The exact JSON Schema of the command's request body (all `$ref`s bundled) — build a valid `--body` from it |
| `twinbay api-keys list --dry-run` | The exact HTTP request (method, URL, headers, body), with no credentials or network call |
| `twinbay api-keys list --output-format json` (or `--jq`) | Machine-readable output |

### Discover the command surface

```bash
# Every command, flag, default, env var and config key, as KDL
twinbay --usage

# One command's subtree only
twinbay api-keys list --usage
```

### Read the exact request schema

`--schema` is available on every command that accepts a request body (`--body`, stdin, or a whole-body flag where the command has one), including intent commands. It prints the JSON Schema the request is validated against and exits without calling the API.

```bash
# JSON Schema (draft 2020-12) of the request body, with every $ref bundled under $defs
twinbay evaluators create-and-wait --schema
```

### Probe before you spend

Start quota-spending commands with `--dry-run`. It validates inputs, resolves the request, redacts secrets and binary payloads, makes no network call, and exits 0. It never reads the OS keychain; credentials supplied by flag, environment, or config file are included only as `[REDACTED]`.

```bash
# Human preview: the [DRY-RUN] block is on stderr and stdout is empty
twinbay api-keys list --dry-run
twinbay evaluators create-and-wait --test-id 7ef34f0b-0ae6-4e26-ae92-f49d239d4d7d --twins stripe --dry-run

# Machine preview: compact JSON on stdout and silent stderr
twinbay api-keys list --dry-run --output-format json
```

The machine form writes one object per would-be request, one per line (NDJSON for multi-request commands), with exactly this shape:

```json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
```

`body` is a parsed JSON value when the body is JSON, a string for text, `"<bytes:N>"` for binary data, and `null` when absent. An explicit caller `--jq` also selects this JSON preview protocol, but the filter is not applied to preview objects. Command-declared jq presets do not select or filter the preview.

Local mutation commands make no request under `--dry-run`: instead of a preview they emit one `{"dry_run":true,"local":true,"command":"…","message":"…"}` object. `select(.request)` keeps only would-be requests; `select(.local)` keeps the local no-ops.

### Machine-readable output

```bash
# JSON on stdout
twinbay api-keys list --output-format json

# Filter or reshape with a jq expression (always emits JSON, overrides --output-format)
twinbay api-keys list --jq '.'

# Print jq string results as plain text instead of JSON strings (like jq -r)
twinbay api-keys list --jq '.' --raw-output
```

`--output-format toon` emits [TOON](https://github.com/toon-format/spec), a compact line-oriented format that uses fewer tokens than JSON; it is the default in agent mode.

### Interactive mode
Required-input prompts and guided `configure` / `auth login` forms are enabled by default. Required-input prompts require an interactive terminal; off-TTY forms read line input from stdin. Use `--no-interactive` to force flag-only execution.

```bash
# Prompt for missing command inputs
twinbay evaluators create-and-wait --interactive

# Open the guided configuration form
twinbay configure --interactive

# Explicitly launch the terminal command explorer
twinbay explore
```

### Agent mode and structured errors
Agent mode turns on automatically when a known agent environment is detected (`CLAUDECODE`, `CURSOR_AGENT`, `CODEX`, `AIDER`, `CLINE`, `WINDSURF_AGENT`, `GITHUB_COPILOT`, `AMAZON_Q`, `GEMINI_CODE_ASSIST`, `SRC_CODY`) or with `--agent-mode` (`--agent-mode=false` disables detection).
In agent mode interactive prompts never launch, output defaults to TOON, and every failure — API errors and CLI usage errors alike — is one JSON envelope on stderr:
Outside agent mode, explicit JSON and `--jq` preserve the compatibility envelope without classification; enable agent mode to request the classified contract.

```json
{
  "error": "...",
  "error_type": "validation_error",
  "error_reason": "CLI_VALIDATION",
  "exit_code": 2,
  "message": "human-readable message",
  "hints": ["what to try next"]
}
```

`error_type` is one of `authentication_error`, `authorization_error`, `not_found`, `validation_error`, `rate_limit_error`, `server_error`, `api_error`, `connection_error`, `protocol_error`, `runtime_error`, `unsupported_error`, `async_failed`, `async_timeout`, `async_unknown_state`. Classification derives from the HTTP status and transport evidence; `error_reason` is absent for API errors. Status-less local failures may use `CLI_VALIDATION`, `CLI_CONNECTION`, `CLI_PROTOCOL`, `CLI_RUNTIME`, `CLI_UNAVAILABLE`, `CLI_AUTHENTICATION`, or the async polling reasons `CLI_ASYNC_FAILED`, `CLI_ASYNC_TIMEOUT`, and `CLI_ASYNC_UNKNOWN_STATE`. `hints` preserves server guidance first, adds the most specific local taxonomy guidance, then typed CLI and command-specific guidance, removing exact duplicates. `exit_code` is always the code for the final `error_type` shown in the envelope: 1 runtime, 2 usage, or 3 authentication/authorization.

```bash
# "evaluators create-and-wait" declares hints for CLI_ASYNC_FAILED, CLI_ASYNC_TIMEOUT; a failing request returns them in the envelope
twinbay evaluators create-and-wait --test-id 7ef34f0b-0ae6-4e26-ae92-f49d239d4d7d --twins stripe
```

### Lists, streams, and files

List commands accept `--all` to fetch every page and stream results as they arrive (one JSON value per line with `--output-format json`; `--max-pages N` bounds the walk).

Structured output and agent mode never write pagination hints to stderr; if a later page fails or the server repeats a cursor, the command exits non-zero after the pages already written.

```bash
twinbay api-keys list --all --output-format json
```

Long-running commands poll to a terminal response; human progress goes to stderr and machine-mode success keeps stderr silent. Add `--async` to `twinbay evaluators create-and-wait --test-id 7ef34f0b-0ae6-4e26-ae92-f49d239d4d7d --twins stripe` to return its handle immediately, or tune foreground polling with `--poll-interval <duration>` and `--poll-timeout <duration>`. Resume an escaped or timed-out operation with `twinbay evaluators retrieve --evaluator-id <id>`.
<!-- End For AI agents [agents] -->

<!-- Start Authentication [security] -->
## Authentication

Authentication credentials can be configured in four ways (in order of priority):

### 1. Command-line flags

Pass credentials directly as flags to any command:

```bash
twinbay --organization-api-key "$CLI_TWINBAY_ORGANIZATION_API_KEY" api-keys list
```

### 2. Environment variables

Set credentials via environment variables:

| Variable | Description |
|----------|-------------|
| `CLI_TWINBAY_ORGANIZATION_API_KEY` | An organization API key. Create one at https://console.twinbay.ai/api-keys |

### 3. OS Keychain (recommended for workstations)

Credentials are stored securely in your operating system's keychain when you run:

```bash
twinbay configure
```

Secret credentials (tokens, API keys, passwords) are automatically stored in:
- **macOS**: Keychain
- **Linux**: GNOME Keyring / KWallet (via D-Bus Secret Service)
- **Windows**: Windows Credential Locker

If no keychain is available (e.g., in CI environments), credentials fall back to the config file.

### 4. Configuration file

Run the interactive `configure` command to store non-secret settings:

```bash
twinbay configure
```

Configuration is stored in `~/.config/twinbay/config.yaml`.
<!-- End Authentication [security] -->

<!-- Start Commands [operations] -->
## Commands

Commands are grouped the way `twinbay --help` shows them. Every command accepts `--help`; body-bearing commands also accept `--schema` (exact request JSON Schema) and `--dry-run` (preview the request without sending it) — see [For AI agents](#for-ai-agents).

### Additional commands

* [`users`](docs/twinbay_users.md) - The current user
  * [`retrieve`](docs/twinbay_users_retrieve.md) - Read the authenticated user
* [`organizations`](docs/twinbay_organizations.md) - Organizations the caller belongs to
  * [`list`](docs/twinbay_organizations_list.md) - List your organizations
  * [`create`](docs/twinbay_organizations_create.md) - Create an organization
  * [`retrieve`](docs/twinbay_organizations_retrieve.md) - Read the active organization
  * [`update`](docs/twinbay_organizations_update.md) - Rename the active organization
* [`api-keys`](docs/twinbay_api-keys.md) - Long-lived credentials for callers that cannot hold an AuthKit session — agents, SDKs, CI
  * [`create`](docs/twinbay_api-keys_create.md) - Create an API key
  * [`list`](docs/twinbay_api-keys_list.md) - List API keys
  * [`revoke`](docs/twinbay_api-keys_revoke.md) - Revoke an API key
* [`catalog`](docs/twinbay_catalog.md) - Browse the digital twins available for new environments
  * [`list`](docs/twinbay_catalog_list.md) - List available twins
  * [`retrieve`](docs/twinbay_catalog_retrieve.md) - Retrieve a twin
* [`environments`](docs/twinbay_environments.md) - Create and edit isolated provider environments
  * [`create`](docs/twinbay_environments_create.md) - Create an environment
  * [`list`](docs/twinbay_environments_list.md) - List environments
  * [`retrieve`](docs/twinbay_environments_retrieve.md) - Retrieve an environment
* [`twins`](docs/twinbay_twins.md) - Manage provisioned twins by their IDs
  * [`retrieve`](docs/twinbay_twins_retrieve.md) - Retrieve a twin
  * [`start`](docs/twinbay_twins_start.md) - Start a twin
  * [`stop`](docs/twinbay_twins_stop.md) - Stop a twin
  * [`collect-credential`](docs/twinbay_twins_collect-credential.md) - Collect the twin's API key
  * [`advance`](docs/twinbay_twins_advance.md) - Advance a deterministic twin lifecycle
* [`twin-records`](docs/twinbay_twin-records.md) - Operations for twin-records
  * [`list`](docs/twinbay_twin-records_list.md) - List twin state
  * [`update`](docs/twinbay_twin-records_update.md) - Replace a twin record
* [`scenarios`](docs/twinbay_scenarios.md) - Reusable starting setups of provider twins and optional seeds
  * [`create`](docs/twinbay_scenarios_create.md) - Create a scenario
  * [`list`](docs/twinbay_scenarios_list.md) - List scenarios
  * [`retrieve`](docs/twinbay_scenarios_retrieve.md) - Retrieve a scenario
  * [`update`](docs/twinbay_scenarios_update.md) - Update a scenario
  * [`delete`](docs/twinbay_scenarios_delete.md) - Delete a scenario
* [`seeds`](docs/twinbay_seeds.md) - Starting states for a twin
  * [`create-and-wait`](docs/twinbay_seeds_create-and-wait.md) - Generate a seed and wait

    ```bash
    # Generate a support backlog
    twinbay seeds create-and-wait --name aging-backlog --twin zendesk "50 open tickets, half aging past 90 days"
    ```

  * [`create`](docs/twinbay_seeds_create.md) - Generate a seed
  * [`list`](docs/twinbay_seeds_list.md) - List seeds
  * [`list-suggestions`](docs/twinbay_seeds_list-suggestions.md) - List suggested starting states
  * [`retrieve`](docs/twinbay_seeds_retrieve.md) - Retrieve a seed
  * [`delete`](docs/twinbay_seeds_delete.md) - Delete a seed
* [`tests`](docs/twinbay_tests.md) - Outcomes described in natural language, compiled once into request matchers, and evaluated deterministically against the traffic an attempt was served
  * [`create`](docs/twinbay_tests_create.md) - Create a test
  * [`list`](docs/twinbay_tests_list.md) - List tests
  * [`retrieve`](docs/twinbay_tests_retrieve.md) - Retrieve a test
  * [`update`](docs/twinbay_tests_update.md) - Update a test
  * [`list-versions`](docs/twinbay_tests_list-versions.md) - List a test's versions
  * [`retrieve-version`](docs/twinbay_tests_retrieve-version.md) - Retrieve a test version
* [`evaluators`](docs/twinbay_evaluators.md) - Operations for evaluators
  * [`create-and-wait`](docs/twinbay_evaluators_create-and-wait.md) - Create an evaluator and wait

    ```bash
    # Compile the latest test version
    twinbay evaluators create-and-wait --test-id 7ef34f0b-0ae6-4e26-ae92-f49d239d4d7d --twins stripe
    # Compile a specific test version
    twinbay evaluators create-and-wait --test-id 7ef34f0b-0ae6-4e26-ae92-f49d239d4d7d --test-version 3 --twins stripe
    ```

  * [`create`](docs/twinbay_evaluators_create.md) - Create an evaluator
  * [`list`](docs/twinbay_evaluators_list.md) - List evaluators
  * [`retrieve`](docs/twinbay_evaluators_retrieve.md) - Retrieve an evaluator
* [`test-runs`](docs/twinbay_test-runs.md) - Operations for test-runs
  * [`create`](docs/twinbay_test-runs_create.md) - Start a test run
  * [`list`](docs/twinbay_test-runs_list.md) - List test runs
  * [`retrieve`](docs/twinbay_test-runs_retrieve.md) - Retrieve a test run
* [`evaluations`](docs/twinbay_evaluations.md) - Operations for evaluations
  * [`create-and-wait`](docs/twinbay_evaluations_create-and-wait.md) - Evaluate a test run and wait

    ```bash
    # Finish and grade a test run
    twinbay evaluations create-and-wait --test-run-id b04bd7fb-3f6c-4de8-9c5a-918909419dcc --phase final
    # Regrade the latest ready final capture
    twinbay evaluations create-and-wait --test-run-id b04bd7fb-3f6c-4de8-9c5a-918909419dcc --phase final --evaluator-id 1177bd8d-986f-44a2-aec2-1518dc46d6b0
    ```

  * [`create`](docs/twinbay_evaluations_create.md) - Evaluate a test run
  * [`list`](docs/twinbay_evaluations_list.md) - List a test run's evaluations
  * [`retrieve`](docs/twinbay_evaluations_retrieve.md) - Retrieve an evaluation
* [`evaluation-inputs`](docs/twinbay_evaluation-inputs.md) - Operations for evaluation-inputs
  * [`download`](docs/twinbay_evaluation-inputs_download.md) - Sign a link to a preserved capture
* [`environment-logs`](docs/twinbay_environment-logs.md) - Operations for environment-logs
  * [`list`](docs/twinbay_environment-logs_list.md) - List recent environment request logs
  * [`retrieve`](docs/twinbay_environment-logs_retrieve.md) - Retrieve an environment request log
* [`environment-exports`](docs/twinbay_environment-exports.md) - Operations for environment-exports
  * [`create`](docs/twinbay_environment-exports_create.md) - Export an environment's traffic
  * [`list`](docs/twinbay_environment-exports_list.md) - List an environment's exports
  * [`retrieve`](docs/twinbay_environment-exports_retrieve.md) - Retrieve an export
  * [`download`](docs/twinbay_environment-exports_download.md) - Sign a link to an exported file
<!-- End Commands [operations] -->

<!-- Start Request Body Input [stdinpiping] -->
## Request Body Input

Commands that accept a request body take it three ways, with a clear priority chain. The examples use `twinbay test-runs create`; every body-bearing command works the same way and prints its exact request schema with `--schema`.

### Individual flags (highest priority)

Each top-level body field is a flag:

```bash
twinbay test-runs create --environment-id '24e86869-c8aa-4fa2-9d08-6690490dcd5a' --evaluator-id '24e86869-c8aa-4fa2-9d08-6690490dcd5a'
```

### `--body` flag

Provide the entire request body as a JSON string:

```bash
twinbay test-runs create --body '{"environment_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a","evaluator_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a"}'
```

Individual flags override `--body` values:

```bash
# Sends {"environment_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a (updated)","evaluator_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a"}
twinbay test-runs create --body '{"environment_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a","evaluator_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a"}' --environment-id '24e86869-c8aa-4fa2-9d08-6690490dcd5a (updated)'
```

### Stdin piping (lowest priority)

Pipe JSON into any command that accepts a request body:

```bash
echo '{"environment_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a","evaluator_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a"}' | twinbay test-runs create
```

Individual flags override stdin values:

```bash
# Sends {"environment_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a (updated)","evaluator_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a"}
echo '{"environment_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a","evaluator_id":"24e86869-c8aa-4fa2-9d08-6690490dcd5a"}' | twinbay test-runs create --environment-id '24e86869-c8aa-4fa2-9d08-6690490dcd5a (updated)'
```

This is useful for chaining commands, reading from files, or scripting:

```bash
# Read body from a file
twinbay test-runs create < request.json

# Pipe from another command
curl -s https://example.com/request.json | twinbay test-runs create
```

### Priority

When multiple input methods are used, the priority is:

| Priority | Source | Description |
|----------|--------|-------------|
| 1 (highest) | Individual flags | `--environment-id ...` always wins |
| 2 | `--body` flag | Whole-body JSON via flag |
| 3 (lowest) | Stdin | Piped JSON input |
<!-- End Request Body Input [stdinpiping] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Override Server URL

Use `--server-url` to override the server URL entirely, bypassing any named or indexed server selection:

```bash
twinbay --server-url https://custom-api.example.com api-keys list
```

**Precedence**: `--server-url` > `--server` > default
<!-- End Server Selection [server] -->

<!-- Start Output Formats [output-formats] -->
## Output Formats

Every command supports a `--output-format` flag that controls how the response is rendered to stdout.

### Available formats

| Format | Flag | Description |
|--------|------|-------------|
| Pretty | `--output-format pretty` (default) | Aligned key-value pairs with color, nested indentation. Human-readable at a glance. |
| JSON | `--output-format json` | JSON output. Passthrough when the response is already JSON (preserves original field order and numeric precision). Falls back to typed marshaling otherwise. |
| YAML | `--output-format yaml` | YAML output via standard marshaling. |
| Table | `--output-format table` | Tabular output for array responses. |
| TOON | `--output-format toon` | [Token-Oriented Object Notation](https://github.com/toon-format/spec) — a compact, line-oriented format that typically uses 30–60% fewer tokens than JSON. Well-suited for piping responses into LLM prompts. |

```bash
# Default pretty output
twinbay api-keys list

# Machine-readable JSON
twinbay api-keys list --output-format json

# TOON for LLM-friendly compact output
twinbay api-keys list --output-format toon

# Pipe JSON to jq without using --output-format
twinbay api-keys list --output-format json | jq '.'
```

### jq filtering

Use `--jq` to filter or transform the response inline using a [jq](https://jqlang.org) expression. This always outputs JSON and overrides `--output-format`:

```bash
# Extract a single field
twinbay api-keys list --jq '.'

# Reshape with any jq program; --raw-output prints string results as plain text (like jq -r)
twinbay api-keys list --jq '.' --raw-output
```

### Color control

Use `--color` to control terminal colors:

| Value | Behavior |
|-------|----------|
| `auto` (default) | Color when stdout is a TTY, plain text otherwise |
| `always` | Always colorize |
| `never` | Never colorize |

The `NO_COLOR` and `FORCE_COLOR` environment variables are also respected.

### Streaming and pagination

When using `--all` (pagination) or streaming operations, output is written incrementally as items arrive:

| Format | Streaming behavior |
|--------|-------------------|
| `json` | One compact JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| `yaml` | YAML documents separated by `---` |
| `toon` | One TOON-encoded object per block, separated by blank lines |
| `pretty` (default) | Pretty-printed items separated by blank lines |
<!-- End Output Formats [output-formats] -->

<!-- Start Pagination [pagination] -->
## Pagination

Some operations in this CLI support automatic pagination. These operations accept `--all` to automatically fetch all pages and stream results incrementally.

### Basic usage

```bash
# Fetch a single page (default behavior)
twinbay api-keys list

# Automatically fetch all pages
twinbay api-keys list --all
```

### Limiting pages

Use `--max-pages` with `--all` to cap the number of pages fetched. A negative value is invalid; `0` means unlimited. Passing `--max-pages` without `--all` is an error.

```bash
# Fetch at most 5 pages
twinbay api-keys list --all --max-pages 5
```

### Output formats

When using `--all`, output is streamed as each page is fetched. Operations whose pagination declaration names an `outputs.results` array emit one item at a time. Other operations emit one complete page object at a time, preserving the single-page response shape and any continuation cursor.

| Format | Behavior |
|--------|----------|
| `--output-format json` | One JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| `--output-format yaml` | YAML documents separated by `---` |
| `--output-format toon` | One TOON-encoded block per item, separated by blank lines |
| Default (pretty) | Pretty-printed items separated by blank lines |

```bash
# Stream all results as NDJSON
twinbay api-keys list --all --output-format json

# Pipe to jq for further processing
twinbay api-keys list --all --output-format json | jq '.'

# Use the built-in --jq flag
twinbay api-keys list --all --jq '.'
```

### How it works

Under the hood, `--all` calls the operation once, then follows the underlying `Next()` pagination closure to fetch subsequent pages. Results are written to stdout as they arrive rather than buffered in memory, so this works well even with large result sets.

Without `--all`, paginated operations behave like any other command — pass cursor, page, offset, or limit flags manually and get a single page of results. In pretty or table output, a cursor response that proves another page exists prints a hint on stderr. JSON, YAML, TOON, `--jq`, and agent mode keep stderr silent on success. Offset/limit responses do not guess from a full result page. Cursor operations that declare both a results array and a mutable limit also suppress the hint because the client cannot safely reproduce the SDK's runtime limit check.

Pagination can fail after earlier pages have already been written. A later-page API failure or a repeated/cyclic continuation cursor stops with a non-zero exit status; callers should treat stdout as partial whenever the command exits non-zero. `--all` tracks cursor values and stops before issuing another request when the server repeats one; the same applies to next URLs when the target generator supports them.
<!-- End Pagination [pagination] -->

<!-- Start Error Handling [errors] -->
## Error Handling

The CLI uses standard exit codes to indicate success or failure:

| Exit Code | Meaning |
|-----------|---------|
| `0` | Success |
| `1` | Runtime/API failure |
| `2` | Usage or input failure |
| `3` | Authentication or authorization failure |

On success, the response data is printed to **stdout** as JSON. On failure, error details are printed to **stderr**.

```bash
# Capture output and handle errors
twinbay api-keys list --output-format json > output.json 2> error.log
if [ $? -ne 0 ]; then
  echo "Error occurred, see error.log"
fi
```
This CLI uses unclassified error rendering outside agent mode: pretty and TOON print the API error text as received, while `--output-format json` and `--jq` emit the unclassified envelope (including the configure `_hint` for HTTP 401/403) plus `exit_code`. Agent mode always emits the classified JSON envelope with `exit_code`, `error_type`, optional `error_reason`, `message`, `hints`, and optional `status_code` — see [For AI agents](#for-ai-agents).
<!-- End Error Handling [errors] -->

<!-- Start Diagnostics [diagnostics] -->
## Diagnostics

The CLI includes two diagnostic flags available on all commands:

### Dry Run

Preview what would be sent without making any network calls:

```bash
twinbay api-keys list --dry-run
```

In human output modes, stdout is empty and the `[DRY-RUN]` block goes to stderr. It includes:
- HTTP method and URL
- Request headers (sensitive values redacted)
- Request body preview (sensitive fields redacted)

With `--output-format json`, or with a caller-explicit `--jq`, stderr is silent and stdout is NDJSON: one compact preview object per would-be request. The jq filter is not applied, and command-declared jq presets do not select the JSON protocol.

```json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
```

JSON bodies remain structured; text bodies are strings; binary bodies are `"<bytes:N>"`; absent bodies are `null`. Headers retain all values as arrays, with credentials replaced by `[REDACTED]`. Dry-run never reads the OS keychain, but credentials supplied by flag, environment, or config file still appear redacted. The command exits successfully without contacting the API.

Local mutation commands emit one `{"dry_run":true,"local":true,"command":"…","message":"…"}` object in place of a preview; filter with `select(.request)` or `select(.local)`.

### Debug

Log request and response diagnostics while running normally:

```bash
twinbay api-keys list --debug
```

Debug output goes to stderr and includes:
- Request method, URL, headers, and body preview
- Response status, headers, and body preview
- Transport errors (if any)

The command still executes normally and produces its regular output on stdout.

### Flag Precedence

If both `--dry-run` and `--debug` are set, `--dry-run` takes precedence and no network calls are made.

### Security

Sensitive information is automatically redacted in diagnostic output:
- **Headers**: `Authorization`, `Cookie`, `Set-Cookie`, `X-API-Key`, and other security headers show `[REDACTED]`
- **Body**: JSON fields named `password`, `secret`, `token`, `api_key`, `client_secret`, etc. show `[REDACTED]`
- **Binary data**: binary media and canonical base64 strings are replaced with `<bytes:N>`
- **URL query**: credential-like query parameters are replaced with `[REDACTED]`

Diagnostic output should still be treated as potentially sensitive operational data.
<!-- End Diagnostics [diagnostics] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This CLI is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

While we value open-source contributions to this CLI, this library is generated programmatically. Any manual changes added to internal files will be overwritten on the next generation. 
We look forward to hearing your feedback. Feel free to open a PR or an issue with a proof of concept and we'll do our best to include it in a future release. 

### CLI Created by [Speakeasy](https://www.speakeasy.com/?utm_source=github-com/panoratech/twinbay-cli&utm_campaign=cli)
