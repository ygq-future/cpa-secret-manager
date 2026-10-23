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
              <h2 class="card-title" data-i18n="settings.pricing_title">Model Pricing</h2>
              <p class="hint" data-i18n="settings.pricing_hint">Configure price per 1M tokens ($). Cost-limited keys only call models priced here.</p>
              <div class="pricing-add-bar">
                <div id="settings-pricing-model-select" class="pricing-select-wrap"></div>
                <div class="input-unit-group">
                  <span class="unit-prefix">$</span>
                  <input id="settings-pricing-input" class="unit-input" type="number" min="0" step="any" inputmode="decimal" placeholder="0.00" data-i18n-placeholder="settings.model_price_placeholder" title="Input ($/M)">
                  <span class="unit-suffix is-in" data-i18n="settings.tag_input">In</span>
                </div>
                <div class="input-unit-group">
                  <span class="unit-prefix">$</span>
                  <input id="settings-pricing-output" class="unit-input" type="number" min="0" step="any" inputmode="decimal" placeholder="0.00" data-i18n-placeholder="settings.model_price_placeholder" title="Output ($/M)">
                  <span class="unit-suffix is-out" data-i18n="settings.tag_output">Out</span>
                </div>
                <div class="input-unit-group">
                  <span class="unit-prefix">$</span>
                  <input id="settings-pricing-cache" class="unit-input" type="number" min="0" step="any" inputmode="decimal" placeholder="0.00" data-i18n-placeholder="settings.model_price_placeholder" title="Cache ($/M)">
                  <span class="unit-suffix is-cache" data-i18n="settings.tag_cache">Cache</span>
                </div>
                <button id="settings-pricing-add" class="btn btn-secondary pricing-save-btn" type="button" aria-busy="false" data-i18n-title="settings.save_price" title="Save Price">
                  <span class="spinner" aria-hidden="true"></span>
                  {{ICON:save}}
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
.pricing-add-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 10px 0 12px;
}
.pricing-add-bar .pricing-select-wrap {
  flex: 1.35;
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
  flex: 1;
  min-width: 72px;
  height: 36px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  align-items: center;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}
.pricing-add-bar .input-unit-group:hover {
  border-color: var(--text-tertiary);
}
.pricing-add-bar .input-unit-group:focus-within {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--focus-ring);
}
.pricing-add-bar .input-unit-group .unit-prefix {
  background: transparent;
  border: none;
  padding: 0 2px 0 8px;
  color: var(--text-tertiary);
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-weight: 600;
}
.pricing-add-bar .input-unit-group .unit-input {
  background: transparent;
  border: none;
  padding: 0 2px;
  min-width: 0;
  width: 100%;
  height: 100%;
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text-primary);
}
.pricing-add-bar .input-unit-group .unit-suffix {
  border: none;
  border-radius: var(--radius-sm);
  margin-right: 4px;
  padding: 2px 6px;
  height: 20px;
  font-size: 11px;
  font-weight: 600;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  flex-shrink: 0;
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
.pricing-tag-tier {
  display: inline-flex;
  align-items: center;
  height: 18px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 600;
  color: var(--text-secondary);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-pill);
  padding: 0 5px;
  white-space: nowrap;
  line-height: 1;
}
.pricing-tag-tier.is-out {
  color: var(--accent-blue-text);
}
.pricing-tag-tier.is-cache {
  color: var(--accent-green-text);
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
  renderPricingList();
  initPricingModelSelect();
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
      inBadge.className = 'pricing-tag-tier';
      inBadge.textContent = t('settings.tag_input') + ' $' + ((price && price.input) || 0).toFixed(2);
      bottomRow.appendChild(inBadge);

      var outBadge = document.createElement('span');
      outBadge.className = 'pricing-tag-tier is-out';
      outBadge.textContent = t('settings.tag_output') + ' $' + ((price && price.output) || 0).toFixed(2);
      bottomRow.appendChild(outBadge);

      var cacheBadge = document.createElement('span');
      cacheBadge.className = 'pricing-tag-tier is-cache';
      cacheBadge.textContent = t('settings.tag_cache') + ' $' + ((price && price.cache) || 0).toFixed(2);
      bottomRow.appendChild(cacheBadge);

      item.appendChild(topRow);
      item.appendChild(bottomRow);
      container.appendChild(item);
    })(models[i]);
  }
}

function editPricingModel(model, price) {
  if (pricingModelSelect) {
    pricingModelSelect.setValue(model);
  }
  var inInput = byId('settings-pricing-input');
  var outInput = byId('settings-pricing-output');
  var cacheInput = byId('settings-pricing-cache');
  if (inInput) inInput.value = (price && price.input !== undefined) ? Number(price.input).toFixed(2) : '0.00';
  if (outInput) outInput.value = (price && price.output !== undefined) ? Number(price.output).toFixed(2) : '0.00';
  if (cacheInput) cacheInput.value = (price && price.cache !== undefined) ? Number(price.cache).toFixed(2) : '0.00';
  if (inInput) inInput.focus();
}

var pricingModelSelect = null;

function initPricingModelSelect() {
  var container = byId('settings-pricing-model-select');
  if (!container) return;
  if (typeof loadAvailableModels !== 'function') return;
  loadAvailableModels().then(function (models) {
    var opts = (models || []).map(function (m) {
      return { value: m, label: m };
    });
    if (opts.length === 0) {
      opts = [{ value: '', label: t('settings.pricing_empty') }];
    }
    var currentVal = pricingModelSelect ? pricingModelSelect.getValue() : (opts[0] ? opts[0].value : '');
    var exists = opts.some(function (o) { return o.value === currentVal; });
    if (!exists) {
      currentVal = opts[0] ? opts[0].value : '';
    }
    pricingModelSelect = mountSelect({
      container: container,
      options: opts,
      value: currentVal
    });
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
    input: Math.round(inVal * 100) / 100,
    output: Math.round(outVal * 100) / 100,
    cache: Math.round(cacheVal * 100) / 100
  };
  saveModelPrices(current).then(function () {
    inInput.value = '';
    outInput.value = '';
    cacheInput.value = '';
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

function saveModelPrices(prices) {
  setPricingSaving(true);
  setGlobalLoading(true, t('shell.loading'));
  return apiFetch(SETTINGS_PATH, { method: 'PUT', body: JSON.stringify({ model_prices: prices }) })
    .then(function () {
      if (!state.settings) state.settings = {};
      state.settings.model_prices = prices;
      renderPricingList();
      showToast(t('settings.price_saved'), 'success');
      if (typeof loadAvailableModels === 'function') {
        loadAvailableModels();
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
  function bindDecimalInput(id) {
    var el = byId(id);
    if (!el) return;
    el.addEventListener('input', function () {
      var val = el.value;
      if (val.indexOf('.') !== -1) {
        var parts = val.split('.');
        if (parts[1].length > 2) {
          el.value = parts[0] + '.' + parts[1].slice(0, 2);
        }
      }
    });
    el.addEventListener('blur', function () {
      var val = parseFloat(el.value);
      if (!isNaN(val) && val >= 0) {
        el.value = val.toFixed(2);
      }
    });
    el.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') addPricingModel();
    });
  }
  bindDecimalInput('settings-pricing-input');
  bindDecimalInput('settings-pricing-output');
  bindDecimalInput('settings-pricing-cache');
  initPricingModelSelect();
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
