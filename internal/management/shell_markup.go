package management

import (
	"strings"

	"cpa-secret-manager/internal/version"
)

// shellMarkupTemplate is the page chrome shared by every tab. Handlers are bound
// by id in the page runtime, never inline, so the page keeps a single script
// scope. The version badge is injected from the shared version constant.
//
// The icons are inline SVG: the page ships no external assets, and a glyph next
// to a label keeps the toolbar readable in both languages.
const shellMarkupTemplate = `<div class="app">
  <header class="app-header">
    <div class="topbar">
      <div class="brand">
        <span class="brand-mark" aria-hidden="true"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="7.5" cy="15.5" r="4.5"></circle><path d="M10.7 12.3 20 3"></path><path d="M16.5 6.5 20 10"></path></svg></span>
        <h1 data-i18n="app.title">API Key Manager</h1>
        <span class="version-badge">v{{VERSION}}</span>
      </div>
      <div class="topbar-actions">
        <button id="lang-toggle" class="btn btn-secondary" type="button" data-i18n-title="shell.language_toggle">中文 / EN</button>
        <button id="key-button" class="btn btn-secondary" type="button" data-i18n-title="shell.management_key">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="7.5" cy="15.5" r="4.5"></circle><path d="M10.7 12.3 20 3"></path><path d="M16.5 6.5 20 10"></path></svg>
          <span data-i18n="shell.management_key">Management key</span>
        </button>
        <button id="refresh-button" class="btn btn-secondary" type="button" data-i18n-title="shell.refresh">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.5 9a9 9 0 0 1 14.9-3.4L23 10M1 14l4.6 4.4A9 9 0 0 0 20.5 15"></path></svg>
          <span data-i18n="shell.refresh">Refresh</span>
        </button>
      </div>
    </div>
    <nav class="tabs" id="tabs" role="tablist">
      <button id="tab-keys" class="tab active" type="button" role="tab" aria-selected="true">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="7.5" cy="15.5" r="4.5"></circle><path d="M10.7 12.3 20 3"></path><path d="M16.5 6.5 20 10"></path></svg>
        <span data-i18n="tab.keys">Keys</span>
      </button>
      <button id="tab-settings" class="tab" type="button" role="tab" aria-selected="false">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3"></path><path d="M1 14h6M9 8h6M17 16h6"></path></svg>
        <span data-i18n="tab.settings">Settings</span>
      </button>
      <button id="tab-help" class="tab" type="button" role="tab" aria-selected="false">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"></circle><path d="M9.1 9a3 3 0 0 1 5.8 1c0 2-2.9 3-2.9 3"></path><path d="M12 17h.01"></path></svg>
        <span data-i18n="tab.help">Help</span>
      </button>
    </nav>
  </header>
  <main>
`

// shellModals closes the page chrome and declares the shared overlays.
const shellModals = `  </main>
  <p id="status-bar" class="status-bar"></p>
</div>

<div id="key-modal" class="modal-backdrop">
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="key-modal-title">
    <h2 id="key-modal-title" data-i18n="key_modal.title">Management key</h2>
    <p class="hint" data-i18n="key_modal.hint">The key stays in this browser tab.</p>
    <div class="field">
      <label for="key-modal-input" data-i18n="key_modal.field">Management key</label>
      <input id="key-modal-input" type="password" autocomplete="off" spellcheck="false">
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
    <form id="key-form" novalidate>
      <div class="field" id="key-form-value-field">
        <label for="key-form-value" data-i18n="key_form.value">API key</label>
        <div class="field-row">
          <input id="key-form-value" type="text" autocomplete="off" spellcheck="false" required>
          <button id="key-form-generate" class="btn btn-secondary" type="button" data-i18n="key_form.generate">Generate</button>
        </div>
      </div>
      <div class="field" id="key-form-chip-field" hidden>
        <span class="field-label" data-i18n="key_form.key_label">Key</span>
        <code id="key-form-key" class="key-value">—</code>
      </div>
      <div class="field">
        <label for="key-form-remark" data-i18n="key_form.remark">Remark</label>
        <textarea id="key-form-remark" maxlength="200" data-i18n-placeholder="key_form.remark_placeholder"></textarea>
        <p class="hint" data-i18n="key_form.remark_hint">Optional; empty clears the remark.</p>
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

<div id="toast" class="toast" role="status" aria-live="polite"></div>
`

// shellMarkup is the chrome markup with the version badge resolved.
var shellMarkup = strings.Replace(shellMarkupTemplate, "{{VERSION}}", version.PluginVersion, 1)
