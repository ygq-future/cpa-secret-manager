<div align="center">

# CPA Secret Manager (`cpa-secret-manager`)

[中文](./README.md) | [English](./README.en.md)

</div>

A proxy API key management and usage attribution plugin for [CLIProxyAPI (CPA)](https://github.com/router-for-me/CLIProxyAPI). The plugin ID, dynamic library basename, and CPA configuration key are all `cpa-secret-manager`.

It provides **per-key remarks** and **per-key token usage attribution**, covering the two capabilities the official management center lacks: human annotations and fine-grained per-key usage visibility.

---

## Navigation

- [Overview](#overview)
- [Workflow](#workflow)
- [Build and Installation](#build-and-installation)
- [Plugin Store Source](#plugin-store-source)
- [Configuration](#configuration)
- [Usage Attribution Semantics](#usage-attribution-semantics)
- [Management Page and API](#management-page-and-api)
- [Local Development](#local-development)
- [License](#license)

---

## Overview

- **Keys Owned by Host, Metadata Owned by Plugin**: API key CRUD operations fully reuse the host `/v0/management/api-keys` API, ensuring no second source of truth. The plugin maintains remarks and token usage indexed solely by `SHA-256(key)` digests—**never persisting or revealing plain key material**.
- **1:1 Official Key Generation Algorithm**: Generated keys use an `sk-` prefix plus 48 alphanumeric characters `[A-Za-z0-9]`, with modulo bias removed through rejection sampling (threshold 248), matching the official management console algorithm.
- **Full-Dimensional Token Attribution**: Intercepts requests through the host `usage_plugin` capability, aggregating request counts, failure counts, and Input/Output tokens, as well as Reasoning, Cached, and CacheRead token breakdowns grouped by model.
- **Automatic Orphan Metadata Pruning**: Implements the `/forget` protocol; when keys are deleted from the host, the plugin detects and safely purges corresponding orphaned metadata and usage counters.
- **Deep CPA Integration & Native Feel**: The embedded management page is completely self-contained (zero external CDNs, zero inline event handlers). It dynamically extracts host CSS variables via same-origin iframe to follow CPA themes (light, white, dark, and custom themes) and UI language (`cli-proxy-language`).
- **Minimalist Host Configuration**: The host `config.yaml` only requires `enabled: true`. Business state is stored in an independent cache document, and metadata `ConfigFields` remains empty to prevent drawer write-backs from triggering unnecessary host reloads.

---

## Workflow

```text
Client Request to CPA
  │
  ▼
CLIProxyAPI Host Routing
  │
  ├─► [usage_plugin Interception]
  │     └─ Extract client proxy API key serving the request
  │     └─ usage.Aggregator accumulates metrics concurrently (requests/failures/tokens/models)
  │     └─ 5-second throttled batch disk flush (forced atomic flush on shutdown)
  │
  └─► [management_api Coordination]
        ├─ Read/Write Keys  ──► Reuses host /v0/management/api-keys (Authoritative Source)
        ├─ Remarks & Usage  ──► Plugin /resolve endpoint returns SHA-256 correlated metadata
        ├─ Prune Stale Keys ──► Plugin /forget endpoint destroys orphaned metadata
        └─ Embedded Web UI  ──► Same-origin iframe reuses cli-proxy-auth, following theme & language
```

---

## Build and Installation

The plugin runs as a CGO dynamic library. CPA derives the plugin ID from the dynamic library filename, so the filename must stay `cpa-secret-manager.<ext>`.

### Local Compilation

Requires a local Go installation (Go `go1.26.7` recommended, matching CI toolchain) and a platform-specific C compiler (GCC / Clang / MinGW):

```bash
# Linux / macOS
CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o cpa-secret-manager.so .

# Windows (MSYS2 / MinGW-w64)
CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o cpa-secret-manager.dll .
```

### Installation in CPA

Place the compiled dynamic library into one of the CPA plugin discovery directories:

- `plugins/<GOOS>/<GOARCH>/cpa-secret-manager.<ext>`
- `plugins/<GOOS>/<GOARCH>-<variant>/cpa-secret-manager.<ext>`
- `plugins/cpa-secret-manager.<ext>`

Extensions by operating system:
- **Linux / FreeBSD**: `.so`
- **macOS**: `.dylib`
- **Windows**: `.dll`

---

## Plugin Store Source

To install this plugin through the CPA Plugin Store with one click, add the raw registry URL to your CPA `config.yaml`:

```yaml
plugins:
  enabled: true
  store-sources:
    - "https://raw.githubusercontent.com/ygq-future/cpa-secret-manager/main/registry.json"
```

> **Note**:
> - You must use the raw URL; do not use GitHub web URLs containing `/blob/`.
> - After saving `config.yaml`, restart CPA or reload plugins through the management UI, then refresh the plugin store list to install with architecture auto-detection.

---

## Configuration

### Minimalist CPA Host Configuration (`config.yaml`)

It is recommended to keep only the basic plugin enablement switch in CPA's `config.yaml`. All operational state is managed internally by the plugin:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpa-secret-manager:
      enabled: true
      priority: 1
      # state_path: "data/cpa-secret-manager-cache.json" # Optional custom cache path
```

| Host Parameter | Default | Required | Description |
| :--- | :--- | :--- | :--- |
| **`enabled`** | `true` | Yes | Global plugin switch. When `false`, all routes and usage interception are deactivated. |
| **`priority`** | `1` | No | Execution priority in the CPA plugin pipeline. |
| **`state_path`** | `data/cpa-secret-manager-cache.json` | No | File path for persisting remarks and aggregated usage data. |

> **Persistence Guarantee**: State updates write through `*.tmp` temporary files followed by atomic `os.Rename`, locked to `0600` file permissions. Deleting the cache file completely resets all remarks and historical usage metrics.

---

## Usage Attribution Semantics

- **Attribution Source**: Directly fed by `UsageRecord` payloads dispatched by `usage_plugin`, grouped by the SHA-256 digest of the client proxy key.
- **Cumulative Totals**: Metrics represent **all-time cumulative values** since the first request observed by the plugin; there are no sliding time windows and no cost estimations.
- **Error Accounting**: Requests returning upstream errors or zero tokens are accurately accounted for in request and failure counters (tokens counted as 0).
- **Unmanaged Keys Discarded**: Only keys currently registered in the CPA host's managed list are tracked; blank or unmanaged requests are discarded without polluting the dashboard.
- **Throttled Persistence**: In-memory aggregations are protected by concurrent mutex locks, written to disk on a 5-second throttled cadence, and synchronously force-flushed upon graceful process shutdown.

---

## Management Page and API

The plugin registers **static Web UI resources** and **dynamic management routes** with the host via `management.register`.

### Static Web Resource (No Separate Auth Required)

- `GET /v0/resource/plugins/cpa-secret-manager/keys`
  - **Access**: Click **"API Key Manager"** in the CPA management sidebar, or access same-origin in your browser.
  - **Key Features**:
    - **Keys & Remarks Dashboard**: Masked by default, with one-click copy, real-time key/remark filtering, and multi-unit (K / M / B) token scaling;
    - **Per-Model Breakdown**: Expandable inspection of input, output, reasoning, and cache token distribution across models;
    - **Pure Native Visuals**: Zero external CDNs, dynamically following CPA themes (light, white, dark, and custom themes);
    - **Interaction Tolerance**: Two-phase mousedown/click verification prevents modal backdrop dismissals when dragging text selections across borders.

### Dynamic Management API (Management Key Required)

All management endpoints require `Authorization: Bearer <management_key>` or `X-Management-Key: <management_key>` in request headers:

| Method | Path | Description |
| :--- | :--- | :--- |
| **`GET`** | `/v0/management/api-keys` | Reads all currently saved proxy keys from the host (host endpoint) |
| **`PUT`** | `/v0/management/api-keys` | Saves the full proxy key list to the host (host endpoint) |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/keys/generate` | Generates a cryptographically secure proxy API key (`sk-` + 48 characters) |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/resolve` | Submits key list and returns matching remarks, usage, and stale hashes |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/remarks` | Sets or clears a remark for a specific key (empty string clears) |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/forget` | Instructs the plugin to permanently purge metadata for stale key hashes |
| **`GET`** | `/v0/management/plugins/cpa-secret-manager/settings` | Reads runtime settings (active cache path, usage toggle status, warnings) |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/settings` | Updates plugin configuration flags (e.g. enabling/disabling usage attribution) |

---

## Local Development

The project includes a high-fidelity local host simulator (`cmd/devserver`) to inspect the embedded UI, API responses, and usage streams without running a full CPA instance:

```bash
# Start local host simulator (defaults to 127.0.0.1:8080)
go run ./cmd/devserver
```

Open `http://127.0.0.1:8080/control-panel` to inspect 1:1 theme following, language synchronization, and usage views in a same-origin iframe container. See [`cmd/devserver/README.md`](cmd/devserver/README.md) for full instructions.

---

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE).
