package management

// settingsFeature renders the plugin business settings and state diagnostics.
var settingsFeature = pageFeature{
	markup: `
    <section id="panel-settings" class="panel">
      <div class="scroll-area">
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="settings.usage_title">Usage tracking</h2>
            <p class="card-sub" data-i18n="settings.usage_hint">Usage counters are cumulative and start with the first request the plugin observes.</p>
            <div class="switch-row">
              <div class="switch-text">
                <span class="switch-label" data-i18n="settings.usage_toggle">Collect token usage</span>
                <span class="hint" data-i18n="settings.usage_toggle_hint">Turning this off stops new accounting; existing counters stay visible.</span>
              </div>
              <div class="switch-control">
                <span id="settings-usage-state" class="toggle-state"></span>
                <label class="toggle-switch">
                  <input id="settings-usage-toggle" type="checkbox" role="switch" data-i18n-title="settings.usage_toggle" data-i18n-aria-label="settings.usage_toggle">
                  <span class="toggle-slider"></span>
                </label>
              </div>
            </div>
            <div class="switch-row" style="margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--border-color);">
              <div class="switch-text">
                <span class="switch-label" data-i18n="settings.retention_title">Data retention</span>
                <span class="hint" data-i18n="settings.retention_hint">Retention period for usage and error data in days (1-365). Older records are pruned.</span>
              </div>
              <div style="display: flex; align-items: center; gap: 8px;">
                <input id="settings-retention-input" type="number" min="1" max="365" style="width: 80px; padding: 6px 10px; border-radius: 6px; border: 1px solid var(--border-color); background: var(--bg-card); color: var(--text-primary); font-size: 14px; text-align: center;">
                <span style="font-size: 13px; color: var(--text-secondary);" data-i18n="settings.days">days</span>
                <button id="settings-retention-save" class="btn btn-secondary" type="button" data-i18n="settings.save">Save</button>
              </div>
            </div>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="settings.state_title">State</h2>
            <dl class="kv">
              <dt data-i18n="settings.state_path">State file</dt>
              <dd id="settings-state-path" class="path">—</dd>
              <dt data-i18n="settings.key_status">Management key</dt>
              <dd id="settings-key-status">—</dd>
            </dl>
            <div id="settings-warning" class="alert" hidden>
              <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"></path><path d="M12 9v4"></path><path d="M12 17h.01"></path></svg>
              <span id="settings-warning-text"></span>
            </div>
          </div>
        </div>
      </div>
    </section>
`,
	styles: ``,
	scripts: `
function loadSettings() {
  return apiFetch(SETTINGS_PATH).then(function (payload) {
    state.settings = payload || null;
    renderSettings();
  });
}

function usageEnabled() {
  if (!state.settings) {
    return true;
  }
  return state.settings.usage_enabled !== false;
}

function renderSettings() {
  var enabled = usageEnabled();
  var toggle = byId('settings-usage-toggle');
  if (toggle) {
    toggle.checked = enabled;
    toggle.setAttribute('aria-checked', enabled ? 'true' : 'false');
  }

  var stateLabel = byId('settings-usage-state');
  if (stateLabel) {
    stateLabel.textContent = enabled ? t('settings.usage_on') : t('settings.usage_off');
  }

  var path = byId('settings-state-path');
  if (path) {
    path.textContent = state.settings && state.settings.state_path ? state.settings.state_path : '—';
  }

  var keyStatus = byId('settings-key-status');
  if (keyStatus) {
    keyStatus.textContent = state.managementKey ? t('settings.key_loaded') : t('settings.key_missing');
  }


  var retInput = byId('settings-retention-input');
  if (retInput && state.settings && state.settings.retention_days) {
    retInput.value = state.settings.retention_days;
  }
  var messages = [];
  if (state.settings && state.settings.state_warning) {
    messages.push(t('settings.state_warning') + ': ' + state.settings.state_warning);
  }
  if (state.settings && state.settings.usage_warning) {
    messages.push(t('settings.usage_warning') + ': ' + state.settings.usage_warning);
  }
  if (state.settings && Array.isArray(state.settings.config_warnings)) {
    messages = messages.concat(state.settings.config_warnings);
  }

  var warning = byId('settings-warning');
  var warningText = byId('settings-warning-text');
  if (warningText) {
    warningText.textContent = messages.join(' ');
  }
  if (warning) {
    warning.hidden = messages.length === 0;
  }
}

function toggleUsageEnabled() {
  var next = !usageEnabled();
  apiFetch(SETTINGS_PATH, { method: 'PUT', body: JSON.stringify({ usage_enabled: next }) })
    .then(function () {
      return loadSettings();
    })
    .then(function () {
      showToast(t('settings.saved'), 'success');
      return refreshAll({ silent: true });
    })
    .catch(function (error) {
      showToast(error.message, 'error');
      renderSettings();
    });
}

function saveRetentionDays() {
  var input = byId('settings-retention-input');
  if (!input) return;
  var val = parseInt(input.value, 10);
  if (isNaN(val) || val < 1 || val > 365) {
    showToast(t('settings.retention_invalid'), 'error');
    return;
  }
  apiFetch(SETTINGS_PATH, { method: 'PUT', body: JSON.stringify({ retention_days: val }) })
    .then(function () {
      return loadSettings();
    })
    .then(function () {
      showToast(t('settings.saved'), 'success');
    })
    .catch(function (error) {
      showToast(error.message, 'error');
      renderSettings();
    });
}
`,
}

// helpFeature documents installation, behaviour boundaries and privacy rules.
var helpFeature = pageFeature{
	markup: `
    <section id="panel-help" class="panel">
      <div class="scroll-area">
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.install_title">Installation</h2>
            <ul class="help-list">
              <li data-i18n="help.install_body"></li>
              <li data-i18n="help.config_body"></li>
              <li data-i18n="help.version_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.keys_title">API keys</h2>
            <ul class="help-list">
              <li data-i18n="help.keys_body"></li>
              <li data-i18n="help.generate_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.usage_title">Token usage</h2>
            <ul class="help-list">
              <li data-i18n="help.usage_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.privacy_title">Privacy</h2>
            <ul class="help-list">
              <li data-i18n="help.privacy_body"></li>
              <li data-i18n="help.state_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.theme_title">Appearance</h2>
            <ul class="help-list">
              <li data-i18n="help.theme_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.repo_title">Open Source</h2>
            <p class="hint" data-i18n="help.repo_desc">Project home and source code: https://github.com/ygq-future/cpa-secret-manager</p>
            <div style="margin-top: 4px;">
              <a href="https://github.com/ygq-future/cpa-secret-manager" target="_blank" rel="noopener noreferrer" class="btn btn-secondary" style="display:inline-flex; width:fit-content; text-decoration:none; gap:6px;">
                <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path></svg>
                <span data-i18n="help.repo_link">Visit GitHub Repository</span>
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
`,
	styles:  "",
	scripts: "",
}
