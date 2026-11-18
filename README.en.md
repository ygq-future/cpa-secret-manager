<div align="center">

# CPA Secret Manager (`cpa-secret-manager`)

[中文](./README.md) | [English](./README.en.md)

</div>

A proxy API key management and usage attribution plugin for [CLIProxyAPI (CPA)](https://github.com/router-for-me/CLIProxyAPI). The plugin ID, dynamic library basename, and CPA configuration key are all `cpa-secret-manager`.

It provides **per-key remarks**, **per-key token usage attribution**, and **quota / model-allowlist enforcement**, covering the capabilities the official management center lacks: human annotations, fine-grained usage visibility, and request boundaries.

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
- **Per-Key Quotas, Independent Model Restrictions & Cost Budgets**: Set token limits, monetary cost budgets (Quota Cost Limit), or independent model allowlists (restricting models without requiring token/cost limits) per key with automatic rejection on violation and one-click quota resets; configure custom token pricing per model with automatic expenditure estimation. The allowlist offers exactly the host `/v1/models` catalog; when an allowlisted model is removed from the host, the key row carries an “N removed” warning badge and the form marks those models as removed with a consolidated notice, while prices no longer referenced by any allowlist are dropped on sync.
- **Key Expiry with Automatic Cleanup**: Set an expiration instant per key from local wall-clock input and store it in UTC. Once reached, the plugin returns `401 key_expired` before the request reaches an upstream, whether or not the page is open; the management page removes the expired key from the host key list on its next refresh or poll and drops its remark, expiry, and usage counters.
- **Model Reference Price Suggestions**: “Sync & save prices” in Settings pulls reference prices from models.dev and merges them automatically: it adds missing models, updates changed prices, and keeps saved models absent from the reference. A saved price that still matches any listed offer is kept even when the applied offer disappeared; otherwise the nearest remaining offer is applied, and a model with no offer keeps its saved price. Model names match by progressive prefix stages (`gemini` → `gemini-3.5` → `gemini-3.5-flash`) and fall back to the deepest stage that matched; matching also skips vendor/channel prefixes and ignores separator differences in `-`, `_`, and spaces (saved names like `wb-gemini-flash` or `gemini3.5flash` still resolve to the catalog base model), so channel suffixes such as `-high` still resolve to the base model. The reference row lists every matched offer in a dropdown with input/output/cache amounts, marks the best match in green, and shows the full offer on hover. A full sync — including a 304 response for an unchanged catalog — drops cached entries for models no longer in the host list. “Sync only” and the refresh button at the end of the reference row update the cache without writing saved prices; a single-model refresh preserves every other cached model. Network access happens only on click, and the streamed catalog keeps only matches for the current model list, never the full catalog.
- **Range Dashboard and Failure Triage**: A single date range drives the stat cards, per-key columns, and model breakdown, including each key's last-seen time. Failure records are grouped into a timeline with upstream-credential, model, and HTTP status filtering that cascades — each dropdown lists only the values still reachable under the other selections, and a selection that stops matching is dropped — and the upstream credential that served the request is shown (keeping the provider prefix and account domain while masking the account body), for locating abnormal keys and upstream errors. A cutoff picker clears older failure logs within the active filters and deducts the matching failure counters, so dashboard statistics stay aligned with the remaining records.
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
  ├─► [request.intercept_before Quota Gate]
  │     ├─ Key already expired ──► return 401 before upstream
  │     └─ Read the client key's token quota and model allowlist
  │     └─ Exceeded or disallowed model ──► return 403 before upstream
  │
  ├─► [usage_plugin Interception]
  │     └─ Extract client proxy API key serving the request
  │     └─ usage.Aggregator folds UTC hourly buckets (requests/failures/tokens/models)
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
      # state_path: "data/cpa-secret-manager/state.json" # Optional state document inside the plugin data directory
```

| Host Parameter | Default | Required | Description |
| :--- | :--- | :--- | :--- |
| **`enabled`** | `true` | Yes | Global plugin switch. When `false`, all routes and usage interception are deactivated. |
| **`priority`** | `1` | No | Execution priority in the CPA plugin pipeline. |
| **`state_path`** | `data/cpa-secret-manager/state.json` | No | State document inside the plugin data directory; the same directory always holds `buckets.json` (hourly usage buckets), `errors.json` (failure records), and `prices.json` (reference-price cache). |

> **Persistence Guarantee**: All four documents (state, usage buckets, failure records, reference-price cache) write through `*.tmp` temporary files followed by atomic `os.Rename`, locked to `0600` file permissions. Deleting them completely resets remarks, usage, failure history, and reference prices. Legacy `data/cpa-secret-manager-cache*.json` files are moved into the new directory on startup without a backup.

---

## Usage Attribution Semantics

- **Attribution Source**: Directly fed by `UsageRecord` payloads dispatched by `usage_plugin`, grouped by the SHA-256 digest of the client proxy key.
- **Hourly Buckets**: Usage is stored in **UTC hourly buckets** (key × model × hour) and the page aggregates whichever range is selected. The range reaches back at most the configured **data retention days** (1-365, default 30); older buckets are pruned on every flush. No cost estimation and no trend charts.
- **Quota Windows**: Natural day, natural week (Monday 00:00 UTC), and natural month (1st 00:00 UTC) align to UTC; fixed-duration windows roll by the configured number of seconds. Requests are terminated with `403` before upstream once the token limit is reached or the model is outside the allowlist.
- **Last Seen**: Each key records the exact instant of its latest request within retention, independent of the selected date range. Legacy buckets without a precise stamp fall back to the bucket start.
- **Error Accounting**: Requests returning upstream errors or zero tokens are accurately accounted for in request and failure counters (tokens counted as 0).
- **Unmanaged Keys Discarded**: Only keys currently registered in the CPA host's managed list are tracked; blank or unmanaged requests are discarded without polluting the dashboard.
- **Throttled Persistence**: In-memory aggregation is protected by mutexes, flushed to disk on a 5-second cadence, and force-flushed on graceful shutdown. Buckets start accumulating at upgrade time: the previous cumulative counters carry no time attribution and are no longer displayed.

---

## Management Page and API

The plugin registers **static Web UI resources** and **dynamic management routes** with the host via `management.register`.

### Static Web Resource (No Separate Auth Required)

- `GET /v0/resource/plugins/cpa-secret-manager/keys`
  - **Access**: Click **"API Key Manager"** in the CPA management sidebar, or access same-origin in your browser.
  - **Key Features**:
    - **Keys & Remarks Dashboard**: Masked by default, with one-click copy, real-time key/remark filtering, and multi-unit (K / M / B) token scaling;
    - **Range Dashboard**: One date range drives the stat cards, per-key columns, model breakdown, and failure log, including each key's last-seen time;
    - **Quota Rule Editor**: Configure a token limit, window period, and model allowlist per key; exceeding either boundary blocks the request;
    - **Expiry Time**: Pick a local-time expiration instant in the date/time picker (including a “Now” shortcut and 1 hour / 1 day / 7 days / 30 days presets); the list shows the exact local minute (e.g. `26/11/14 10:40`). At expiry requests are rejected and the page removes the key from the host list on its next refresh or poll;
    - **Per-Model Breakdown**: Expandable inspection of input, output, reasoning, and cache token distribution across models;
    - **Failure Log Pruning**: Clear older failure logs from within the failure viewer by picking an instant no later than now (shortcuts for 30 days / 7 days / 24 hours / 6 hours / 1 hour ago); the scope follows the active filters and the failure counters are deducted accordingly;
    - **Pure Native Visuals**: Zero external CDNs, dynamically following CPA themes (light, white, dark, and custom themes);
    - **Interaction Tolerance**: Two-phase mousedown/click verification prevents modal backdrop dismissals when dragging text selections across borders.

### Dynamic Management API (Management Key Required)

All management endpoints require `Authorization: Bearer <management_key>` or `X-Management-Key: <management_key>` in request headers:

| Method | Path | Description |
| :--- | :--- | :--- |
| **`GET`** | `/v0/management/api-keys` | Reads all currently saved proxy keys from the host (host endpoint) |
| **`PUT`** | `/v0/management/api-keys` | Saves the full proxy key list to the host (host endpoint) |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/keys/generate` | Generates a cryptographically secure proxy API key (`sk-` + 48 characters) |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/resolve` | Submits a key list and window; returns remarks, in-window usage, lifetime `last_seen`, and stale hashes |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/remarks` | Sets or clears a remark for a specific key (empty string clears) |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/expiry` | Sets or clears a key's expiration instant (RFC3339; empty clears). At expiry requests are rejected and the page removes the key from the host list |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/forget` | Instructs the plugin to permanently purge metadata for stale key hashes |
| **`GET`** | `/v0/management/plugins/cpa-secret-manager/quota` | Reads a key's quota rule and current window consumption |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/quota` | Creates or updates a key's token limit, window period, and model allowlist |
| **`DELETE`** | `/v0/management/plugins/cpa-secret-manager/quota` | Deletes a key's quota rule |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/failures` | Queries failure records by key digest and time range, including HTTP status and upstream credential identifiers |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/failures/clear` | Clears failure records older than a cutoff instant (optionally scoped by key digest, upstream credential, model, and HTTP status), returning the removed count and deducting failure counters |
| **`GET`** | `/v0/management/plugins/cpa-secret-manager/models` | Lists models observed by the plugin |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/pricing` | Reads cached model reference prices (cache-only, no network) |
| **`POST`** | `/v0/management/plugins/cpa-secret-manager/pricing/refresh` | Explicitly syncs the models.dev catalog; by default only submitted models are kept, while `merge: true` replaces just those models and preserves other cache entries |
| **`GET`** | `/v0/management/plugins/cpa-secret-manager/settings` | Reads runtime settings (cache path, usage toggle, retention days, reference-price status, warnings) |
| **`PUT`** | `/v0/management/plugins/cpa-secret-manager/settings` | Updates the usage toggle and data retention days |

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
