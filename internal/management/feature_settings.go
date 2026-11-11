package management

// settingsFeature renders the plugin business settings and state diagnostics.
// Every control comes from the shared component layer: the usage switch is the
// shared toggle, the retention field is the shared number+unit input.
var settingsFeature = pageFeature{
	markup: `
    <section id="panel-settings" class="panel">
      <div class="scroll-area settings-layout">
        <div class="settings-col-main">
          <div class="card pricing-card">
            <div class="card-body">
              <div class="pricing-head">
                <div class="pricing-head-text">
                  <h2 class="card-title" data-i18n="settings.pricing_title">Model Pricing</h2>
                </div>
                <div class="pricing-head-actions">
                  <span id="settings-pricing-meta" class="pricing-meta"></span>
                  <button id="settings-pricing-refresh" class="btn btn-secondary pricing-sync-btn" type="button" aria-busy="false" data-i18n-title="settings.pricing_refresh_title" title="Refresh reference prices without saving">
                    <span class="spinner" aria-hidden="true"></span>
                    {{ICON:refresh}}
                    <span data-i18n="settings.pricing_refresh">Refresh only</span>
                  </button>
                  <button id="settings-pricing-sync" class="btn btn-secondary pricing-sync-btn" type="button" aria-busy="false" data-i18n-title="settings.pricing_sync_title" title="Sync reference prices">
                    <span class="spinner" aria-hidden="true"></span>
                    {{ICON:refresh}}
                    <span data-i18n="settings.pricing_sync">Sync</span>
                  </button>
                </div>
              </div>
              <div class="pricing-add-bar">
                <div id="settings-pricing-model-select" class="pricing-select-wrap"></div>
                <div class="input-unit-group">
                  <span class="unit-prefix">$</span>
                  <input id="settings-pricing-input" class="unit-input" type="number" min="0" step="any" inputmode="decimal" placeholder="0.000" data-i18n-placeholder="settings.model_price_placeholder" title="Input ($/M)">
                  <span class="unit-suffix is-in" data-i18n="settings.tag_input">In</span>
                </div>
                <div class="input-unit-group">
                  <span class="unit-prefix">$</span>
                  <input id="settings-pricing-output" class="unit-input" type="number" min="0" step="any" inputmode="decimal" placeholder="0.000" data-i18n-placeholder="settings.model_price_placeholder" title="Output ($/M)">
                  <span class="unit-suffix is-out" data-i18n="settings.tag_output">Out</span>
                </div>
                <div class="input-unit-group">
                  <span class="unit-prefix">$</span>
                  <input id="settings-pricing-cache" class="unit-input" type="number" min="0" step="any" inputmode="decimal" placeholder="0.000" data-i18n-placeholder="settings.model_price_placeholder" title="Cache ($/M)">
                  <span class="unit-suffix is-cache" data-i18n="settings.tag_cache">Cache</span>
                </div>
                <button id="settings-pricing-add" class="btn btn-secondary pricing-save-btn" type="button" aria-busy="false" data-i18n-title="settings.save_price" title="Save Price">
                  <span class="spinner" aria-hidden="true"></span>
                  {{ICON:save}}
                </button>
              </div>
              <div id="settings-pricing-suggest" class="pricing-suggest" hidden>
                <span class="pricing-suggest-label" data-i18n="settings.pricing_reference">Reference</span>
                <span id="settings-pricing-suggest-count" class="pricing-suggest-count"></span>
                <div id="settings-pricing-offer-select" class="pricing-offer-select"></div>
                <span id="settings-pricing-suggest-empty" class="pricing-suggest-text" hidden></span>
                <button id="settings-pricing-refresh-model" class="btn-icon pricing-offer-refresh" type="button" aria-busy="false" data-i18n-title="settings.pricing_refresh_model" title="Refresh this model's reference prices">
                  <span class="spinner" aria-hidden="true"></span>
                  {{ICON:refresh}}
                </button>
              </div>
              <div id="settings-pricing-list" class="pricing-list"></div>
            </div>
          </div>
        </div>
        <div class="settings-col-side">
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
.settings-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(0, 1fr);
  gap: 16px;
  align-items: stretch;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
@media (max-width: 980px) {
  .settings-layout {
    grid-template-columns: 1fr;
    height: auto;
    overflow-y: auto;
  }
}
.settings-col-main {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
.settings-col-side {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
}
.pricing-card {
  flex: 1 1 0;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: visible;
}
.pricing-card .card-body {
  flex: 1 1 0;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: visible;
}
.pricing-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.pricing-head-text {
  min-width: 0;
}
.pricing-head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}
.pricing-meta {
  font-size: 11.5px;
  color: var(--text-tertiary);
  white-space: nowrap;
}
.pricing-meta.is-muted {
  color: var(--text-tertiary);
}
.pricing-sync-btn {
  height: 30px;
  padding: 0 10px;
  gap: 6px;
  font-size: 12px;
  white-space: nowrap;
}
.pricing-sync-btn .icon {
  width: 14px;
  height: 14px;
}
.pricing-suggest {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  min-height: 30px;
  margin: -2px 0 10px;
  padding: 6px 10px;
  border: 1px dashed var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-secondary);
  font-size: 12px;
}
.pricing-suggest[hidden] {
  display: none;
}
.pricing-suggest-label {
  flex: 0 0 auto;
  font-weight: 650;
  color: var(--text-primary);
}
.pricing-suggest-count {
  flex: 0 0 auto;
  color: var(--text-tertiary);
  font-size: 11.5px;
  white-space: nowrap;
}
.pricing-offer-select {
  flex: 1 1 auto;
  min-width: 0;
}
.pricing-offer-select[hidden] {
  display: none;
}
.pricing-offer-select .select-trigger {
  height: 28px;
  min-height: 28px;
  padding: 0 8px;
  font-family: var(--font-mono);
  font-size: 12px;
}
.pricing-offer-select .select-label {
  text-align: left;
}
.pricing-offer-select .select-menu {
  width: 100%;
  min-width: 100%;
  max-width: none;
}
.pricing-offer-select .select-option {
  gap: 10px;
}
.pricing-offer-price {
  white-space: nowrap;
}
.pricing-offer-price.is-in {
  color: var(--text-primary);
}
.pricing-offer-price.is-out {
  color: var(--accent-blue-text);
}
.pricing-offer-price.is-cache {
  color: var(--accent-green-text);
}
.pricing-offer-sep {
  color: var(--text-tertiary);
}
.pricing-offer-name {
  color: var(--text-primary);
}
.pricing-offer-name.is-best {
  color: var(--accent-green-text);
  font-weight: 650;
}
.pricing-offer-source {
  color: var(--text-tertiary);
}
.pricing-offer-status.is-applied {
  color: var(--accent-green-text);
  font-weight: 600;
}
.pricing-offer-status.is-pending {
  color: var(--accent-amber-text);
  font-weight: 600;
}
.pricing-offer-refresh {
  width: 28px;
  height: 28px;
  padding: 0;
  flex: 0 0 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--text-tertiary);
}
.pricing-offer-refresh .icon {
  width: 14px;
  height: 14px;
  display: block;
}
.pricing-offer-refresh .spinner {
  display: none;
  width: 14px;
  height: 14px;
  border-width: 2px;
}
.pricing-offer-refresh:hover {
  color: var(--accent-blue-text);
  background: var(--bg-hover);
}
.pricing-offer-refresh.is-loading .icon {
  display: none;
}
.pricing-offer-refresh.is-loading .spinner {
  display: block;
}
.pricing-suggest.is-empty .pricing-suggest-text {
  color: var(--text-tertiary);
}
@media (max-width: 640px) {
  .pricing-head {
    flex-direction: column;
    align-items: stretch;
  }
  .pricing-head-actions {
    justify-content: space-between;
  }
}
.pricing-add-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 10px 0 12px;
}
.pricing-add-bar .pricing-select-wrap {
  flex: 1 1 auto;
  min-width: 120px;
}
.pricing-select-wrap {
  min-width: 0;
  width: 100%;
}
.pricing-select-wrap.select,
.pricing-select-wrap .select {
  width: 100%;
}
.pricing-select-wrap .select-trigger {
  width: 100%;
  height: 36px;
  font-family: var(--font-mono);
  font-size: 13px;
}
.pricing-add-bar .input-unit-group {
  flex: 0 0 auto;
  min-width: 0;
}
.pricing-add-bar .input-unit-group .unit-input {
  width: calc(6ch + 8px);
  padding: 0 4px;
}
@supports (field-sizing: content) {
  .pricing-add-bar .input-unit-group .unit-input {
    field-sizing: content;
    width: auto;
    min-width: calc(3ch + 8px);
    max-width: calc(7ch + 8px);
  }
}
.pricing-add-bar .input-unit-group .unit-suffix.is-out {
  color: var(--accent-blue-text);
  background: var(--accent-blue-subtle);
}
.pricing-add-bar .input-unit-group .unit-suffix.is-cache {
  color: var(--accent-green-text);
  background: var(--accent-green-soft);
}
#settings-pricing-add {
  width: 36px;
  height: 36px;
  padding: 0;
  flex: 0 0 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
}
#settings-pricing-add .icon {
  width: 16px;
  height: 16px;
  display: block;
  flex-shrink: 0;
}
.pricing-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-auto-rows: max-content;
  align-content: start;
  align-items: start;
  gap: 8px;
  flex: 1 1 0;
  min-height: 0;
  max-height: none;
  overflow-y: auto;
  padding: 2px 4px 2px 2px;
}
.pricing-item {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 6px 10px 7px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  transition: border-color 0.15s ease;
  height: auto;
  box-sizing: border-box;
}
@media (max-width: 640px) {
  .pricing-add-bar {
    flex-wrap: wrap;
  }
  .pricing-list {
    grid-template-columns: 1fr;
  }
}
.pricing-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  line-height: 1.2;
}
.pricing-item-model {
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-weight: 650;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pricing-item-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.pricing-item-bottom {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: nowrap;
}
.pricing-item .btn-icon {
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border-radius: var(--radius-sm);
  color: var(--text-tertiary);
  flex-shrink: 0;
}
.pricing-item .btn-icon .icon {
  width: 13px;
  height: 13px;
  display: block;
}
.pricing-item .btn-icon:hover {
  color: var(--accent-blue-text);
  background: var(--bg-hover);
}
.pricing-item .btn-icon.is-danger:hover {
  color: var(--accent-red-text);
  background: var(--accent-red-subtle);
}
.pricing-empty {
  grid-column: 1 / -1;
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
  border: 1px dashed var(--border-subtle);
  border-radius: var(--radius-sm);
}
`,
	scripts: `
var PRICING_PATH = PLUGIN_BASE + '/pricing';
var PRICING_REFRESH_PATH = PLUGIN_BASE + '/pricing/refresh';

function loadSettings() {
  return apiFetch(SETTINGS_PATH).then(function (payload) {
    state.settings = payload || null;
    pricingMeta = (payload && payload.pricing) || null;
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
    'errors': 'settings.file_errors',
    'prices': 'settings.file_prices'
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
  if (state.settings && state.settings.pricing_warning) {
    messages.push(t('settings.pricing_warning') + ': ' + state.settings.pricing_warning);
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
  renderPricingMeta();
  renderPricingList();
  renderPricingSuggestion(pricingModelSelect ? pricingModelSelect.getValue() : '');
  if (state.activeTab === 'settings') {
    initPricingModelSelect();
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
function setPricingSaving(saving) {
  var btn = byId('settings-pricing-add');
  if (btn) {
    btn.disabled = saving;
    btn.setAttribute('aria-busy', saving ? 'true' : 'false');
  }
}

var pricingMeta = null;

function currentPricingMeta() {
  if (pricingMeta) {
    return pricingMeta;
  }
  if (state.settings && state.settings.pricing) {
    return state.settings.pricing;
  }
  return {};
}

function formatPricingValue(value) {
  var number = Number(value);
  return isFinite(number) ? number.toFixed(3) : '0.000';
}

function renderPricingMeta() {
  var meta = byId('settings-pricing-meta');
  if (!meta) return;
  var status = currentPricingMeta();
  if (!status.fetched_at) {
    meta.textContent = t('settings.pricing_never_synced');
    meta.classList.add('is-muted');
    return;
  }
  meta.classList.remove('is-muted');
  meta.textContent = status.stale
    ? tf('settings.pricing_stale', { time: formatTimestamp(status.fetched_at) })
    : tf('settings.pricing_synced_at', { time: formatTimestamp(status.fetched_at) });
  meta.title = t('settings.pricing_sync_title');
}

function pricingSuggestionFor(model) {
  if (!model || !pricingSuggestions || !pricingSuggestions.models) {
    return null;
  }
  return pricingSuggestions.models[model] || null;
}

function pricingValuesEqual(a, b) {
  return Math.abs(Number(a && a.input || 0) - Number(b && b.input || 0)) < 0.0005 &&
    Math.abs(Number(a && a.output || 0) - Number(b && b.output || 0)) < 0.0005 &&
    Math.abs(Number(a && a.cache || 0) - Number(b && b.cache || 0)) < 0.0005;
}

function pricingOffersFor(suggestion) {
  if (!suggestion) {
    return [];
  }
  if (Array.isArray(suggestion.offers) && suggestion.offers.length > 0) {
    return suggestion.offers;
  }
  return [suggestion];
}

function pricingOfferPrices(offer) {
  return '$' + formatPricingValue(offer.input) +
    ' / $' + formatPricingValue(offer.output) +
    ' / $' + formatPricingValue(offer.cache);
}

function pricingOfferPrice(offer) {
  return {
    input: roundPricingValue(offer.input),
    output: roundPricingValue(offer.output),
    cache: roundPricingValue(offer.cache)
  };
}

function pricingOfferDistance(left, right) {
  return Math.abs(Number(left && left.input || 0) - Number(right && right.input || 0)) +
    Math.abs(Number(left && left.output || 0) - Number(right && right.output || 0)) +
    Math.abs(Number(left && left.cache || 0) - Number(right && right.cache || 0));
}

// nearestPricingOffer picks the remaining offer closest to the configured
// price, so a disappeared applied offer degrades to the least surprising quote.
function nearestPricingOffer(offers, configured) {
  var nearest = null;
  var distance = Infinity;
  for (var index = 0; index < offers.length; index++) {
    var candidate = pricingOfferDistance(configured, offers[index]);
    if (candidate < distance) {
      nearest = offers[index];
      distance = candidate;
    }
  }
  return nearest;
}

function pricingPriceParts(offer) {
  return [
    { text: '$' + formatPricingValue(offer.input), className: 'pricing-offer-price is-in' },
    { text: ' / ', className: 'pricing-offer-sep' },
    { text: '$' + formatPricingValue(offer.output), className: 'pricing-offer-price is-out' },
    { text: ' / ', className: 'pricing-offer-sep' },
    { text: '$' + formatPricingValue(offer.cache), className: 'pricing-offer-price is-cache' }
  ];
}

function pricingOfferStatus(offer, configured) {
  if (configured && pricingValuesEqual(configured, offer)) {
    return { text: t('settings.pricing_applied'), className: 'is-applied' };
  }
  if (configured) {
    return { text: t('settings.pricing_pending_update'), className: 'is-pending' };
  }
  return { text: t('settings.pricing_pending_add'), className: 'is-pending' };
}

function buildPricingOfferOptions(offers, configured) {
  return offers.map(function (offer, index) {
    var provider = offer.provider || '—';
    var prices = pricingOfferPrices(offer);
    var status = pricingOfferStatus(offer, configured);
    var nameClass = index === 0 ? 'pricing-offer-name is-best' : 'pricing-offer-name';
    var nameParts = [
      { text: provider, className: index === 0 ? nameClass : 'pricing-offer-source' }
    ];
    if (offer.model) {
      nameParts.push({ text: ' · ' + offer.model, className: nameClass });
    }
    var triggerParts = pricingPriceParts(offer);
    triggerParts.push({ text: ' · ' + provider, className: 'pricing-offer-source' });
    triggerParts.push({ text: ' · ' + status.text, className: 'pricing-offer-status ' + status.className });
    return {
      value: String(index),
      label: provider + (offer.model ? ' · ' + offer.model : '') + ' · ' + prices + ' · ' + status.text,
      triggerLabel: prices + ' · ' + provider + ' · ' + status.text,
      triggerParts: triggerParts,
      labelParts: nameParts,
      trailingParts: pricingPriceParts(offer),
      configured: Boolean(configured && pricingValuesEqual(configured, offer))
    };
  });
}

function applyPricingOffer(offer) {
  var inInput = byId('settings-pricing-input');
  var outInput = byId('settings-pricing-output');
  var cacheInput = byId('settings-pricing-cache');
  if (!inInput || !outInput || !cacheInput) return;
  inInput.value = Number(offer.input || 0).toFixed(3);
  outInput.value = Number(offer.output || 0).toFixed(3);
  cacheInput.value = Number(offer.cache || 0).toFixed(3);
}

function initPricingOfferSelect(options) {
  var container = byId('settings-pricing-offer-select');
  if (!container) return null;
  if (!pricingOfferSelect) {
    pricingOfferSelect = mountSelect({
      container: container,
      options: options,
      value: options[0] ? options[0].value : '',
      onChange: function (selected) {
        var index = parseInt(selected, 10);
        var offer = currentPricingOffers[index];
        if (offer) {
          applyPricingOffer(offer);
        }
      }
    });
  } else {
    pricingOfferSelect.setOptions(options);
  }
  return pricingOfferSelect;
}

function renderPricingSuggestion(model) {
  var box = byId('settings-pricing-suggest');
  var selectWrap = byId('settings-pricing-offer-select');
  var empty = byId('settings-pricing-suggest-empty');
  var count = byId('settings-pricing-suggest-count');
  if (!box || !selectWrap || !empty) return;
  if (!model) {
    box.hidden = true;
    if (pricingOfferSelect) pricingOfferSelect.close();
    currentPricingOffers = [];
    return;
  }

  var suggestion = pricingSuggestionFor(model);
  if (!suggestion) {
    if (!currentPricingMeta().fetched_at) {
      box.hidden = true;
      if (pricingOfferSelect) pricingOfferSelect.close();
      currentPricingOffers = [];
      return;
    }
    box.hidden = false;
    box.className = 'pricing-suggest is-empty';
    if (pricingOfferSelect) pricingOfferSelect.close();
    selectWrap.hidden = true;
    empty.hidden = false;
    empty.textContent = t('settings.pricing_no_suggestion');
    if (count) count.textContent = '';
    currentPricingOffers = [];
    return;
  }

  box.hidden = false;
  box.className = 'pricing-suggest';
  empty.hidden = true;
  selectWrap.hidden = false;

  var offers = pricingOffersFor(suggestion);
  currentPricingOffers = offers;
  var prices = (state.settings && state.settings.model_prices) || {};
  var configured = prices[model];
  var options = buildPricingOfferOptions(offers, configured);
  var select = initPricingOfferSelect(options);
  if (!select) return;

  var selected = options[0] ? options[0].value : '';
  for (var index = 0; index < options.length; index++) {
    if (options[index].configured) {
      selected = options[index].value;
      break;
    }
  }
  select.setValue(selected);

  if (count) {
    count.textContent = suggestion.providers > 1
      ? tf('settings.pricing_provider_count', { count: suggestion.providers })
      : '';
  }
}

function roundPricingValue(value) {
  return Math.round((Number(value) || 0) * 1000) / 1000;
}

// mergePricingSuggestions applies the synced reference prices on top of the
// saved prices. A configured price that still matches any listed offer is kept,
// so an applied offer that vanished upstream falls back to a same-price offer;
// otherwise the nearest remaining offer is applied. When a model has no offer
// left, the saved price stays untouched and the page shows no applied offer. A
// saved model absent from the reference is retained unchanged.
function mergePricingSuggestions(suggestions) {
  var current = (state.settings && state.settings.model_prices) || {};
  var next = {};
  for (var existingModel in current) {
    next[existingModel] = current[existingModel];
  }

  var added = 0;
  var updated = 0;
  var unchanged = 0;
  var models = (suggestions && suggestions.models) || {};
  for (var model in models) {
    var suggestion = models[model];
    var offers = suggestion && Array.isArray(suggestion.offers)
      ? suggestion.offers
      : pricingOffersFor(suggestion);
    var existing = next[model];
    if (!existing) {
      if (offers.length === 0) {
        continue;
      }
      next[model] = pricingOfferPrice(offers[0]);
      added += 1;
      continue;
    }
    var applied = false;
    for (var index = 0; index < offers.length; index++) {
      if (pricingValuesEqual(existing, offers[index])) {
        applied = true;
        break;
      }
    }
    if (applied || offers.length === 0) {
      unchanged += 1;
      continue;
    }
    next[model] = pricingOfferPrice(nearestPricingOffer(offers, existing));
    updated += 1;
  }
  return { prices: next, added: added, updated: updated, unchanged: unchanged };
}

function loadPricingSuggestions(models, force) {
  var list = (models || []).slice();
  var signature = list.join('\n');
  if (!force && pricingSuggestions && pricingSuggestionsSignature === signature) {
    return Promise.resolve(pricingSuggestions);
  }
  if (!force && pricingSuggestionsPending && pricingSuggestionsSignature === signature) {
    return pricingSuggestionsPending;
  }

  pricingSuggestionsSignature = signature;
  var path = force ? PRICING_REFRESH_PATH : PRICING_PATH;
  var pending = apiFetch(path, {
    method: 'POST',
    body: JSON.stringify({ models: list })
  }).then(function (payload) {
    pricingSuggestions = payload || { models: {} };
    pricingMeta = {
      source: payload.source,
      fetched_at: payload.fetched_at,
      stale: payload.stale
    };
    return pricingSuggestions;
  }).catch(function (error) {
    pricingSuggestions = null;
    throw error;
  });
  pricingSuggestionsPending = pending;
  var clearPending = function () {
    if (pricingSuggestionsPending === pending) {
      pricingSuggestionsPending = null;
    }
  };
  pending.then(clearPending, clearPending);
  return pending;
}

function setPricingButtonBusy(id, busy) {
  var button = byId(id);
  if (!button) return;
  button.disabled = busy;
  button.classList.toggle('is-loading', busy);
  button.setAttribute('aria-busy', busy ? 'true' : 'false');
}

function setPricingSyncBusy(busy) {
  setPricingButtonBusy('settings-pricing-sync', busy);
}

function refreshCacheFiles() {
  return apiFetch(SETTINGS_PATH).then(function (payload) {
    if (!payload || !state.settings) return;
    state.settings.cache_files = payload.cache_files || [];
    state.settings.state_warning = payload.state_warning || '';
    state.settings.usage_warning = payload.usage_warning || '';
    state.settings.pricing_warning = payload.pricing_warning || '';
    renderCacheFiles();
  }).catch(function () {
    // The sync/save already succeeded; a stale cache-file badge is harmless.
  });
}

function syncPricingSuggestions() {
  var models = pricingLoadedModels.slice();
  if (models.length === 0) {
    showToast(t('settings.pricing_no_models'), 'error');
    return;
  }
  setPricingSyncBusy(true);
  loadPricingSuggestions(models, true)
    .then(function (result) {
      renderPricingMeta();
      var merge = mergePricingSuggestions(result);
      if (merge.added === 0 && merge.updated === 0) {
        renderPricingSuggestion(pricingModelSelect ? pricingModelSelect.getValue() : '');
        showToast(t('settings.pricing_unchanged'), 'success');
        return refreshCacheFiles();
      }
      return saveModelPrices(merge.prices, tf('settings.pricing_synced_saved', {
        added: merge.added,
        updated: merge.updated,
        unchanged: merge.unchanged
      })).then(function () {
        return refreshCacheFiles();
      });
    })
    .catch(function (error) {
      showToast(error && error.message ? error.message : t('settings.pricing_sync_failed'), 'error');
    })
    .then(function () {
      setPricingSyncBusy(false);
    });
}

// refreshPricingSuggestions downloads the latest catalog for every loaded model
// and updates the cache only. It never writes model_prices.
function refreshPricingSuggestions() {
  var models = pricingLoadedModels.slice();
  if (models.length === 0) {
    showToast(t('settings.pricing_no_models'), 'error');
    return;
  }
  setPricingButtonBusy('settings-pricing-refresh', true);
  loadPricingSuggestions(models, true)
    .then(function () {
      renderPricingMeta();
      renderPricingSuggestion(pricingModelSelect ? pricingModelSelect.getValue() : '');
      showToast(t('settings.pricing_refreshed'), 'success');
      return refreshCacheFiles();
    })
    .catch(function (error) {
      showToast(error && error.message ? error.message : t('settings.pricing_refresh_failed'), 'error');
    })
    .then(function () {
      setPricingButtonBusy('settings-pricing-refresh', false);
    });
}

// refreshCurrentPricing refreshes one model and merges the result into the
// cached suggestion map, so switching to another model keeps its prices.
function refreshCurrentPricing() {
  var model = pricingModelSelect ? pricingModelSelect.getValue() : '';
  if (!model) return;
  setPricingButtonBusy('settings-pricing-refresh-model', true);
  apiFetch(PRICING_REFRESH_PATH, {
    method: 'POST',
    body: JSON.stringify({ models: [model], merge: true })
  }).then(function (payload) {
    if (!pricingSuggestions) {
      pricingSuggestions = { models: {} };
    }
    if (!pricingSuggestions.models) {
      pricingSuggestions.models = {};
    }
    if (payload && payload.models && payload.models[model]) {
      pricingSuggestions.models[model] = payload.models[model];
    } else {
      delete pricingSuggestions.models[model];
    }
    pricingMeta = {
      source: payload.source,
      fetched_at: payload.fetched_at,
      stale: payload.stale
    };
    renderPricingMeta();
    renderPricingSuggestion(model);
    showToast(t('settings.pricing_refreshed'), 'success');
    return refreshCacheFiles();
  }).catch(function (error) {
    showToast(error && error.message ? error.message : t('settings.pricing_refresh_failed'), 'error');
  }).then(function () {
    setPricingButtonBusy('settings-pricing-refresh-model', false);
  });
}

function renderPricingList() {
  var container = byId('settings-pricing-list');
  if (!container) return;
  container.innerHTML = '';
  var prices = (state.settings && state.settings.model_prices) || {};
  var models = Object.keys(prices).sort();
  if (models.length === 0) {
    var empty = document.createElement('div');
    empty.className = 'pricing-empty';
    empty.textContent = t('settings.pricing_empty');
    container.appendChild(empty);
    return;
  }
  for (var i = 0; i < models.length; i++) {
    (function (model) {
      var price = prices[model];
      var item = document.createElement('div');
      item.className = 'pricing-item';

      var topRow = document.createElement('div');
      topRow.className = 'pricing-item-top';
      var modelSpan = document.createElement('span');
      modelSpan.className = 'pricing-item-model';
      modelSpan.textContent = model;
      modelSpan.title = model;
      topRow.appendChild(modelSpan);
      var actions = document.createElement('div');
      actions.className = 'pricing-item-actions';

      var editBtn = buildIconButton('', 'edit', t('settings.edit_price'), function () {
        editPricingModel(model, price);
      });
      actions.appendChild(editBtn);

      var delBtn = buildIconButton('is-danger', 'trash', t('settings.delete_price'), function () {
        deletePricingModel(model);
      });
      actions.appendChild(delBtn);

      topRow.appendChild(actions);
      var bottomRow = document.createElement('div');
      bottomRow.className = 'pricing-item-bottom';
      var inBadge = document.createElement('span');
      inBadge.className = 'price-tier-tag';
      inBadge.textContent = t('settings.tag_input') + ' $' + ((price && price.input) || 0).toFixed(3);
      bottomRow.appendChild(inBadge);

      var outBadge = document.createElement('span');
      outBadge.className = 'price-tier-tag is-out';
      outBadge.textContent = t('settings.tag_output') + ' $' + ((price && price.output) || 0).toFixed(3);
      bottomRow.appendChild(outBadge);

      var cacheBadge = document.createElement('span');
      cacheBadge.className = 'price-tier-tag is-cache';
      cacheBadge.textContent = t('settings.tag_cache') + ' $' + ((price && price.cache) || 0).toFixed(3);
      bottomRow.appendChild(cacheBadge);

      item.appendChild(topRow);
      item.appendChild(bottomRow);
      container.appendChild(item);
    })(models[i]);
  }
}

function syncPricingInputs(model) {
  var inInput = byId('settings-pricing-input');
  var outInput = byId('settings-pricing-output');
  var cacheInput = byId('settings-pricing-cache');
  if (!inInput || !outInput || !cacheInput) return;
  if (!model) {
    inInput.value = '0.000';
    outInput.value = '0.000';
    cacheInput.value = '0.000';
    renderPricingSuggestion('');
    return;
  }
  var prices = (state.settings && state.settings.model_prices) || {};
  var price = prices[model];
  if (price) {
    inInput.value = (price.input !== undefined) ? Number(price.input).toFixed(3) : '0.000';
    outInput.value = (price.output !== undefined) ? Number(price.output).toFixed(3) : '0.000';
    cacheInput.value = (price.cache !== undefined) ? Number(price.cache).toFixed(3) : '0.000';
  } else {
    inInput.value = '0.000';
    outInput.value = '0.000';
    cacheInput.value = '0.000';
  }
  renderPricingSuggestion(model);
}

function editPricingModel(model, price) {
  if (pricingModelSelect) {
    pricingModelSelect.setValue(model);
  }
  syncPricingInputs(model);
  var inInput = byId('settings-pricing-input');
  if (inInput) inInput.focus();
}

var pricingModelSelect = null;
var pricingOfferSelect = null;
var currentPricingOffers = [];
var pricingLoadedModels = [];
var pricingModelsLoaded = false;
var pricingSuggestions = null;
var pricingSuggestionsSignature = '';
var pricingSuggestionsPending = null;

function buildPricingModelOptions(models) {
  var prices = (state.settings && state.settings.model_prices) || {};
  var opts = (models || []).map(function (m) {
    return {
      value: m,
      label: m,
      configured: Boolean(prices[m])
    };
  });
  if (opts.length === 0) {
    opts = [{ value: '', label: t('settings.pricing_empty'), configured: false }];
  }
  return opts;
}

function pricingLoadingOptions() {
  return [{ value: '', label: t('shell.loading'), disabled: true }];
}

function applyPricingModelOptions() {
  var opts = buildPricingModelOptions(pricingLoadedModels);
  var currentVal = pricingModelSelect.getValue();
  pricingModelSelect.setOptions(opts);
  if (!opts.some(function (o) { return o.value === currentVal; })) {
    currentVal = opts[0] ? opts[0].value : '';
    pricingModelSelect.setValue(currentVal);
    syncPricingInputs(currentVal);
  }
}

function refreshPricingModelOptions() {
  if (!pricingModelSelect) return;
  if (!pricingModelsLoaded && pricingLoadedModels.length === 0) return;
  applyPricingModelOptions();
}

function initPricingModelSelect() {
  var container = byId('settings-pricing-model-select');
  if (!container) return;
  if (typeof loadAvailableModels !== 'function') return;
  if (!pricingModelSelect) {
    if (!pricingModelsLoaded && state.availableModels && state.availableModels.length > 0) {
      pricingLoadedModels = state.availableModels.slice();
      pricingModelsLoaded = true;
    }
    var opts = pricingModelsLoaded ? buildPricingModelOptions(pricingLoadedModels) : pricingLoadingOptions();
    pricingModelSelect = mountSelect({
      container: container,
      options: opts,
      value: opts[0] ? opts[0].value : '',
      searchable: true,
      searchPlaceholder: t('key_form.quota_models_search'),
      onChange: function (selectedModel) {
        syncPricingInputs(selectedModel);
      }
    });
    if (pricingModelsLoaded) {
      syncPricingInputs(pricingModelSelect.getValue());
    }
  } else if (!pricingModelsLoaded) {
    pricingModelSelect.setOptions(pricingLoadingOptions());
  }
  loadAvailableModels().then(function (models) {
    pricingLoadedModels = models || [];
    pricingModelsLoaded = true;
    refreshPricingModelOptions();
    return loadPricingSuggestions(pricingLoadedModels).catch(function () {
      return null;
    });
  }).then(function () {
    renderPricingMeta();
    renderPricingSuggestion(pricingModelSelect ? pricingModelSelect.getValue() : '');
  }).catch(function () {
    // Keep the current options; the next init retries the fetch.
  });
}
function addPricingModel() {
  if (!pricingModelSelect) return;
  var model = pricingModelSelect.getValue();
  var inInput = byId('settings-pricing-input');
  var outInput = byId('settings-pricing-output');
  var cacheInput = byId('settings-pricing-cache');
  if (!inInput || !outInput || !cacheInput) return;
  var inVal = parseFloat(inInput.value) || 0;
  var outVal = parseFloat(outInput.value) || 0;
  var cacheVal = parseFloat(cacheInput.value) || 0;
  if (!model) {
    showToast(t('settings.model_required'), 'error');
    return;
  }
  if (isNaN(inVal) || inVal < 0 || isNaN(outVal) || outVal < 0 || isNaN(cacheVal) || cacheVal < 0) {
    showToast(t('settings.price_invalid'), 'error');
    inInput.focus();
    return;
  }
  var current = {};
  if (state.settings && state.settings.model_prices) {
    for (var k in state.settings.model_prices) {
      current[k] = state.settings.model_prices[k];
    }
  }
  current[model] = {
    input: Math.round(inVal * 1000) / 1000,
    output: Math.round(outVal * 1000) / 1000,
    cache: Math.round(cacheVal * 1000) / 1000
  };
  saveModelPrices(current).then(function () {
    var selected = pricingModelSelect ? pricingModelSelect.getValue() : '';
    syncPricingInputs(selected);
  });
}

function deletePricingModel(model) {
  openConfirm(tf('settings.delete_price_confirm', { model: model }), function () {
    var current = {};
      for (var k in state.settings.model_prices) {
        if (k !== model) current[k] = state.settings.model_prices[k];
      }
    saveModelPrices(current);
  });
}

function saveModelPrices(prices, successMessage) {
  setPricingSaving(true);
  setGlobalLoading(true, t('shell.loading'));
  return apiFetch(SETTINGS_PATH, { method: 'PUT', body: JSON.stringify({ model_prices: prices }) })
    .then(function () {
      if (!state.settings) state.settings = {};
      state.settings.model_prices = prices;
      renderPricingList();
      refreshPricingModelOptions();
      showToast(successMessage || t('settings.price_saved'), 'success');
      var selected = pricingModelSelect ? pricingModelSelect.getValue() : '';
      syncPricingInputs(selected);
      if (typeof loadAvailableModels === 'function') {
        loadAvailableModels().then(function (m) {
          pricingLoadedModels = m || [];
          pricingModelsLoaded = true;
          refreshPricingModelOptions();
        });
      }
    })
    .catch(function (error) {
      showToast(error.message, 'error');
    })
    .then(function () {
      setPricingSaving(false);
      setGlobalLoading(false);
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
  bindClick('settings-pricing-add', addPricingModel);
  bindClick('settings-pricing-sync', syncPricingSuggestions);
  bindClick('settings-pricing-refresh', refreshPricingSuggestions);
  bindClick('settings-pricing-refresh-model', refreshCurrentPricing);
  function bindDecimalInput(id) {
    var el = byId(id);
    if (!el) return;
    el.addEventListener('input', function () {
      var val = el.value;
      if (val.indexOf('.') !== -1) {
        var parts = val.split('.');
        if (parts[1].length > 3) {
          el.value = parts[0] + '.' + parts[1].slice(0, 3);
        }
      }
    });
    el.addEventListener('blur', function () {
      var val = parseFloat(el.value);
      if (!isNaN(val) && val >= 0) {
        el.value = val.toFixed(3);
      }
    });
    el.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') addPricingModel();
    });
  }
  bindDecimalInput('settings-pricing-input');
  bindDecimalInput('settings-pricing-output');
  bindDecimalInput('settings-pricing-cache');
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
              <li data-i18n="help.usage_window_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.quota_title">Quotas &amp; Pricing</h2>
            <ul class="help-list">
              <li data-i18n="help.quota_pricing_body"></li>
              <li data-i18n="help.pricing_source_body"></li>
              <li data-i18n="help.quota_limits_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.diagnostics_title">Diagnostics</h2>
            <ul class="help-list">
              <li data-i18n="help.diagnostics_body"></li>
            </ul>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h2 class="card-title" data-i18n="help.privacy_title">Privacy &amp; Storage</h2>
            <ul class="help-list">
              <li data-i18n="help.privacy_body"></li>
              <li data-i18n="help.state_body"></li>
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
