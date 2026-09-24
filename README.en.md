# cpa-secret-manager

A **proxy API key manager and token usage attribution plugin** for CLIProxyAPI (CPA): keep one remark per proxy key and see the token usage and model breakdown of that key.

CLIProxyAPI can add, edit and delete proxy keys, but it offers no way to annotate a key and no per-key token usage view. This plugin fills both gaps.

---

## Features

- **Key management**: list, add and delete proxy keys, always masked with per-row copy, plus filtering by key or remark; editing changes the remark only (the host owns the key, which is never rendered in clear text).
- **Remarks**: one remark per key (there is no name field), indexed by the SHA-256 digest of the key; the plugin **never stores key material**.
- **Generation**: keys are generated with exactly the official algorithm — an `sk-` prefix plus 48 alphanumeric characters, with modulo bias removed by rejection sampling.
- **Token usage**: cumulative requests, failures, input/output/reasoning/cache tokens and totals per key, with a per-model breakdown.
- **Native CPA integration**: the page is fully self-contained (no external CDN), reuses the management key the official center already stored when opened same-origin, and follows the host theme (including custom themes) and interface language in real time.

## Installation

Drop the plugin dynamic library into the CLIProxyAPI plugins directory. **The file name must stay `cpa-secret-manager`** (the plugin ID is derived from it):

| Platform | Path |
| --- | --- |
| Linux | `plugins/linux/amd64/cpa-secret-manager.so` |
| macOS | `plugins/darwin/arm64/cpa-secret-manager.dylib` |
| Windows | `plugins/windows/amd64/cpa-secret-manager.dll` |

Enable it in `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpa-secret-manager:
      enabled: true
      priority: 1
      # Optional: custom plugin state document path
      state_path: "data/cpa-secret-manager-cache.json"
```

Open the official management center and pick “API Key Manager” from the plugin menu.

> The host must support the plugin capabilities `usage_plugin` and `management_api`; the latest CLIProxyAPI release is recommended.

## Usage

- **Management key**: when opened same-origin the page reuses the session key the official center stored in `cli-proxy-auth`; for cross-origin deployments or when “remember password” is off, enter it through the header button — a manual entry stays in the current browser tab only.
- **State document**: remarks and usage live in `data/cpa-secret-manager-cache.json` (the Settings tab shows the resolved path). Deleting that file clears every remark and counter.

## Usage accounting

- Data comes from the host usage records (`usage_plugin`), attributed to the **proxy key that served the request** and grouped by model.
- Counters are **cumulative** and start with the first request the plugin observes. There are no time windows and no cost estimates.
- Requests whose upstream response carried no token counts still count as requests, with zero tokens.
- Only requests that hit a managed key are counted; records without one - or with a key that is not managed - are dropped outright, and the page never shows an "unattributed" bucket.
- Usage is flushed to disk on a 5 second cadence, so a hard kill loses at most one window of counters.

## Privacy

- No API key material is stored; only the `SHA-256(key)` digest.
- Prompts, request bodies and response bodies are never read, and client IP addresses and upstream response headers are never persisted.

## Management API

The page is served by the host at `/v0/resource/plugins/cpa-secret-manager/keys`; every other route requires the management key.

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/v0/management/api-keys` | read the proxy key list (host route) |
| PUT | `/v0/management/api-keys` | replace the proxy key list (host route) |
| GET / PUT | `/v0/management/plugins/cpa-secret-manager/settings` | read / update plugin settings |
| POST | `/v0/management/plugins/cpa-secret-manager/resolve` | resolve remarks and usage for a submitted key list |
| PUT | `/v0/management/plugins/cpa-secret-manager/remarks` | store a remark (an empty string clears it) |
| POST | `/v0/management/plugins/cpa-secret-manager/keys/generate` | generate one official-format key |

## Local development

Develop without a real CPA instance:

```bash
go run ./cmd/devserver
```

Then open <http://localhost:8080/control-panel>. The simulator provides the management API, a same-origin iframe shell and usage injection; the shell draws no controls of its own, so **what you see is the plugin page exactly as a real host would render it** — switch theme and language from the console with `setTheme(...)` / `setLanguage(...)` and seed demo data with `seedDemo(3)`, see [`cmd/devserver/README.md`](cmd/devserver/README.md).

## Building

Go plus CGO (a C compiler for the target platform) is required:

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o cpa-secret-manager.so .
```

## Quality gates

```bash
golangci-lint run --timeout=5m
go test ./...
go test -race ./...
```

## Documentation

- Domain glossary: [`CONTEXT.md`](CONTEXT.md)
- Architecture decisions: [`docs/adr/`](docs/adr/)
- Roadmap: [`docs/requirements/v1.0.0-roadmap.md`](docs/requirements/v1.0.0-roadmap.md)
- Release checklist: [`docs/release-checklist.md`](docs/release-checklist.md)

## License

MIT
