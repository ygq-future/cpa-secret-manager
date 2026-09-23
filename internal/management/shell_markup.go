package management

import (
	"strings"

	"cpa-secret-manager/internal/version"
)

// shellMarkupTemplate is the page chrome shared by every tab. Handlers are bound
// by id in the page runtime, never inline, so the page keeps a single script
// scope. The version badge is injected from the shared version constant.
const shellMarkupTemplate = `<div class="app">
  <header class="topbar">
    <div class="brand">
      <h1 data-i18n="app.title">API Key Manager</h1>
      <span class="version-badge">v{{VERSION}}</span>
    </div>
    <div class="topbar-actions">
      <button id="lang-toggle" class="btn btn-ghost" type="button" data-i18n-title="shell.language_toggle">中文 / EN</button>
      <button id="key-button" class="btn btn-ghost" type="button" data-i18n="shell.management_key">Management key</button>
      <button id="refresh-button" class="btn" type="button" data-i18n="shell.refresh">Refresh</button>
    </div>
  </header>
  <nav class="tabs" id="tabs">
    <button id="tab-keys" class="tab active" type="button" data-i18n="tab.keys">Keys</button>
    <button id="tab-settings" class="tab" type="button" data-i18n="tab.settings">Settings</button>
    <button id="tab-help" class="tab" type="button" data-i18n="tab.help">Help</button>
  </nav>
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
      <button id="key-modal-cancel" class="btn btn-ghost" type="button" data-i18n="shell.cancel">Cancel</button>
      <button id="key-modal-save" class="btn" type="button" data-i18n="shell.save">Save</button>
    </div>
  </div>
</div>

<div id="key-form-modal" class="modal-backdrop">
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="key-form-title">
    <h2 id="key-form-title" data-i18n="key_form.add_title">Add API key</h2>
    <form id="key-form">
      <div class="field">
        <label for="key-form-value" data-i18n="key_form.value">API key</label>
        <div class="field-row">
          <input id="key-form-value" type="text" autocomplete="off" spellcheck="false" required>
          <button id="key-form-generate" class="btn btn-ghost" type="button" data-i18n="key_form.generate">Generate</button>
        </div>
      </div>
      <div class="field">
        <label for="key-form-remark" data-i18n="key_form.remark">Remark</label>
        <textarea id="key-form-remark" maxlength="200" data-i18n-placeholder="key_form.remark_placeholder"></textarea>
        <p class="hint" data-i18n="key_form.remark_hint">Optional; empty clears the remark.</p>
      </div>
      <div class="modal-actions">
        <button id="key-form-cancel" class="btn btn-ghost" type="button" data-i18n="shell.cancel">Cancel</button>
        <button id="key-form-save" class="btn" type="submit" data-i18n="shell.save">Save</button>
      </div>
    </form>
  </div>
</div>

<div id="confirm-modal" class="modal-backdrop">
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
    <h2 id="confirm-title" data-i18n="confirm.title">Please confirm</h2>
    <p id="confirm-message" class="hint"></p>
    <div class="modal-actions">
      <button id="confirm-cancel" class="btn btn-ghost" type="button" data-i18n="shell.cancel">Cancel</button>
      <button id="confirm-accept" class="btn btn-danger" type="button" data-i18n="shell.confirm">Confirm</button>
    </div>
  </div>
</div>

<div id="toast" class="toast" role="status" aria-live="polite"></div>
`

// shellMarkup is the chrome markup with the version badge resolved.
var shellMarkup = strings.Replace(shellMarkupTemplate, "{{VERSION}}", version.PluginVersion, 1)
