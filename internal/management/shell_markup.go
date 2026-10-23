package management

import (
	"strings"

	"cpa-secret-manager/internal/version"
)

// shellMarkupTemplate is the page chrome shared by every tab. Handlers are bound
// by id in the page runtime, never inline, so the page keeps a single script
// scope. Every glyph comes from the shared icon table and every control from the
// shared component layer, both resolved from ComponentsCSS/ComponentsScript.
//
// The markup is pre-processed by resolveMarkup: {{ICON:name}} and the component
// placeholders below are expanded while the page is assembled, so a typo fails
// loudly instead of shipping an empty control.
const shellMarkupTemplate = `<div class="app">
  <header class="app-header">
    <div class="topbar">
      <div class="brand">
        <span class="brand-mark" aria-hidden="true">{{ICON:key}}</span>
        <h1 data-i18n="app.title">API Key Manager</h1>
        <span class="version-badge">v{{VERSION}}</span>
      </div>
      <div class="topbar-actions">
        <button id="lang-toggle" class="btn btn-secondary" type="button" data-i18n-title="shell.language_toggle">中文 / EN</button>
        <button id="key-button" class="btn btn-secondary" type="button" data-i18n-title="shell.management_key">
          {{ICON:key}}
          <span data-i18n="shell.management_key">Management key</span>
        </button>
        <button id="refresh-button" class="btn btn-secondary" type="button" data-i18n-title="shell.refresh">
          {{ICON:refresh}}
          <span class="spinner" aria-hidden="true"></span>
          <span data-i18n="shell.refresh">Refresh</span>
        </button>
      </div>
    </div>
    <nav class="tabs" id="tabs" role="tablist">
      <button id="tab-keys" class="tab active" type="button" role="tab" aria-selected="true">
        {{ICON:key}}
        <span data-i18n="tab.keys">Keys</span>
      </button>
      <button id="tab-settings" class="tab" type="button" role="tab" aria-selected="false">
        {{ICON:sliders}}
        <span data-i18n="tab.settings">Settings</span>
      </button>
      <button id="tab-help" class="tab" type="button" role="tab" aria-selected="false">
        {{ICON:help}}
        <span data-i18n="tab.help">Help</span>
      </button>
    </nav>
  </header>
  <main>
`

// shellModals closes the page chrome and declares the shared overlays: the
// management key dialog, the add/edit key dialog, the confirm dialog and the two
// records dialogs.
const shellModals = `  </main>
  <p id="status-bar" class="status-bar"></p>
</div>

<div id="key-modal" class="modal-backdrop">
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="key-modal-title">
    <h2 id="key-modal-title" data-i18n="key_modal.title">Management key</h2>
    <p class="hint" data-i18n="key_modal.hint">The key stays in this browser tab.</p>
    <div class="field">
      <label for="key-modal-input" data-i18n="key_modal.field">Management key</label>
      <input id="key-modal-input" type="password" autocomplete="new-password" spellcheck="false" data-lpignore="true" data-1p-ignore data-bwignore>
    </div>
    <div class="modal-actions">
      <button id="key-modal-cancel" class="btn btn-secondary" type="button" data-i18n="shell.cancel">Cancel</button>
      <button id="key-modal-save" class="btn btn-primary" type="button" data-i18n="shell.save">Save</button>
    </div>
  </div>
</div>

<div id="key-form-modal" class="modal-backdrop">
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="key-form-title">
    <h2 id="key-form-title" data-i18n="key_form.add_title">Add API key</h2>
    <form id="key-form" novalidate autocomplete="off" data-bwignore="true" data-lpignore="true" data-1p-ignore>
      <div class="field" id="key-form-value-field">
        <label for="key-form-value" data-i18n="key_form.value">API key</label>
        <div class="field-row">
          <input id="key-form-value" type="text" autocomplete="off" spellcheck="false" readonly required placeholder="Generating key..." data-bwignore="true" data-lpignore="true" data-1p-ignore>
          <input id="key-form-raw-value" type="hidden">
          <button id="key-form-generate" class="btn btn-secondary" type="button" data-i18n="key_form.generate">Regenerate</button>
        </div>
      </div>
      <div class="field">
        <label for="key-form-remark" data-i18n="key_form.remark">Remark</label>
        <textarea id="key-form-remark" maxlength="200" data-i18n-placeholder="key_form.remark_placeholder" autocomplete="off" spellcheck="false" data-bwignore="true" data-lpignore="true" data-1p-ignore></textarea>
        <p class="hint" data-i18n="key_form.remark_hint">Optional; empty clears the remark.</p>
      </div>
      <div class="quota-card">
        <div class="quota-switch-header">
          <div class="switch-text">
            <span class="switch-label" data-i18n="key_form.quota_enable">Enable limit rules for this key</span>
            <span class="hint" data-i18n="key_form.quota_enable_hint">Restrict window tokens and allowed models</span>
          </div>
          {{TOGGLE_QUOTA}}
        </div>
        <div id="key-form-quota-fields" class="quota-fields" hidden>
          <div class="field-row">
            <div class="field flex-1">
              <span class="field-label" id="key-form-quota-window-label" data-i18n="key_form.quota_window">Time window period</span>
              <div id="key-form-quota-window"></div>
            </div>
            <div class="field flex-1" id="key-form-quota-duration-field" hidden>
              <label for="key-form-quota-duration" data-i18n="key_form.quota_duration_sec">Window duration</label>
              <div class="input-unit-group">
                <input id="key-form-quota-duration" class="unit-input" type="number" min="1" step="1" value="3600" inputmode="numeric">
                <div id="key-form-quota-duration-unit"></div>
              </div>
              <input id="key-form-quota-duration-sec" type="hidden" value="3600">
            </div>
          </div>
          <div class="field">
            <span class="field-label" data-i18n="key_form.quota_mode">Limit Rules</span>
            <div class="quota-mode-toggles">
              <button type="button" class="quota-mode-card is-active" id="key-form-quota-mode-token" role="checkbox" aria-checked="true">
                <span class="quota-mode-indicator"></span>
                <span class="quota-mode-label" data-i18n="key_form.quota_mode_tokens">Limit Total Tokens</span>
              </button>
              <button type="button" class="quota-mode-card" id="key-form-quota-mode-cost" role="checkbox" aria-checked="false">
                <span class="quota-mode-indicator"></span>
                <span class="quota-mode-label" data-i18n="key_form.quota_mode_cost">Limit Total Cost</span>
              </button>
            </div>
            <p id="key-form-quota-dual-hint" class="hint text-accent" data-i18n="key_form.quota_dual_hint" hidden>Dual rules active: requests blocked if either tokens or cost exceed the limit.</p>
          </div>
          <div class="field-row" id="key-form-quota-limits-row">
            <div id="key-form-quota-token-section" class="field flex-1">
              <label for="key-form-quota-limit-val" data-i18n="key_form.quota_limit">Token Limit</label>
              <div class="input-unit-group">
                <input id="key-form-quota-limit-val" class="unit-input" type="number" min="1" step="any" inputmode="decimal" placeholder="0">
                <div id="key-form-quota-limit-unit"></div>
              </div>
              <input id="key-form-quota-limit" type="hidden" value="0">
            </div>
            <div id="key-form-quota-cost-section" class="field flex-1" hidden>
              <label for="key-form-quota-cost-val" data-i18n="key_form.quota_cost_limit">Cost Limit ($)</label>
              <div class="input-unit-group">
                <span class="unit-prefix">$</span>
                <input id="key-form-quota-cost-val" class="unit-input" type="number" min="0.001" step="any" inputmode="decimal" placeholder="0.000">
                <span class="unit-suffix">/ M</span>
              </div>
            </div>
          </div>
          <div class="field">
            <div class="row-between">
              <span class="field-label" data-i18n="key_form.quota_models">Allowed Models</span>
              <span class="action-row">
                <button id="key-form-models-all" class="btn-text" type="button" data-i18n="key_form.quota_select_all">Select all</button>
                <button id="key-form-models-none" class="btn-text" type="button" data-i18n="key_form.quota_clear_all">Clear</button>
              </span>
            </div>
            <p id="key-form-quota-models-hint" class="hint" data-i18n="key_form.quota_models_hint">Unchecked allows all available models</p>
            <div id="key-form-models-container"></div>
            <div id="key-form-quota-no-priced-alert" class="alert is-warning" hidden>
              {{ICON:warning}}
              <span data-i18n="key_form.quota_no_priced_models">No models have configured prices yet. Please add model pricing in Settings first.</span>
            </div>
          </div>
        </div>
      </div>
      <div class="modal-actions">
        <button id="key-form-cancel" class="btn btn-secondary" type="button" data-i18n="shell.cancel">Cancel</button>
        <button id="key-form-save" class="btn btn-primary" type="submit" data-i18n="shell.save">Save</button>
      </div>
    </form>
  </div>
</div>

<div id="confirm-modal" class="modal-backdrop">
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
    <h2 id="confirm-title" data-i18n="confirm.title">Please confirm</h2>
    <p id="confirm-message" class="hint"></p>
    <div class="modal-actions">
      <button id="confirm-cancel" class="btn btn-secondary" type="button" data-i18n="shell.cancel">Cancel</button>
      <button id="confirm-accept" class="btn btn-danger" type="button" data-i18n="shell.confirm">Confirm</button>
    </div>
  </div>
</div>

<div id="failures-modal" class="modal-backdrop">
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="failures-title">
    <div class="row-between modal-header-row">
      <h2 id="failures-title" data-i18n="failures.title">Failure records & diagnostics</h2>
      <span id="failures-range-tag" class="failures-range-tag"></span>
    </div>
    <div id="failures-content"></div>
    <div class="modal-actions">
      <button id="failures-close" class="btn btn-secondary" type="button" data-i18n="shell.cancel">Close</button>
      <button id="failures-open-logs" class="btn btn-primary" type="button" data-i18n="failures.open_logs">Open Host Logs</button>
    </div>
  </div>
</div>

<div id="date-range-modal" class="modal-backdrop">
  <div class="date-range-popover" role="dialog" aria-modal="true" aria-labelledby="date-range-title">
    <div class="date-dialog-body">
      <div class="date-dialog-head">
        <div class="date-dialog-heading">
          <h2 id="date-range-title" class="date-dialog-title" data-i18n="range.calendar_title">Select date range</h2>
          <div id="date-selection-status" class="date-selection-status"></div>
        </div>
        <div class="action-row">
          <button class="pill" type="button" data-preset="last_5_hours" data-i18n="range.last_5_hours">Last 5 hours</button>
          <button class="pill" type="button" data-preset="last_7_days" data-i18n="range.last_7_days">Last 7 days</button>
          <button class="pill" type="button" data-preset="last_30_days" data-i18n="range.last_30_days">Last 30 days</button>
          <button class="pill" type="button" data-preset="this_month" data-i18n="range.this_month">This month</button>
        </div>
      </div>
      <div class="calendar-panels">
        <div class="calendar-panel" id="calendar-left"></div>
        <div class="calendar-panel" id="calendar-right"></div>
      </div>
      <div class="time-pickers-row">
        <div class="time-picker-col">
          <label class="time-picker-label" for="time-start" data-i18n="range.start_time">Start time</label>
          <input type="text" id="time-start" class="time-input" value="00:00:00" placeholder="00:00:00" maxlength="8">
        </div>
        <div class="time-picker-col">
          <label class="time-picker-label" for="time-end" data-i18n="range.end_time">End time</label>
          <input type="text" id="time-end" class="time-input" value="23:59:59" placeholder="23:59:59" maxlength="8">
        </div>
      </div>
      <div class="date-dialog-hint" data-i18n="range.hint">Includes start time, excludes end time</div>
      <div class="date-dialog-footer">
        <button id="date-range-cancel" class="btn btn-secondary" type="button" data-i18n="shell.cancel">Cancel</button>
        <button id="date-range-reset" class="btn btn-secondary" type="button" data-i18n="range.reset_today">Reset to today</button>
        <button id="date-range-confirm" class="btn btn-primary" type="button" data-i18n="range.confirm">Confirm</button>
      </div>
    </div>
  </div>
</div>

<div id="toast" class="toast" role="status" aria-live="polite"></div>
<div id="global-loading" class="global-loading" hidden>
  <div class="global-loading-box">
    <span class="spinner global-spinner" aria-hidden="true"></span>
    <span id="global-loading-text" class="global-loading-text" data-i18n="shell.loading">Loading...</span>
  </div>
</div>
`

// shellMarkup is the chrome markup with the version badge resolved. The
// component placeholders are expanded while the page is assembled.
var shellMarkup = strings.Replace(shellMarkupTemplate, "{{VERSION}}", version.PluginVersion, 1)
