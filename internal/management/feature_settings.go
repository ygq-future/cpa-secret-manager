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
                <button id="settings-retention-save" class="btn btn-secondary" type="button" disabled aria-busy="false">
                  <span class="spinner" aria-hidden="true"></span>
                  <span data-i18n="settings.save">Save</span>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="settings.state_title">State</h2>
            <div class="state-meta-row">
              <span class="state-meta-label" data-i18n="settings.key_status">Management key</span>
              <span id="settings-key-status" class="state-meta-badge">—</span>
            </div>
            <div class="settings-files-section">
              <div class="settings-files-header">
                <span class="settings-files-title" data-i18n="settings.cache_files_title">Cache files</span>
              </div>
              <div id="settings-cache-files" class="cache-files-list"></div>
            </div>

            <div id="settings-warning" class="alert" hidden>
              {{ICON:warning}}
              <span id="settings-warning-text"></span>
            </div>
          </div>
        </div>
      </div>
    </section>
`,
	styles: `
.state-meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0 8px;
  border-bottom: 1px solid var(--border-subtle);
}
.state-meta-label {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 650;
}
.state-meta-badge {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
}
.state-meta-badge.is-ready {
  color: var(--accent-green-text);
  background: var(--accent-green-soft);
  border-color: var(--accent-green-border);
}
.state-meta-badge.is-missing {
  color: var(--accent-amber-text);
  background: var(--accent-amber-soft);
  border-color: var(--accent-amber-border);
}
.settings-files-section {
  margin-top: 2px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.settings-files-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.settings-files-title {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 650;
}
.cache-files-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cache-file-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  transition: border-color 0.15s ease, background 0.15s ease;
}
.cache-file-item:hover {
  border-color: var(--border-focus);
  background: var(--bg-hover);
}
.cache-file-meta {
  min-width: 0;
  flex: 1 1 auto;
}
.cache-file-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}
.cache-file-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}
.cache-file-path {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cache-file-size {
  flex: 0 0 auto;
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text-primary);
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  padding: 3px 10px;
  border-radius: var(--radius-pill);
}
.cache-file-size.is-missing {
  color: var(--text-tertiary);
  background: transparent;
  border-style: dashed;
  font-weight: 500;
  font-family: inherit;
  font-size: 12px;
}
@media (max-width: 640px) {
  .cache-file-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .cache-file-size {
    align-self: flex-start;
  }
}
`,
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

// savedRetentionDays is the baseline the save action compares against: the
// value the plugin reported, not whatever the field currently shows.
function savedRetentionDays() {
  if (!state.settings) {
    return null;
  }
  var saved = Number(state.settings.retention_days);
  return isFinite(saved) ? saved : null;
}

function enteredRetentionDays() {
  var input = byId('settings-retention-input');
  if (!input || input.value === '') {
    return null;
  }
  var entered = parseInt(input.value, 10);
  return isNaN(entered) ? null : entered;
}

// syncRetentionSaveState keeps the action honest: it is only clickable once the
// field holds a value that differs from the persisted one.
function syncRetentionSaveState() {
  var button = byId('settings-retention-save');
  if (!button || button.classList.contains('is-loading')) {
    return;
  }
  var entered = enteredRetentionDays();
  var saved = savedRetentionDays();
  button.disabled = entered === null || saved === null || entered === saved;
}

// setRetentionSaving swaps the label for the shared spinner while the PUT is in
// flight and keeps the button disabled until the round trip settles.
function setRetentionSaving(saving) {
  var button = byId('settings-retention-save');
  if (!button) {
    return;
  }
  button.classList.toggle('is-loading', saving);
  button.setAttribute('aria-busy', saving ? 'true' : 'false');
  button.disabled = saving;
  if (!saving) {
    syncRetentionSaveState();
  }
}
function formatBytes(bytes) {
  if (typeof bytes !== 'number' || isNaN(bytes) || bytes <= 0) {
    return '0 B';
  }
  var units = ['B', 'KB', 'MB', 'GB'];
  var i = Math.floor(Math.log(bytes) / Math.log(1024));
  if (i >= units.length) {
    i = units.length - 1;
  }
  if (i === 0) {
    return bytes + ' B';
  }
  var val = (bytes / Math.pow(1024, i)).toFixed(i === 1 ? 1 : 2);
  return parseFloat(val) + ' ' + units[i];
}

function renderCacheFiles() {
  var container = byId('settings-cache-files');
  if (!container) return;
  var files = (state.settings && Array.isArray(state.settings.cache_files)) ? state.settings.cache_files : [];
  if (files.length === 0 && state.settings && state.settings.state_path) {
    files = [{
      key: 'state',
      name: 'State Cache',
      path: state.settings.state_path,
      exists: false,
      size_bytes: 0
    }];
  }

  container.innerHTML = '';
  var keyMap = {
    'state': 'settings.file_state',
    'buckets': 'settings.file_buckets',
    'errors': 'settings.file_errors'
  };

  for (var i = 0; i < files.length; i++) {
    var f = files[i];
    var item = document.createElement('div');
    item.className = 'cache-file-item';

    var meta = document.createElement('div');
    meta.className = 'cache-file-meta';

    var titleRow = document.createElement('div');
    titleRow.className = 'cache-file-title-row';

    var label = document.createElement('span');
    label.className = 'cache-file-label';
    var i18nKey = keyMap[f.key];
    label.textContent = (i18nKey ? t(i18nKey) : f.name) || f.key;
    titleRow.appendChild(label);

    var path = document.createElement('div');
    path.className = 'cache-file-path';
    path.textContent = f.path;
    path.title = f.path;

    meta.appendChild(titleRow);
    meta.appendChild(path);

    var sizeBadge = document.createElement('div');
    sizeBadge.className = 'cache-file-size';
    if (f.exists) {
      sizeBadge.textContent = formatBytes(f.size_bytes);
    } else {
      sizeBadge.classList.add('is-missing');
      sizeBadge.textContent = t('settings.file_not_created');
    }

    item.appendChild(meta);
    item.appendChild(sizeBadge);
    container.appendChild(item);
  }
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


  var keyStatus = byId('settings-key-status');
  if (keyStatus) {
    var hasKey = Boolean(state.managementKey);
    keyStatus.textContent = hasKey ? t('settings.key_loaded') : t('settings.key_missing');
    keyStatus.classList.toggle('is-ready', hasKey);
    keyStatus.classList.toggle('is-missing', !hasKey);
  }

  var retInput = byId('settings-retention-input');
  if (retInput && state.settings && state.settings.retention_days) {
    retInput.value = state.settings.retention_days;
  }
  syncRetentionSaveState();
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
  renderCacheFiles();
}

function toggleUsageEnabled() {
  var next = !usageEnabled();
  apiFetch(SETTINGS_PATH, { method: 'PUT', body: JSON.stringify({ usage_enabled: next }) })
    .then(function () {
      return loadSettings();
    })
    .then(function () {
      showToast(t('settings.saved'), 'success');
      return refreshUsage({ silent: true });
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
  if (val === savedRetentionDays()) {
    return;
  }
  setRetentionSaving(true);
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
    })
    .then(function () {
      setRetentionSaving(false);
    });
}

function bindSettingsEvents() {
  bindClick('settings-usage-toggle', toggleUsageEnabled);
  bindClick('settings-retention-save', saveRetentionDays);
  var retInput = byId('settings-retention-input');
  if (retInput) {
    retInput.addEventListener('input', syncRetentionSaveState);
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
