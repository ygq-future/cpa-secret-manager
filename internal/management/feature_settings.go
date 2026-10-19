package management

// settingsFeature renders the plugin business settings and state diagnostics.
// Every control comes from the shared component layer: the usage switch is the
// shared toggle, the retention field is the shared number+unit input.
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
                {{TOGGLE_USAGE}}
              </div>
            </div>
            <div class="switch-row">
              <div class="switch-text">
                <span class="switch-label" data-i18n="settings.retention_title">Data retention</span>
                <span class="hint" data-i18n="settings.retention_hint">Retention period for usage and error data in days (1-365). Older records are pruned.</span>
              </div>
              <div class="switch-control">
                <div class="input-unit-group">
                  <input id="settings-retention-input" class="unit-input" type="number" min="1" max="365" step="1" inputmode="numeric" data-i18n-aria-label="settings.retention_title">
                  <span class="unit-suffix" data-i18n="settings.days"></span>
                </div>
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
              {{ICON:warning}}
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

function bindSettingsEvents() {
  bindClick('settings-usage-toggle', toggleUsageEnabled);
  bindClick('settings-retention-save', saveRetentionDays);
  var retInput = byId('settings-retention-input');
  if (retInput) {
    retInput.addEventListener('keydown', function (event) {
      if (event.key === 'Enter') {
        saveRetentionDays();
      }
    });
  }
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
            <div class="action-row">
              <a href="https://github.com/ygq-future/cpa-secret-manager" target="_blank" rel="noopener noreferrer" class="btn btn-secondary">
                {{ICON:github}}
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
