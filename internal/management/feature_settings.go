package management

// settingsFeature renders the plugin business settings and state diagnostics.
var settingsFeature = pageFeature{
	markup: `
    <section id="panel-settings" class="panel">
      <div class="card">
        <h2 data-i18n="settings.usage_title">Usage tracking</h2>
        <p class="hint" data-i18n="settings.usage_hint">Usage counters are cumulative and start with the first request the plugin observes.</p>
        <div class="switch-row">
          <div>
            <div class="switch-label" data-i18n="settings.usage_toggle">Collect token usage</div>
            <p class="hint" data-i18n="settings.usage_toggle_hint">Turning this off stops new accounting; existing counters stay visible.</p>
          </div>
          <div id="settings-usage-toggle" class="switch" role="switch" aria-checked="false" tabindex="0"></div>
        </div>
      </div>
      <div class="card">
        <h2 data-i18n="settings.state_title">State</h2>
        <dl class="kv">
          <dt data-i18n="settings.state_path">State file</dt>
          <dd id="settings-state-path">—</dd>
          <dt data-i18n="settings.key_status">Management key</dt>
          <dd id="settings-key-status">—</dd>
        </dl>
        <p id="settings-warning" class="note error-text"></p>
      </div>
    </section>
`,
	styles: "",
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
  var toggle = byId('settings-usage-toggle');
  if (toggle) {
    toggle.setAttribute('aria-checked', usageEnabled() ? 'true' : 'false');
  }

  var path = byId('settings-state-path');
  if (path) {
    path.textContent = state.settings && state.settings.state_path ? state.settings.state_path : '—';
  }

  var keyStatus = byId('settings-key-status');
  if (keyStatus) {
    keyStatus.textContent = state.managementKey ? t('settings.key_loaded') : t('settings.key_missing');
  }

  var warning = byId('settings-warning');
  if (warning) {
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
    warning.textContent = messages.join(' ');
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
`,
}

// helpFeature documents installation, behaviour boundaries and privacy rules.
var helpFeature = pageFeature{
	markup: `
    <section id="panel-help" class="panel">
      <div class="card">
        <h2 data-i18n="help.install_title">Installation</h2>
        <ul class="help-list">
          <li data-i18n="help.install_body"></li>
          <li data-i18n="help.config_body"></li>
          <li data-i18n="help.version_body"></li>
        </ul>
      </div>
      <div class="card">
        <h2 data-i18n="help.keys_title">API keys</h2>
        <ul class="help-list">
          <li data-i18n="help.keys_body"></li>
          <li data-i18n="help.generate_body"></li>
        </ul>
      </div>
      <div class="card">
        <h2 data-i18n="help.usage_title">Token usage</h2>
        <ul class="help-list">
          <li data-i18n="help.usage_body"></li>
          <li data-i18n="help.usage_unattributed_body"></li>
        </ul>
      </div>
      <div class="card">
        <h2 data-i18n="help.privacy_title">Privacy</h2>
        <ul class="help-list">
          <li data-i18n="help.privacy_body"></li>
          <li data-i18n="help.state_body"></li>
        </ul>
      </div>
      <div class="card">
        <h2 data-i18n="help.theme_title">Appearance</h2>
        <ul class="help-list">
          <li data-i18n="help.theme_body"></li>
        </ul>
      </div>
    </section>
`,
	styles:  "",
	scripts: "",
}
