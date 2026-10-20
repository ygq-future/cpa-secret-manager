package management

// keysFeature renders the managed proxy API key list with per-key usage.
var keysFeature = pageFeature{
	markup: `
    <section id="panel-keys" class="panel active">
      <div class="stats-grid">
        <div class="stat-card">
          <span class="stat-label" data-i18n="keys.stat.keys">Keys</span>
          <span class="stat-value" id="stat-keys">0</span>
          <span class="stat-hint" data-i18n="keys.stat.keys_hint">Owned by the host</span>
        </div>
        <div class="stat-card">
          <span class="stat-label" data-i18n="keys.stat.requests">Requests</span>
          <span class="stat-value" id="stat-requests">0</span>
          <span class="stat-hint" data-i18n="keys.stat.requests_hint">Attributed to these keys</span>
        </div>
        <div class="stat-card" id="stat-failed-card">
          <span class="stat-label" data-i18n="keys.stat.failed">Failed</span>
          <span class="stat-value" id="stat-failed">0</span>
          <span class="stat-hint" data-i18n="keys.stat.failed_hint">Upstream or client errors</span>
        </div>
        <div class="stat-card">
          <span class="stat-label" data-i18n="keys.stat.tokens">Total tokens</span>
          <span class="stat-value" id="stat-tokens">0</span>
          <span class="stat-hint" data-i18n="keys.stat.tokens_hint">All models, cumulative</span>
        </div>
      </div>
      <div class="card">
        <div class="toolbar">
          <div class="search">
            {{ICON:search}}
            <input id="keys-filter" name="keys-filter-query" type="search" autocomplete="off" autocapitalize="off" spellcheck="false" readonly data-lpignore="true" data-1p-ignore data-bwignore data-i18n-placeholder="keys.filter" data-i18n-aria-label="keys.filter">
            <button id="keys-clear" class="search-clear" type="button" hidden data-i18n-title="keys.filter_clear" data-i18n-aria-label="keys.filter_clear">{{ICON:close}}</button>
          </div>
          <div class="toolbar-actions">
            <button id="date-range-btn" class="btn btn-secondary date-range-btn" type="button" data-i18n-title="range.calendar_title">
              {{ICON:calendar}}
              <span id="date-range-label"></span>
            </button>
            <button id="keys-unit" class="btn btn-secondary" type="button" data-i18n-title="keys.unit_title">
              {{ICON:unit}}
              <span id="keys-unit-label"></span>
            </button>
            <span id="keys-count" class="count-pill"></span>
            <button id="keys-add" class="btn btn-primary" type="button">
              {{ICON:plus}}
              <span data-i18n="keys.add">Add key</span>
            </button>
          </div>
        </div>
        <div id="keys-table-wrap" class="table-wrap">
          <div id="keys-loading" class="table-loading" aria-hidden="true">
            <div class="spinner"></div>
            <span class="table-loading-text" data-i18n="shell.loading">Loading…</span>
          </div>
          <table class="data-table">
            <thead>
              <tr>
                <th data-i18n="keys.column.key">Key</th>
                <th class="col-remark" data-i18n="keys.column.remark">Remark</th>
                <th class="num" data-i18n="keys.column.last_seen">Last seen</th>
                <th class="num" data-i18n="keys.column.requests">Requests</th>
                <th class="num" data-i18n="keys.column.failed">Failed</th>
                <th class="num" data-i18n="keys.column.input">Input</th>
                <th class="num" data-i18n="keys.column.output">Output</th>
                <th class="num" data-i18n="keys.column.total">Total</th>
                <th class="num" data-i18n="keys.column.models">Models</th>
                <th data-i18n="keys.column.actions">Actions</th>
              </tr>
            </thead>
            <tbody id="keys-body"></tbody>
          </table>
        </div>
      </div>
    </section>
`,
	styles: `
#keys-table-wrap {
  position: relative;
  min-height: 200px;
}

/* The mask is the shared .table-loading component; the keys panel only owns the
   state that reveals it. */
#keys-table-wrap.is-loading .table-loading {
  opacity: 0.88;
  visibility: visible;
  pointer-events: auto;
}

#keys-table-wrap.is-loading .data-table {
  pointer-events: none;
  user-select: none;
  opacity: 0.35;
  transition: opacity 0.2s ease;
}

.key-cell {
  display: flex;
  gap: 8px;
  align-items: center;
}

.empty-row:hover td { background: transparent; }

.data-table tbody tr.key-row.is-expanded td {
  background-color: var(--bg-hover);
  border-bottom: 1px dashed var(--border-color);
}

/* The detail row is a sunken well: the page canvas sits below the card, and the
   nested model table stays on --bg-card so it reads as a sheet inside the well.
   Hover deliberately does not repaint it; the generic row hover excludes this
   row. */
.detail-row > td {
  padding: 0;
  border-top: none;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-inset);
}

.detail-panel {
  padding: 10px 16px 14px 20px;
  animation: detailIn 200ms cubic-bezier(0.16, 1, 0.3, 1);
}

.detail-row.closing .detail-panel {
  animation: detailOut 160ms ease-in forwards;
}

@keyframes detailIn {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: none; }
}

@keyframes detailOut {
  from { opacity: 1; transform: none; }
  to { opacity: 0; transform: translateY(-4px); }
}

@media (prefers-reduced-motion: reduce) {
  .detail-panel, .detail-row.closing .detail-panel { animation: none; }
}

.detail-head {
  display: flex;
  gap: 8px;
  align-items: baseline;
  margin-bottom: 8px;
}

.detail-head h3 {
  font-size: 12px;
  font-weight: 650;
  color: var(--text-secondary);
}

table.detail-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  overflow: hidden;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  font-size: 12px;
}

.detail-table th,
.detail-table td {
  padding: 6px 10px;
  text-align: left;
  vertical-align: middle;
  white-space: nowrap;
}

.detail-table thead th {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 650;
}

.detail-table tbody td { border-top: 1px solid var(--border-color); }
.detail-table tbody tr:first-child td { border-top: 0; }

.detail-table .model-name {
  font-family: var(--font-mono);
  font-size: 11.5px;
}
`,
	scripts: `
function isModelsExpanded(key, models) {
  if (!key) {
    return false;
  }
  if (state.expanded && typeof state.expanded[key] === 'boolean') {
    return state.expanded[key];
  }
  if (Array.isArray(models)) {
    return models.length > 1;
  }
  if (Array.isArray(state.keys) && Array.isArray(state.entries)) {
    var idx = state.keys.indexOf(key);
    if (idx >= 0 && state.entries[idx] && state.entries[idx].usage && Array.isArray(state.entries[idx].usage.models)) {
      return state.entries[idx].usage.models.length > 1;
    }
  }
  return false;
}

function setKeysLoading(loading) {
  var wrap = byId('keys-table-wrap');
  var mask = byId('keys-loading');
  if (!wrap) {
    return;
  }
  if (loading) {
    wrap.classList.add('is-loading');
    if (mask) {
      mask.setAttribute('aria-hidden', 'false');
    }
  } else {
    wrap.classList.remove('is-loading');
    if (mask) {
      mask.setAttribute('aria-hidden', 'true');
    }
  }
}

// forgetHashes drops the plugin-side metadata of hash indexes. Deleting is
// always an explicit call; resolving the key list never destroys anything.
function forgetHashes(hashes) {
  if (!hashes || hashes.length === 0) {
    return Promise.resolve(null);
  }
  return apiFetch(FORGET_PATH, { method: 'POST', body: JSON.stringify({ hashes: hashes }) }).catch(function () {
    // A failed cleanup is retried by the next refresh instead of being lost.
    for (var index = 0; index < hashes.length; index++) {
      if (state.pendingStale.indexOf(hashes[index]) < 0) {
        state.pendingStale.push(hashes[index]);
      }
    }
    return null;
  });
}

// adoptStaleHashes reconciles plugin state with the host key list: entries the
// plugin still stores but the host no longer lists. They are forgotten only once
// they stayed stale for two consecutive reads, so a single bad or partial host
// response can never erase anything.
function adoptStaleHashes(stale) {
  var pending = {};
  for (var index = 0; index < state.pendingStale.length; index++) {
    pending[state.pendingStale[index]] = true;
  }

  var confirmed = [];
  for (var position = 0; position < stale.length; position++) {
    if (pending[stale[position]]) {
      confirmed.push(stale[position]);
    }
  }

  state.pendingStale = stale.slice();
  return forgetHashes(confirmed);
}

function refreshAll(options) {
  var settings = options || {};
  if (state.busy) {
    // A mutation-triggered refresh must not be dropped while a poll is running.
    state.pendingRefresh = settings;
    return Promise.resolve();
  }
  state.busy = true;
  // Loading feedback covers the first paint and every explicit refresh; the
  // background poll stays silent, otherwise the table mask would blink every
  // 15 seconds.
  var showLoading = !settings.silent || state.lastKeysSignature === null;
  if (showLoading) {
    setKeysLoading(true);
    setRefreshLoading(true);
  }
  if (!settings.silent) {
    setStatus(t('shell.loading'), false);
  }
  return loadSettings()
    .then(function () {
      initDateRange();
      return loadKeys();
    })
    .then(function () {
      setStatus(tf('shell.updated_at', { time: new Date().toLocaleTimeString() }), false);
    })
    .catch(function (error) {
      if (state.authBlocked) {
        setStatus(t('error.auth_required'), true);
      } else {
        setStatus(error.message, true);
        if (!settings.silent) {
          showToast(error.message, 'error');
        }
      }
    })
    .then(function () {
      state.busy = false;
      if (showLoading) {
        setKeysLoading(false);
        setRefreshLoading(false);
      }
      var pending = state.pendingRefresh;
      state.pendingRefresh = null;
      if (pending) {
        return refreshAll(pending);
      }
      return null;
    });
}

function loadKeys() {
  return apiFetch(KEYS_PATH).then(function (payload) {
    var list = [];
    if (payload && Array.isArray(payload['api-keys'])) {
      list = payload['api-keys'];
    } else if (payload && Array.isArray(payload.apiKeys)) {
      list = payload.apiKeys;
    }
    var nextKeys = list.map(function (value) {
      return String(value);
    });
    var currentSig = nextKeys.slice().sort().join('|');
    var keysChanged = state.lastKeysSignature !== null && state.lastKeysSignature !== currentSig;
    state.lastKeysSignature = currentSig;
    state.keys = nextKeys;
    return loadEntries(keysChanged);
  }).then(function () {
    renderKeys();
  });
}

// usageWindowPayload is the resolve window. Without an explicit range the
// backend answers with everything retention still allows, which is exactly what
// the first paint should show.
function usageWindowPayload() {
  var payload = { keys: state.keys };
  if (state.dateRange && state.dateRange.start && state.dateRange.end) {
    payload.start = state.dateRange.start.toISOString();
    payload.end = state.dateRange.end.toISOString();
  }
  return payload;
}

function loadEntries(keysChanged) {
  return apiFetch(RESOLVE_PATH, { method: 'POST', body: JSON.stringify(usageWindowPayload()) }).then(function (payload) {
    state.entries = payload && Array.isArray(payload.items) ? payload.items : [];
    var stale = payload && Array.isArray(payload.stale_hashes) ? payload.stale_hashes : [];
    if (keysChanged) {
      return adoptStaleHashes(stale.map(function (value) {
        return String(value);
      }));
    }
    return Promise.resolve(null);
  });
}

function persistKeys(keys) {
  return apiFetch(KEYS_PATH, { method: 'PUT', body: JSON.stringify(keys) });
}

function saveRemark(key, remark) {
  return apiFetch(REMARKS_PATH, { method: 'PUT', body: JSON.stringify({ key: key, remark: remark }) });
}

function setKeyFormSaving(saving) {
  var button = byId('key-form-save');
  if (button) {
    button.disabled = saving;
  }
}

function readFilter() {
  var filter = byId('keys-filter');
  return filter ? filter.value.trim().toLowerCase() : '';
}

// updateFilterClear mirrors the filter state on the search component: the page
// draws its own clear affordance instead of the native Chromium search cross.
function updateFilterClear() {
  var filter = byId('keys-filter');
  var clear = byId('keys-clear');
  if (!clear) {
    return;
  }
  clear.hidden = !filter || filter.value === '';
}

function clearKeyFilter() {
  var filter = byId('keys-filter');
  if (filter) {
    filter.value = '';
    filter.focus();
  }
  renderKeys();
}

// TOKEN_UNITS cycles the display of every token counter: exact numbers, then
// K/M/B. Request and failure counters stay exact - they are counts, not tokens.
var TOKEN_UNITS = ['', 'K', 'M', 'B'];
var DETAIL_COLLAPSE_MS = 160;

function cycleTokenUnit() {
  state.tokenUnit = (state.tokenUnit + 1) % TOKEN_UNITS.length;
  writeStorage(window, 'localStorage', TOKEN_UNIT_STORAGE_KEY, String(state.tokenUnit));
  renderKeys();
}

function tokenUnitLabel() {
  return state.tokenUnit === 0 ? t('keys.unit_raw') : TOKEN_UNITS[state.tokenUnit];
}

function formatTokenValue(value) {
  var number = typeof value === 'number' ? value : Number(value);
  if (!isFinite(number) || state.tokenUnit === 0) {
    return formatNumber(number);
  }
  // The selected unit is a ceiling: a counter below it keeps the next smaller
  // unit, so a selected "M" never renders a real value as "0M".
  var unit = state.tokenUnit;
  while (unit > 0 && Math.abs(number) < Math.pow(1000, unit)) {
    unit--;
  }
  var scaled = number / Math.pow(1000, unit);
  var digits = scaled >= 100 ? 0 : (scaled >= 10 ? 1 : 2);
  return scaled.toFixed(digits).replace(/\.0+$/, '').replace(/(\.\d*?)0+$/, '$1') + TOKEN_UNITS[unit];
}

function renderStats() {
  var requests = 0;
  var failed = 0;
  var tokens = 0;
  for (var index = 0; index < state.entries.length; index++) {
    var usage = state.entries[index] ? state.entries[index].usage : null;
    if (!usage) {
      continue;
    }
    requests += usage.requests || 0;
    failed += usage.failed || 0;
    tokens += usage.total || 0;
  }

  setStatText('stat-keys', state.keys.length);
  setStatText('stat-requests', requests);
  setStatText('stat-failed', failed);
  setStatText('stat-tokens', tokens, formatTokenValue);

  var failedCard = byId('stat-failed-card');
  if (failedCard) {
    var hasFailures = failed > 0;
    failedCard.classList.toggle('is-danger', hasFailures);
    failedCard.classList.toggle('is-actionable', hasFailures);
    failedCard.title = hasFailures ? t('failures.view_details') : '';
    failedCard.onclick = hasFailures ? function () { openFailuresModal(-1); } : null;
  }
}

function setStatText(id, value, formatter) {
  var node = byId(id);
  if (node) {
    node.textContent = (formatter || formatNumber)(value);
  }
}

function renderKeys() {
  initDateRange();
  var body = byId('keys-body');
  if (!body) {
    return;
  }
  body.textContent = '';
  var filter = readFilter();
  var shown = 0;
  for (var index = 0; index < state.keys.length; index++) {
    var value = state.keys[index];
    var item = state.entries[index] || {};
    var remark = typeof item.remark === 'string' ? item.remark : '';
    if (filter && value.toLowerCase().indexOf(filter) < 0 && remark.toLowerCase().indexOf(filter) < 0) {
      continue;
    }
    shown++;
    var models = item.usage && Array.isArray(item.usage.models) ? item.usage.models : [];
    body.appendChild(buildKeyRow(index, value, remark, item.usage));
    if (isModelsExpanded(value, models)) {
      body.appendChild(buildUsageDetailRow(index, item.usage));
    }
  }

  if (shown === 0) {
    body.appendChild(buildEmptyRow(filter ? t('keys.empty_filter') : t('keys.empty')));
  }

  var count = byId('keys-count');
  if (count) {
    count.textContent = tf('keys.count', { shown: formatNumber(shown), total: formatNumber(state.keys.length) });
  }
  var unitLabel = byId('keys-unit-label');
  if (unitLabel) {
    unitLabel.textContent = tokenUnitLabel();
  }
  updateFilterClear();
  renderStats();
}

// quotaBadgeTitle spells out the three parts of a quota badge: the usage of the
// current window, the limit and the reset instant. Both numbers go through
// formatTokenValue, which degrades the unit to the value, so a 1K usage against a
// 1M limit reads "1K", never "0M".
function quotaBadgeTitle(quota) {
  return tf('keys.quota_badge_title', {
    used: formatTokenValue(quota.used_value),
    limit: formatTokenValue(quota.limit_value),
    reset: formatTimestamp(quota.reset_at)
  });
}

// buildQuotaBadge renders the quota tag: a gauge glyph plus "used / limit". Both
// numbers go through formatTokenValue, which degrades the unit to the value, so a
// 1K usage against a 1M limit reads "1K", never "0M".
function buildQuotaBadge(quota) {
  var badge = document.createElement('span');
  badge.className = 'badge-quota';
  badge.title = quotaBadgeTitle(quota);
  badge.appendChild(iconMarkup('quota', ''));
  var text = document.createElement('span');
  text.textContent = formatTokenValue(quota.used_value) + ' / ' + formatTokenValue(quota.limit_value);
  badge.appendChild(text);
  return badge;
}

// buildQuotaResetBadge renders the reset tag: a clock glyph plus the countdown to
// the window reset; the tooltip carries the exact local instant.
function buildQuotaResetBadge(quota) {
  var badge = document.createElement('span');
  badge.className = 'badge-reset';
  badge.title = tf('keys.quota_reset_title', { time: formatTimestamp(quota.reset_at) });
  badge.appendChild(iconMarkup('clock', ''));
  var text = document.createElement('span');
  text.textContent = formatRemaining(quota.reset_at);
  badge.appendChild(text);
  return badge;
}

function buildKeyRow(index, value, remark, usage) {
  var row = document.createElement('tr');
  var item = state.entries[index] || {};
  var models = usage && Array.isArray(usage.models) ? usage.models : [];
  var expanded = isModelsExpanded(value, models);
  row.className = 'key-row' + (expanded ? ' is-expanded' : '');
  var keyCell = document.createElement('td');
  var keyWrap = document.createElement('div');
  keyWrap.className = 'key-cell';
  var chip = document.createElement('code');
  chip.className = 'key-value';
  chip.textContent = maskKey(value);
  keyWrap.appendChild(chip);
  var copy = document.createElement('button');
  copy.className = 'btn-icon';
  copy.type = 'button';
  copy.title = t('keys.copy_title');
  copy.setAttribute('aria-label', t('keys.copy_title'));
  copy.appendChild(iconMarkup('copy', 'icon-copy'));
  copy.appendChild(iconMarkup('check', 'icon-check'));
  copy.addEventListener('click', function () {
    copyKeyValue(index, copy);
  });
  keyWrap.appendChild(copy);
  keyCell.appendChild(keyWrap);
  row.appendChild(keyCell);

  var remarkCell = document.createElement('td');
  remarkCell.className = 'remark';
  // One row of tags, so every tag shares the row's centre line and height.
  var tagRow = document.createElement('div');
  tagRow.className = 'tag-row';
  if (remark) {
    var remarkBadge = document.createElement('span');
    remarkBadge.className = 'badge badge-remark';
    remarkBadge.title = remark;
    var remarkText = document.createElement('span');
    remarkText.textContent = remark;
    remarkBadge.appendChild(remarkText);
    tagRow.appendChild(remarkBadge);
  } else {
    var emptyRemark = document.createElement('span');
    emptyRemark.className = 'muted';
    emptyRemark.textContent = '—';
    tagRow.appendChild(emptyRemark);
  }
  var entryQuota = item.quota;
  if (entryQuota && entryQuota.enabled) {
    if (entryQuota.exceeded) {
      var exBadge = document.createElement('span');
      exBadge.className = 'badge-quota is-exceeded';
      exBadge.title = quotaBadgeTitle(entryQuota);
      var exText = document.createElement('span');
      exText.textContent = t('keys.quota_exceeded');
      exBadge.appendChild(exText);
      tagRow.appendChild(exBadge);
    } else if (entryQuota.limit_value > 0) {
      tagRow.appendChild(buildQuotaBadge(entryQuota));
    }
    if (entryQuota.reset_at) {
      tagRow.appendChild(buildQuotaResetBadge(entryQuota));
    }
    if (Array.isArray(entryQuota.allowed_models) && entryQuota.allowed_models.length > 0) {
      var mBadge = document.createElement('span');
      mBadge.className = 'badge-models';
      mBadge.title = entryQuota.allowed_models.join(', ');
      var mText = document.createElement('span');
      mText.textContent = tf('keys.quota_models_badge', { count: entryQuota.allowed_models.length });
      mBadge.appendChild(mText);
      tagRow.appendChild(mBadge);
    }
  }
  remarkCell.appendChild(tagRow);
  row.appendChild(remarkCell);

  // Last call is lifetime information: it comes from the entry, not from the
  // window-scoped usage block, so a key idle during the selected range still
  // shows when it was last used.
  var lastSeenCell = document.createElement('td');
  lastSeenCell.className = 'num' + (item.last_seen ? '' : ' muted');
  lastSeenCell.textContent = item.last_seen ? formatTimestamp(item.last_seen) : '—';
  row.appendChild(lastSeenCell);

  row.appendChild(buildNumberCell(usage ? usage.requests : 0, !usage));
  row.appendChild(buildFailedCell(usage, index));
  row.appendChild(buildValueCell(usage ? usage.input : 0, !usage, false, formatTokenValue));
  row.appendChild(buildValueCell(usage ? usage.output : 0, !usage, false, formatTokenValue));
  row.appendChild(buildValueCell(usage ? usage.total : 0, !usage, true, formatTokenValue));

  var modelsCell = document.createElement('td');
  modelsCell.className = 'num';
  var models = usage && Array.isArray(usage.models) ? usage.models : [];
  var modelsButton = document.createElement('button');
  modelsButton.className = 'pill-btn';
  modelsButton.type = 'button';
  modelsButton.title = t('keys.models_title');
  modelsButton.setAttribute('aria-label', tf('keys.models_a11y', { count: formatNumber(models.length) }));
  modelsButton.setAttribute('aria-expanded', expanded ? 'true' : 'false');
  var modelsCount = document.createElement('span');
  modelsCount.textContent = formatNumber(models.length);
  modelsButton.appendChild(modelsCount);
  modelsButton.appendChild(iconMarkup('chevron', 'icon-chevron'));
  modelsButton.addEventListener('click', function () {
    toggleModels(index);
  });
  modelsCell.appendChild(modelsButton);
  row.appendChild(modelsCell);

  var actions = document.createElement('td');
  var actionsWrap = document.createElement('div');
  actionsWrap.className = 'row-actions';
  actionsWrap.appendChild(buildIconButton('', 'edit', t('keys.edit'), function () {
    openEditKeyForm(index);
  }));
  actionsWrap.appendChild(buildIconButton('is-danger', 'trash', t('keys.delete'), function () {
    confirmDeleteKey(index);
  }));
  actions.appendChild(actionsWrap);
  row.appendChild(actions);
  return row;
}

function buildNumberCell(value, empty, strong) {
  return buildValueCell(value, empty, strong, formatNumber);
}

function buildValueCell(value, empty, strong, formatter) {
  var cell = document.createElement('td');
  cell.className = 'num' + (strong ? ' num-strong' : '') + (empty ? ' muted' : '');
  cell.textContent = empty ? '—' : formatter(value);
  return cell;
}

function buildFailedCell(usage, index) {
  var cell = document.createElement('td');
  cell.className = 'num';
  var failed = usage ? usage.failed || 0 : 0;
  if (!usage) {
    cell.classList.add('muted');
    cell.textContent = '—';
    return cell;
  }
  if (failed > 0) {
    var badge = document.createElement('span');
    badge.className = 'badge badge-danger badge-clickable';
    badge.textContent = formatNumber(failed);
    badge.title = t('failures.view_details');
    badge.addEventListener('click', function (e) {
      e.stopPropagation();
      openFailuresModal(index);
    });
    cell.appendChild(badge);
    return cell;
  }
  cell.textContent = '0';
  return cell;
}

// toggleModels expands or collapses the model breakdown of one key. Expanding
// renders the row, whose panel plays the detailIn animation; collapsing plays
// detailOut first so the row does not disappear in a single frame.
function toggleModels(index) {
  var value = state.keys[index] || '';
  if (!value) {
    return;
  }
  var item = state.entries[index] || {};
  var models = item.usage && Array.isArray(item.usage.models) ? item.usage.models : [];
  if (state.collapseTimers[index]) {
    window.clearTimeout(state.collapseTimers[index]);
    delete state.collapseTimers[index];
  }
  var currentlyExpanded = isModelsExpanded(value, models);
  if (!currentlyExpanded) {
    state.expanded[value] = true;
    renderKeys();
    return;
  }
  var row = findDetailRow(index);
  if (!row) {
    state.expanded[value] = false;
    renderKeys();
    return;
  }
  row.classList.add('closing');
  if (state.collapseTimers[index]) {
    window.clearTimeout(state.collapseTimers[index]);
  }
  state.collapseTimers[index] = window.setTimeout(function () {
    delete state.collapseTimers[index];
    state.expanded[value] = false;
    renderKeys();
  }, DETAIL_COLLAPSE_MS);
}

function findDetailRow(index) {
  var body = byId('keys-body');
  if (!body) {
    return null;
  }
  var rows = body.querySelectorAll('.detail-row');
  for (var position = 0; position < rows.length; position++) {
    if (rows[position].getAttribute('data-index') === String(index)) {
      return rows[position];
    }
  }
  return null;
}

// keyTableColumnCount keeps full-width rows aligned with the live header, so a
// new column can never leave a dead strip beside a detail or empty row.
function keyTableColumnCount() {
  var body = byId('keys-body');
  var table = body ? body.closest('table') : null;
  // table.tHead is the direct header only; a descendant query would also count
  // the nested model tables inside earlier detail rows.
  if (table && table.tHead && table.tHead.rows.length > 0) {
    return table.tHead.rows[0].cells.length;
  }
  return 10;
}

function buildUsageDetailRow(index, usage) {
  var row = document.createElement('tr');
  row.className = 'detail-row';
  row.setAttribute('data-index', String(index));
  var cell = document.createElement('td');
  cell.colSpan = keyTableColumnCount();

  var panel = document.createElement('div');
  panel.className = 'detail-panel';

  var models = usage && Array.isArray(usage.models) ? usage.models : [];
  var head = document.createElement('div');
  head.className = 'detail-head';
  var title = document.createElement('h3');
  title.textContent = t('keys.detail_title');
  head.appendChild(title);
  if (models.length > 0) {
    var count = document.createElement('span');
    count.className = 'hint';
    count.textContent = tf('keys.detail_count', { count: formatNumber(models.length) });
    head.appendChild(count);
  }
  panel.appendChild(head);

  if (models.length === 0) {
    var empty = document.createElement('p');
    empty.className = 'hint';
    empty.textContent = t('keys.detail_empty');
    panel.appendChild(empty);
  } else {
    panel.appendChild(buildModelTable(models));
  }

  cell.appendChild(panel);
  row.appendChild(cell);
  return row;
}

function buildModelTable(models) {
  var columns = [
    ['model', 'keys.column.model', false],
    ['requests', 'keys.column.requests', false],
    ['failed', 'keys.column.failed', false],
    ['input', 'keys.column.input', true],
    ['output', 'keys.column.output', true],
    ['reasoning', 'keys.column.reasoning', true],
    ['cache_read', 'keys.column.cache_read', true],
    ['cache_creation', 'keys.column.cache_write', true],
    ['total', 'keys.column.total', true]
  ];

  var table = document.createElement('table');
  table.className = 'detail-table';
  var head = document.createElement('thead');
  var headRow = document.createElement('tr');
  for (var columnIndex = 0; columnIndex < columns.length; columnIndex++) {
    var header = document.createElement('th');
    header.textContent = t(columns[columnIndex][1]);
    if (columnIndex > 0) {
      header.className = 'num';
    }
    headRow.appendChild(header);
  }
  head.appendChild(headRow);
  table.appendChild(head);

  var body = document.createElement('tbody');
  for (var rowIndex = 0; rowIndex < models.length; rowIndex++) {
    var model = models[rowIndex] || {};
    var tableRow = document.createElement('tr');
    for (var cellIndex = 0; cellIndex < columns.length; cellIndex++) {
      var key = columns[cellIndex][0];
      var cell = document.createElement('td');
      if (cellIndex === 0) {
        cell.className = 'model-name';
        cell.textContent = model[key] ? String(model[key]) : t('keys.unknown_model');
      } else {
        cell.className = 'num';
        cell.textContent = (columns[cellIndex][2] ? formatTokenValue : formatNumber)(model[key] || 0);
      }
      tableRow.appendChild(cell);
    }
    body.appendChild(tableRow);
  }
  table.appendChild(body);
  return table;
}

function buildEmptyRow(message) {
  var row = document.createElement('tr');
  row.className = 'empty-row';
  var cell = document.createElement('td');
  cell.colSpan = keyTableColumnCount();
  var box = document.createElement('div');
  box.className = 'empty-state';
  box.textContent = message;
  cell.appendChild(box);
  row.appendChild(cell);
  return row;
}

function fallbackCopyText(value) {
  try {
    var area = document.createElement('textarea');
    area.className = 'copy-shuttle';
    area.value = value;
    area.setAttribute('readonly', '');
    document.body.appendChild(area);
    area.focus();
    area.select();
    area.setSelectionRange(0, value.length);
    var ok = document.execCommand('copy');
    document.body.removeChild(area);
    return ok;
  } catch (err) {
    return false;
  }
}

function writeClipboardText(value) {
  if (!value) {
    return Promise.resolve(false);
  }
  var nav = window.navigator;
  if (nav && nav.clipboard && typeof nav.clipboard.writeText === 'function') {
    return nav.clipboard.writeText(value).then(function () {
      return true;
    }).catch(function () {
      return fallbackCopyText(value) ? Promise.resolve(true) : Promise.resolve(false);
    });
  }
  try {
    if (window.parent && window.parent !== window && window.parent.navigator && window.parent.navigator.clipboard) {
      return window.parent.navigator.clipboard.writeText(value).then(function () {
        return true;
      }).catch(function () {
        return fallbackCopyText(value) ? Promise.resolve(true) : Promise.resolve(false);
      });
    }
  } catch (e) {
    // Ignore cross-origin error and fallback below
  }
  return fallbackCopyText(value) ? Promise.resolve(true) : Promise.resolve(false);
}

function copyKeyValue(index, button) {
  var value = state.keys[index] || '';
  if (!value) {
    return;
  }
  writeClipboardText(value).then(function (ok) {
    if (ok) {
      flashCopied(button);
      showToast(t('keys.copied'), 'success');
    } else {
      showToast(t('error.copy_failed'), 'error');
    }
  });
}

function flashCopied(button) {
  if (!button) {
    return;
  }
  button.classList.add('copied');
  window.setTimeout(function () {
    button.classList.remove('copied');
  }, 1400);
}

// setKeyFormMode switches the dialog between "add" (a key is typed or
// generated) and "edit" (the key is immutable and shown masked; only the remark
// can change).
function setKeyFormMode(mode) {
  var valueField = byId('key-form-value-field');
  var chipField = byId('key-form-chip-field');
  var valueInput = byId('key-form-value');
  if (valueField) {
    valueField.hidden = mode === 'edit';
  }
  if (chipField) {
    chipField.hidden = mode !== 'edit';
  }
  if (valueInput) {
    valueInput.required = mode !== 'edit';
    valueInput.disabled = mode === 'edit';
  }
}

function loadAvailableModels() {
  if (state.availableModels && state.availableModels.length > 0) {
    return Promise.resolve(state.availableModels);
  }
  return apiFetch(V1_MODELS_PATH)
    .then(function (payload) {
      var list = [];
      if (payload && Array.isArray(payload.data)) {
        list = payload.data.map(function (item) { return item.id; });
      }
      return list;
    })
    .catch(function () {
      return apiFetch(MODELS_PATH).then(function (payload) {
        return payload && Array.isArray(payload.models) ? payload.models : [];
      }).catch(function () {
        return [];
      });
    })
    .then(function (models) {
      var set = {};
      for (var i = 0; i < models.length; i++) {
        if (models[i]) set[models[i]] = true;
      }
      for (var j = 0; j < state.entries.length; j++) {
        var u = state.entries[j] && state.entries[j].usage;
        if (u && Array.isArray(u.models)) {
          for (var k = 0; k < u.models.length; k++) {
            if (u.models[k] && u.models[k].model) set[u.models[k].model] = true;
          }
        }
      }
      var res = Object.keys(set).sort();
      state.availableModels = res;
      return res;
    });
}

// modelChipGrid is the shared chip grid of the key form. The selection is read
// from the component instead of from the DOM, so no feature re-implements the
// chip markup.
var modelChipGrid = null;
var modelChipToken = 0;

function renderModelsSelection(allowedModels) {
  var container = byId('key-form-models-container');
  if (!container) {
    return;
  }
  var token = ++modelChipToken;
  var allowed = Array.isArray(allowedModels) ? allowedModels : [];
  loadAvailableModels().then(function (models) {
    // A slower earlier load must never overwrite the selection of a newer dialog.
    if (token !== modelChipToken) {
      return;
    }
    modelChipGrid = mountChipGrid({
      container: container,
      items: (models || []).map(function (name) {
        return { value: name, label: name };
      }),
      selected: allowed
    });
  });
}

function readSelectedModels() {
  return modelChipGrid ? modelChipGrid.getSelected() : [];
}

// The quota form uses three shared components: two listboxes and the chip grid.
// They are mounted once, then driven through their own API.
var quotaWindowSelect = null;
var quotaUnitSelect = null;
var quotaDurationSelect = null;

var QUOTA_WINDOW_OPTIONS = [
  { value: 'daily', i18n: 'key_form.quota_window_daily' },
  { value: 'weekly', i18n: 'key_form.quota_window_weekly' },
  { value: 'monthly', i18n: 'key_form.quota_window_monthly' },
  { value: 'duration', i18n: 'key_form.quota_window_duration' }
];

var QUOTA_UNIT_OPTIONS = [
  { value: '1', i18n: 'key_form.quota_unit_tokens' },
  { value: '1000', label: 'K' },
  { value: '1000000', label: 'M' },
  { value: '1000000000', label: 'B' }
];

var QUOTA_DURATION_UNITS = [
  { value: '1', i18n: 'key_form.quota_duration_unit_sec' },
  { value: '60', i18n: 'key_form.quota_duration_unit_min' },
  { value: '3600', i18n: 'key_form.quota_duration_unit_hour' },
  { value: '86400', i18n: 'key_form.quota_duration_unit_day' }
];

function initQuotaForm() {
  quotaWindowSelect = mountSelect({
    container: byId('key-form-quota-window'),
    options: QUOTA_WINDOW_OPTIONS,
    value: 'daily',
    labelledBy: 'key-form-quota-window-label',
    onChange: updateQuotaDurationField
  });
  quotaUnitSelect = mountSelect({
    container: byId('key-form-quota-limit-unit'),
    options: QUOTA_UNIT_OPTIONS,
    value: '1000000',
    titleKey: 'key_form.quota_unit_title',
    onChange: syncQuotaInput
  });
  quotaDurationSelect = mountSelect({
    container: byId('key-form-quota-duration-unit'),
    options: QUOTA_DURATION_UNITS,
    value: '3600',
    titleKey: 'key_form.quota_duration_title',
    onChange: syncQuotaDuration
  });
}

function updateQuotaDurationField() {
  var field = byId('key-form-quota-duration-field');
  if (field && quotaWindowSelect) {
    field.hidden = quotaWindowSelect.getValue() !== 'duration';
  }
}

function setQuotaStatusBanner(quota) {
  var status = byId('key-form-quota-status');
  if (!status) {
    return;
  }
  if (!quota || !quota.enabled) {
    status.hidden = true;
    status.textContent = '';
    return;
  }
  status.hidden = false;
  status.className = 'quota-status-banner' + (quota.exceeded ? ' is-exceeded' : '');
  status.textContent = tf('key_form.quota_current_status', {
    used: formatTokenValue(quota.used_value),
    limit: formatTokenValue(quota.limit_value),
    reset: formatTimestamp(quota.reset_at)
  });
}

function resetQuotaForm() {
  var enabled = byId('key-form-quota-enabled');
  if (enabled) {
    enabled.checked = false;
  }
  var fields = byId('key-form-quota-fields');
  if (fields) {
    fields.hidden = true;
  }
  if (quotaWindowSelect) {
    quotaWindowSelect.setValue('daily');
  }
  updateQuotaDurationField();
  setQuotaDurationValues(3600);
  setQuotaInputValues(0);
  setQuotaStatusBanner(null);
  renderModelsSelection([]);
}

function applyQuotaForm(quota) {
  if (!quota || !quota.enabled) {
    resetQuotaForm();
    return;
  }
  var enabled = byId('key-form-quota-enabled');
  if (enabled) {
    enabled.checked = true;
  }
  var fields = byId('key-form-quota-fields');
  if (fields) {
    fields.hidden = false;
  }
  if (quotaWindowSelect) {
    quotaWindowSelect.setValue(quota.window || 'daily');
  }
  updateQuotaDurationField();
  setQuotaDurationValues(quota.duration_sec || 3600);
  setQuotaInputValues(quota.limit_value || 0);
  setQuotaStatusBanner(quota);
  renderModelsSelection(quota.allowed_models || []);
}

// The duration field mirrors the token limit field: a value plus a unit listbox.
// The wire value stays seconds.
function syncQuotaDuration() {
  var valueInput = byId('key-form-quota-duration');
  var hiddenInput = byId('key-form-quota-duration-sec');
  if (!valueInput || !hiddenInput || !quotaDurationSelect) return;

  var value = parseInt(valueInput.value, 10);
  var multiplier = parseInt(quotaDurationSelect.getValue(), 10) || 1;
  var seconds = isFinite(value) && value > 0 ? Math.round(value * multiplier) : 0;
  hiddenInput.value = String(seconds);
}

function setQuotaDurationValues(seconds) {
  var valueInput = byId('key-form-quota-duration');
  var hiddenInput = byId('key-form-quota-duration-sec');
  if (!valueInput || !hiddenInput) return;

  var total = parseInt(seconds, 10) || 0;
  if (total <= 0) {
    total = 3600;
  }
  hiddenInput.value = String(total);
  var unit = '3600';
  var value = '1';
  if (total % 86400 === 0) {
    unit = '86400';
    value = String(total / 86400);
  } else if (total % 3600 === 0) {
    unit = '3600';
    value = String(total / 3600);
  } else if (total % 60 === 0) {
    unit = '60';
    value = String(total / 60);
  } else {
    unit = '1';
    value = String(total);
  }
  valueInput.value = value;
  if (quotaDurationSelect) {
    quotaDurationSelect.setValue(unit);
  }
  syncQuotaDuration();
}

function syncQuotaInput() {
  var valInput = byId('key-form-quota-limit-val');
  var hiddenInput = byId('key-form-quota-limit');
  var hint = byId('key-form-quota-calc-hint');
  if (!valInput || !hiddenInput || !quotaUnitSelect) return;

  var num = parseFloat(valInput.value);
  if (!isFinite(num) || num <= 0) {
    hiddenInput.value = '0';
    if (hint) {
      hint.textContent = t('key_form.quota_limit_hint');
    }
    return;
  }
  var multiplier = parseInt(quotaUnitSelect.getValue(), 10) || 1;
  var total = Math.round(num * multiplier);
  hiddenInput.value = String(total);
  if (hint) {
    hint.textContent = tf('key_form.quota_limit_calc', {
      total: formatNumber(total),
      scaled: formatTokenValue(total)
    });
  }
}

function setQuotaInputValues(tokens) {
  var valInput = byId('key-form-quota-limit-val');
  var hiddenInput = byId('key-form-quota-limit');
  if (!valInput || !hiddenInput) return;

  var total = parseInt(tokens, 10) || 0;
  hiddenInput.value = String(total);
  var unit = '1000000';
  var value = '';
  if (total > 0) {
    if (total >= 1000000000 && total % 1000000000 === 0) {
      unit = '1000000000';
      value = String(total / 1000000000);
    } else if (total >= 1000000 && total % 1000000 === 0) {
      unit = '1000000';
      value = String(total / 1000000);
    } else if (total >= 1000 && total % 1000 === 0) {
      unit = '1000';
      value = String(total / 1000);
    } else {
      unit = '1';
      value = String(total);
    }
  }
  valInput.value = value;
  if (quotaUnitSelect) {
    quotaUnitSelect.setValue(unit);
  }
  syncQuotaInput();
}

function saveQuotaForm(targetKey) {
  var qEnabled = byId('key-form-quota-enabled');
  var enabled = qEnabled ? qEnabled.checked : false;
  var windowType = quotaWindowSelect ? quotaWindowSelect.getValue() : 'daily';
  var qDuration = byId('key-form-quota-duration-sec');
  var durationSec = qDuration ? parseInt(qDuration.value, 10) || 3600 : 3600;
  var qLimit = byId('key-form-quota-limit');
  var limitValue = qLimit ? parseInt(qLimit.value, 10) || 0 : 0;
  var selectedModels = readSelectedModels();

  if (enabled) {
    var payload = {
      key: targetKey,
      enabled: true,
      strategy: 'token',
      window: windowType,
      duration_sec: durationSec,
      limit_value: limitValue,
      allowed_models: selectedModels
    };
    return apiFetch(QUOTA_PATH, { method: 'PUT', body: JSON.stringify(payload) });
  }

  return apiFetch(QUOTA_PATH + '?key=' + encodeURIComponent(targetKey), { method: 'DELETE' }).catch(function () {
    return null;
  });
}

function openAddKeyForm() {
  state.editingKey = '';
  setKeyFormMode('add');
  var title = byId('key-form-title');
  if (title) {
    title.textContent = t('key_form.add_title');
  }
  var value = byId('key-form-value');
  if (value) {
    value.value = '';
    value.readOnly = true;
  }
  var remark = byId('key-form-remark');
  if (remark) {
    remark.value = '';
  }
  resetQuotaForm();
  var generate = byId('key-form-generate');
  if (generate) {
    generate.disabled = false;
  }
  openModal('key-form-modal');
  generateKeyIntoForm({ silent: true });
}

function openEditKeyForm(index) {
  // Track the key itself: a background refresh must never re-point the dialog at
  // another row.
  state.editingKey = state.keys[index] || '';
  setKeyFormMode('edit');
  var title = byId('key-form-title');
  if (title) {
    title.textContent = t('key_form.edit_title');
  }
  var chip = byId('key-form-key');
  if (chip) {
    chip.textContent = maskKey(state.keys[index] || '');
  }
  var entry = state.entries[index] || {};
  var remark = byId('key-form-remark');
  if (remark) {
    remark.value = typeof entry.remark === 'string' ? entry.remark : '';
  }
  applyQuotaForm(entry.quota);
  openModal('key-form-modal');
  if (remark) {
    remark.focus();
  }
}
function closeKeyForm() {
  state.editingKey = '';
  closeModal('key-form-modal');
}

function generateKeyIntoForm(options) {
  var settings = options && typeof options === 'object' && !options.preventDefault ? options : {};
  var generate = byId('key-form-generate');
  if (generate) {
    generate.disabled = true;
  }
  apiFetch(GENERATE_PATH, { method: 'POST' }).then(function (payload) {
    var value = payload && payload.key ? String(payload.key) : '';
    var input = byId('key-form-value');
    if (input && value) {
      input.value = value;
      input.readOnly = true;
    }
    return copyToClipboard(value);
  }).then(function () {
    if (!settings.silent) {
      showToast(t('keys.generated'), 'success');
    }
  }).catch(function (error) {
    showToast(error.message, 'error');
  }).then(function () {
    if (generate) {
      generate.disabled = false;
    }
  });
}

function copyToClipboard(value) {
  return writeClipboardText(value);
}

function saveKeyForm() {
  var remarkInput = byId('key-form-remark');
  var remark = remarkInput ? remarkInput.value.trim() : '';

  if (state.editingKey) {
    var targetKey = state.editingKey;
    setKeyFormSaving(true);
    closeKeyForm();
    setKeysLoading(true);
    saveRemark(targetKey, remark)
      .then(function () {
        return saveQuotaForm(targetKey);
      })
      .then(function () {
        showToast(t('keys.updated'), 'success');
        return refreshAll({ silent: true });
      })
      .catch(function (error) {
        showToast(error.message, 'error');
      })
      .then(function () {
        setKeyFormSaving(false);
        setKeysLoading(false);
      });
    return;
  }

  var valueInput = byId('key-form-value');
  var value = valueInput ? valueInput.value.trim() : '';
  if (!value) {
    showToast(t('error.key_required'), 'error');
    return;
  }
  var duplicate = state.keys.some(function (existing) {
    return existing === value;
  });
  if (duplicate) {
    showToast(t('error.duplicate_key'), 'error');
    return;
  }

  var next = state.keys.slice();
  next.push(value);

  setKeyFormSaving(true);
  closeKeyForm();
  setKeysLoading(true);
  persistKeys(next)
    .then(function () {
      return saveRemark(value, remark);
    })
    .then(function () {
      return saveQuotaForm(value);
    })
    .then(function () {
      showToast(t('keys.added'), 'success');
      return refreshAll({ silent: true });
    })
    .catch(function (error) {
      showToast(error.message, 'error');
    })
    .then(function () {
      setKeyFormSaving(false);
      setKeysLoading(false);
    });
}

function confirmDeleteKey(index) {
  var value = state.keys[index] || '';
  var entry = state.entries[index] || {};
  openConfirm(tf('keys.delete_confirm', { key: maskKey(value) }), function () {
    var next = state.keys.slice();
    next.splice(index, 1);
    delete state.expanded[value];
    setKeysLoading(true);
    persistKeys(next)
      .then(function () {
        // The host owns the key; the plugin owns its remark and counters.
        if (entry.hash) {
          return apiFetch(FORGET_PATH, { method: 'POST', body: JSON.stringify({ hashes: [entry.hash] }) }).catch(function () {
            // Non-fatal: adoptStaleHashes cleans up unreferenced hashes on subsequent refreshes.
            return null;
          });
        }
        return null;
      })
      .then(function () {
        showToast(t('keys.deleted'), 'success');
        return refreshAll({ silent: true });
      })
      .catch(function (error) {
        showToast(error.message, 'error');
      })
      .then(function () {
        setKeysLoading(false);
      });
  });
}

var calendarBaseMonth = null;
var draftStart = null;
var draftEnd = null;
var draftMode = 'today';

function padZero(n) {
  return n < 10 ? '0' + n : '' + n;
}

function formatDateStr(d) {
  if (!d) return '';
  return d.getFullYear() + '/' + padZero(d.getMonth() + 1) + '/' + padZero(d.getDate());
}

function formatTimeStr(d) {
  if (!d) return '00:00:00';
  return padZero(d.getHours()) + ':' + padZero(d.getMinutes()) + ':' + padZero(d.getSeconds());
}

function parseTimePart(timeStr) {
  var parts = (timeStr || '').split(':');
  var h = parseInt(parts[0], 10) || 0;
  var m = parseInt(parts[1], 10) || 0;
  var s = parseInt(parts[2], 10) || 0;
  return {
    h: Math.min(23, Math.max(0, h)),
    m: Math.min(59, Math.max(0, m)),
    s: Math.min(59, Math.max(0, s))
  };
}

// retentionDays is the configured data retention, which also bounds the range
// the page may query: nothing older than it exists on disk any more.
function retentionDays() {
  var days = state.settings && state.settings.retention_days ? state.settings.retention_days : 30;
  return Math.min(365, Math.max(1, days));
}

// dateRangeBounds is the selectable window: from the first retained day to now.
function dateRangeBounds() {
  var now = new Date();
  var days = retentionDays();
  return {
    min: new Date(now.getFullYear(), now.getMonth(), now.getDate() - (days - 1), 0, 0, 0, 0),
    max: now
  };
}

// clampDateRange keeps a requested window inside the retained data. A range that
// reaches past retention is shortened instead of silently returning nothing.
function clampDateRange(start, end) {
  var bounds = dateRangeBounds();
  var clampedStart = start < bounds.min ? bounds.min : start;
  var clampedEnd = end > bounds.max ? bounds.max : end;
  if (clampedEnd < clampedStart) {
    clampedEnd = clampedStart;
  }
  return { start: clampedStart, end: clampedEnd };
}

// clampDraftToRetention shortens the draft window to the retained days and
// drops a preset selection that no longer describes what is selected. Comparison
// is by day because the picker always rebuilds the end instant with millisecond
// precision.
function clampDraftToRetention() {
  var requestedLabel = formatRangeLabel(draftStart, draftEnd);
  var clamped = clampDateRange(draftStart, draftEnd);
  draftStart = clamped.start;
  draftEnd = clamped.end;
  if (formatRangeLabel(draftStart, draftEnd) !== requestedLabel) {
    draftMode = 'custom';
  }
}

// saveDateRange persists the picked window so a page refresh, a host reload or a
// tab switch does not silently fall back to "today".
function saveDateRange() {
  if (!state.dateRange || !state.dateRange.start || !state.dateRange.end) {
    return;
  }
  writeStorage(window, 'localStorage', RANGE_STORAGE_KEY, JSON.stringify({
    mode: state.dateRange.mode || 'custom',
    start: state.dateRange.start.toISOString(),
    end: state.dateRange.end.toISOString()
  }));
}

// restoreDateRange reads the persisted window and clamps it to the retention
// that is in force now, so shrinking retention invalidates old selections.
function restoreDateRange() {
  var raw = readStorage(window, 'localStorage', RANGE_STORAGE_KEY);
  if (!raw) {
    return null;
  }
  try {
    var saved = JSON.parse(raw);
    var start = new Date(saved.start);
    var end = new Date(saved.end);
    if (isNaN(start.getTime()) || isNaN(end.getTime())) {
      return null;
    }
    var clamped = clampDateRange(start, end);
    return {
      mode: saved.mode || 'custom',
      start: clamped.start,
      end: clamped.end,
      label: formatRangeLabel(clamped.start, clamped.end)
    };
  } catch (error) {
    return null;
  }
}

function initDateRange() {
  if (state.dateRange) {
    var clamped = clampDateRange(state.dateRange.start, state.dateRange.end);
    if (clamped.start.getTime() !== state.dateRange.start.getTime() || clamped.end.getTime() !== state.dateRange.end.getTime()) {
      state.dateRange.start = clamped.start;
      state.dateRange.end = clamped.end;
      state.dateRange.label = formatRangeLabel(clamped.start, clamped.end);
      updateDateRangeLabel();
      saveDateRange();
    }
    return;
  }
  var restored = restoreDateRange();
  if (restored) {
    state.dateRange = restored;
    updateDateRangeLabel();
    saveDateRange();
    return;
  }
  var now = new Date();
  var start = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0, 0);
  var end = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59, 999);
  state.dateRange = {
    mode: 'today',
    start: start,
    end: end,
    label: formatDateStr(start)
  };
  updateDateRangeLabel();
  saveDateRange();
}

// formatRangeLabel renders the picker label: one date for a single day, a range
// otherwise.
function formatRangeLabel(start, end) {
  if (formatDateStr(start) === formatDateStr(end)) {
    return formatDateStr(start);
  }
  return formatDateStr(start) + ' - ' + formatDateStr(end);
}

function updateDateRangeLabel() {
  var labelEl = byId('date-range-label');
  if (labelEl && state.dateRange) {
    labelEl.textContent = state.dateRange.label || formatDateStr(state.dateRange.start);
  }
}

function openDateRangeModal() {
  initDateRange();
  draftStart = new Date(state.dateRange.start.getTime());
  draftEnd = new Date(state.dateRange.end.getTime());
  draftMode = state.dateRange.mode || 'custom';

  calendarBaseMonth = new Date(draftStart.getFullYear(), draftStart.getMonth(), 1);

  var timeStartInput = byId('time-start');
  if (timeStartInput) {
    timeStartInput.value = formatTimeStr(draftStart);
  }
  var timeEndInput = byId('time-end');
  if (timeEndInput) {
    timeEndInput.value = formatTimeStr(draftEnd);
  }

  updatePresetButtons();
  updateDateSelectionStatus();
  renderDateCalendars();
  openModal('date-range-modal');
}

function closeDateRangeModal() {
  closeModal('date-range-modal');
}

function updateDateSelectionStatus() {
  var statusEl = byId('date-selection-status');
  if (!statusEl) return;
  if (!draftStart) {
    statusEl.textContent = '—';
    return;
  }
  var startText = formatDateStr(draftStart);
  if (!draftEnd || formatDateStr(draftStart) === formatDateStr(draftEnd)) {
    statusEl.textContent = startText;
  } else {
    statusEl.textContent = startText + ' - ' + formatDateStr(draftEnd);
  }
}

function updatePresetButtons() {
  var buttons = document.querySelectorAll('.pill[data-preset]');
  for (var i = 0; i < buttons.length; i++) {
    buttons[i].classList.toggle('is-active', buttons[i].getAttribute('data-preset') === draftMode);
  }
}

function applyDateRangePreset(preset) {
  var now = new Date();
  draftMode = preset;
  if (preset === 'last_5_hours') {
    draftStart = new Date(now.getTime() - 5 * 3600 * 1000);
    draftEnd = new Date(now.getTime());
  } else if (preset === 'last_7_days') {
    draftStart = new Date(now.getTime() - 7 * 86400 * 1000);
    draftEnd = new Date(now.getTime());
  } else if (preset === 'last_30_days') {
    draftStart = new Date(now.getTime() - 30 * 86400 * 1000);
    draftEnd = new Date(now.getTime());
  } else if (preset === 'this_month') {
    draftStart = new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0);
    draftEnd = new Date(now.getTime());
  }
  clampDraftToRetention();
  calendarBaseMonth = new Date(draftStart.getFullYear(), draftStart.getMonth(), 1);

  var timeStartInput = byId('time-start');
  if (timeStartInput) timeStartInput.value = formatTimeStr(draftStart);
  var timeEndInput = byId('time-end');
  if (timeEndInput) timeEndInput.value = formatTimeStr(draftEnd);

  updatePresetButtons();
  updateDateSelectionStatus();
  renderDateCalendars();
}

function resetDateRangeToday() {
  var now = new Date();
  draftMode = 'today';
  draftStart = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 0, 0, 0, 0);
  draftEnd = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59, 999);
  calendarBaseMonth = new Date(draftStart.getFullYear(), draftStart.getMonth(), 1);

  var timeStartInput = byId('time-start');
  if (timeStartInput) timeStartInput.value = '00:00:00';
  var timeEndInput = byId('time-end');
  if (timeEndInput) timeEndInput.value = '23:59:59';

  updatePresetButtons();
  updateDateSelectionStatus();
  renderDateCalendars();
}

function confirmDateRange() {
  if (!draftStart) {
    draftStart = new Date();
  }
  if (!draftEnd) {
    draftEnd = new Date(draftStart.getTime());
  }
  var timeStartInput = byId('time-start');
  var timeEndInput = byId('time-end');
  var ts = parseTimePart(timeStartInput ? timeStartInput.value : '');
  var te = parseTimePart(timeEndInput ? timeEndInput.value : '');

  draftStart.setHours(ts.h, ts.m, ts.s, 0);
  draftEnd.setHours(te.h, te.m, te.s, 999);

  if (draftStart > draftEnd) {
    var tmp = draftStart;
    draftStart = draftEnd;
    draftEnd = tmp;
  }

  clampDraftToRetention();

  state.dateRange = {
    mode: draftMode,
    start: draftStart,
    end: draftEnd,
    label: formatRangeLabel(draftStart, draftEnd)
  };

  updateDateRangeLabel();
  saveDateRange();
  closeDateRangeModal();
  // The whole table is a windowed read, so the numbers must be fetched again,
  // and the mask has to be explicit: this path is not a refreshAll() call.
  setKeysLoading(true);
  loadEntries(false)
    .then(function () {
      renderKeys();
    })
    .catch(function (error) {
      showToast(error.message, 'error');
    })
    .then(function () {
      setKeysLoading(false);
    });
}

function renderDateCalendars() {
  if (!calendarBaseMonth) {
    calendarBaseMonth = new Date();
    calendarBaseMonth.setDate(1);
  }
  var leftPanel = byId('calendar-left');
  var rightPanel = byId('calendar-right');
  if (!leftPanel || !rightPanel) return;

  var rightMonth = new Date(calendarBaseMonth.getFullYear(), calendarBaseMonth.getMonth() + 1, 1);

  renderSingleCalendar(leftPanel, calendarBaseMonth, true, false);
  renderSingleCalendar(rightPanel, rightMonth, false, true);
}

function renderSingleCalendar(container, monthDate, showPrev, showNext) {
  container.textContent = '';

  var head = document.createElement('div');
  head.className = 'calendar-head';

  var prevBtn = document.createElement('button');
  prevBtn.type = 'button';
  prevBtn.className = 'btn-icon' + (showPrev ? '' : ' is-invisible');
  prevBtn.title = t('range.previous_month');
  prevBtn.setAttribute('aria-label', t('range.previous_month'));
  prevBtn.appendChild(iconMarkup('prev', ''));
  prevBtn.onclick = function (e) {
    e.stopPropagation();
    calendarBaseMonth = new Date(calendarBaseMonth.getFullYear(), calendarBaseMonth.getMonth() - 1, 1);
    renderDateCalendars();
  };

  var nextBtn = document.createElement('button');
  nextBtn.type = 'button';
  nextBtn.className = 'btn-icon' + (showNext ? '' : ' is-invisible');
  nextBtn.title = t('range.next_month');
  nextBtn.setAttribute('aria-label', t('range.next_month'));
  nextBtn.appendChild(iconMarkup('next', ''));
  nextBtn.onclick = function (e) {
    e.stopPropagation();
    calendarBaseMonth = new Date(calendarBaseMonth.getFullYear(), calendarBaseMonth.getMonth() + 1, 1);
    renderDateCalendars();
  };

  var title = document.createElement('span');
  title.textContent = monthDate.getFullYear() + '年' + (monthDate.getMonth() + 1) + '月';

  head.appendChild(prevBtn);
  head.appendChild(title);
  head.appendChild(nextBtn);
  container.appendChild(head);

  var grid = document.createElement('div');
  grid.className = 'calendar-grid';

  var weekNames = [t('range.week_sun'), t('range.week_mon'), t('range.week_tue'), t('range.week_wed'), t('range.week_thu'), t('range.week_fri'), t('range.week_sat')];
  for (var w = 0; w < 7; w++) {
    var wk = document.createElement('div');
    wk.className = 'calendar-week';
    wk.textContent = weekNames[w];
    grid.appendChild(wk);
  }

  var firstDayOfWeek = new Date(monthDate.getFullYear(), monthDate.getMonth(), 1).getDay();
  var totalDays = new Date(monthDate.getFullYear(), monthDate.getMonth() + 1, 0).getDate();
  var bounds = dateRangeBounds();
  var minDay = new Date(bounds.min.getFullYear(), bounds.min.getMonth(), bounds.min.getDate()).getTime();
  var maxDay = new Date(bounds.max.getFullYear(), bounds.max.getMonth(), bounds.max.getDate()).getTime();

  for (var empty = 0; empty < firstDayOfWeek; empty++) {
    var blank = document.createElement('div');
    blank.className = 'calendar-day empty';
    grid.appendChild(blank);
  }

  for (var day = 1; day <= totalDays; day++) {
    var cell = document.createElement('div');
    cell.className = 'calendar-day';
    cell.textContent = '' + day;
    var thisDate = new Date(monthDate.getFullYear(), monthDate.getMonth(), day);
    var dateMidnight = new Date(thisDate.getFullYear(), thisDate.getMonth(), thisDate.getDate()).getTime();

    var startMidnight = draftStart ? new Date(draftStart.getFullYear(), draftStart.getMonth(), draftStart.getDate()).getTime() : 0;
    var endMidnight = draftEnd ? new Date(draftEnd.getFullYear(), draftEnd.getMonth(), draftEnd.getDate()).getTime() : 0;

    if (startMidnight && dateMidnight === startMidnight) {
      cell.classList.add('range-start');
    }
    if (endMidnight && dateMidnight === endMidnight) {
      cell.classList.add('range-end');
    }
    if (startMidnight && endMidnight && dateMidnight > startMidnight && dateMidnight < endMidnight) {
      cell.classList.add('in-range');
    }

    if (dateMidnight < minDay || dateMidnight > maxDay) {
      // Outside retention the plugin holds no data, so the day is not a choice.
      cell.classList.add('is-disabled');
      cell.setAttribute('aria-disabled', 'true');
    } else {
      cell.onclick = (function (selectedDate) {
        return function () {
          handleDateClick(selectedDate);
        };
      })(thisDate);
    }

    grid.appendChild(cell);
  }

  container.appendChild(grid);
}

function handleDateClick(selectedDate) {
  draftMode = 'custom';
  var selMidnight = new Date(selectedDate.getFullYear(), selectedDate.getMonth(), selectedDate.getDate()).getTime();
  var startMidnight = draftStart ? new Date(draftStart.getFullYear(), draftStart.getMonth(), draftStart.getDate()).getTime() : 0;
  var endMidnight = draftEnd ? new Date(draftEnd.getFullYear(), draftEnd.getMonth(), draftEnd.getDate()).getTime() : 0;

  if (!draftStart || (startMidnight && endMidnight)) {
    draftStart = new Date(selectedDate.getTime());
    draftEnd = null;
  } else if (draftStart && !draftEnd) {
    if (selMidnight < startMidnight) {
      draftEnd = new Date(draftStart.getTime());
      draftStart = new Date(selectedDate.getTime());
    } else {
      draftEnd = new Date(selectedDate.getTime());
    }
  }

  updatePresetButtons();
  updateDateSelectionStatus();
  renderDateCalendars();
}

// The failure viewer is rebuilt on every open. Its two filters are shared
// listboxes, its records are plain components.
function openFailuresModal(index) {
  var container = byId('failures-content');
  if (!container) return;
  container.textContent = '';

  openModal('failures-modal');
  initDateRange();

  var startISO = state.dateRange && state.dateRange.start ? state.dateRange.start.toISOString() : '';
  var endISO = state.dateRange && state.dateRange.end ? state.dateRange.end.toISOString() : '';
  var initialHash = (index >= 0 && state.entries[index]) ? state.entries[index].hash : '';

  var rangeTag = byId('failures-range-tag');
  if (rangeTag) {
    rangeTag.textContent = t('failures.in_range') + ': ' + (state.dateRange ? state.dateRange.label : '');
  }

  var filterBar = document.createElement('div');
  filterBar.className = 'failures-filter-bar';

  var keyOptions = [{ value: '', label: t('failures.all_keys') }];
  for (var k = 0; k < state.keys.length; k++) {
    var kVal = state.keys[k];
    var kEntry = state.entries[k] || {};
    if (kEntry.hash) {
      keyOptions.push({
        value: kEntry.hash,
        label: maskKey(kVal) + (kEntry.remark ? ' (' + kEntry.remark + ')' : '')
      });
    }
  }

  var listWrap = document.createElement('div');
  listWrap.className = 'failures-list-wrap';

  var currentItems = [];

  var keySelect = mountSelect({
    container: document.createElement('div'),
    options: keyOptions,
    value: initialHash,
    titleKey: 'failures.filter_key',
    onChange: function (hash) {
      loadFailures(hash);
    }
  });

  var modelSelect = mountSelect({
    container: document.createElement('div'),
    options: [{ value: '', label: t('failures.all_models') }],
    value: '',
    titleKey: 'failures.filter_model',
    onChange: function () {
      renderItems();
    }
  });

  filterBar.appendChild(keySelect.element);
  filterBar.appendChild(modelSelect.element);
  container.appendChild(filterBar);
  container.appendChild(listWrap);

  function renderItems() {
    listWrap.textContent = '';
    var selectedModel = modelSelect.getValue();
    var filtered = [];
    for (var i = 0; i < currentItems.length; i++) {
      if (!selectedModel || currentItems[i].model === selectedModel) {
        filtered.push(currentItems[i]);
      }
    }

    if (filtered.length === 0) {
      var emptyBox = document.createElement('div');
      emptyBox.className = 'placeholder-note';
      emptyBox.textContent = t('failures.none');
      listWrap.appendChild(emptyBox);
      return;
    }

    for (var j = 0; j < filtered.length; j++) {
      var item = filtered[j];
      var card = document.createElement('div');
      card.className = 'failure-item';

      var meta = document.createElement('div');
      meta.className = 'failure-meta';

      var metaLeft = document.createElement('div');
      metaLeft.className = 'failure-meta-left';

      var timeSpan = document.createElement('span');
      timeSpan.className = 'failure-time';
      timeSpan.textContent = item.time ? new Date(item.time).toLocaleString() : '—';
      metaLeft.appendChild(timeSpan);

      if (item.model) {
        var modelBadge = document.createElement('span');
        modelBadge.className = 'failure-model-badge';
        modelBadge.textContent = item.model;
        metaLeft.appendChild(modelBadge);
      }

      var code = item.status_code || 0;
      var statusBadge = document.createElement('span');
      statusBadge.className = 'failure-status-badge';
      if (code >= 500 || code === 429) {
        statusBadge.classList.add('is-error');
      } else if (code >= 400) {
        statusBadge.classList.add('is-warning');
      }
      statusBadge.textContent = code > 0 ? 'HTTP ' + code : 'Failed';
      metaLeft.appendChild(statusBadge);

      meta.appendChild(metaLeft);

      if (item.message) {
        meta.appendChild(buildIconButton('', 'copy', t('failures.copy_msg'), (function (message) {
          return function () {
            copyToClipboard(message);
            showToast(t('failures.msg_copied'), 'success');
          };
        })(item.message)));
      }

      card.appendChild(meta);

      if (item.message) {
        var bodyEl = document.createElement('div');
        bodyEl.className = 'failure-body';
        bodyEl.textContent = item.message;
        card.appendChild(bodyEl);
      }

      listWrap.appendChild(card);
    }
  }

  function loadFailures(hash) {
    listWrap.textContent = '';
    var loading = document.createElement('div');
    loading.className = 'placeholder-note is-compact';
    loading.textContent = t('shell.loading');
    listWrap.appendChild(loading);

    apiFetch(FAILURES_PATH, {
      method: 'POST',
      body: JSON.stringify({
        hash: hash,
        start: startISO,
        end: endISO
      })
    })
    .then(function (res) {
      currentItems = (res && res.items) || [];

      var modelSet = {};
      for (var i = 0; i < currentItems.length; i++) {
        if (currentItems[i].model) {
          modelSet[currentItems[i].model] = true;
        }
      }
      var newModelOpts = [{ value: '', label: t('failures.all_models') }];
      var models = Object.keys(modelSet).sort();
      for (var m = 0; m < models.length; m++) {
        newModelOpts.push({ value: models[m], label: models[m] });
      }
      modelSelect.setOptions(newModelOpts);
      // A model filter that the new result set no longer offers would silently
      // hide every record; fall back to "all models".
      for (var n = 0; n < newModelOpts.length; n++) {
        if (newModelOpts[n].value === modelSelect.getValue()) {
          return renderItems();
        }
      }
      modelSelect.setValue('');
      renderItems();
    })
    .catch(function (err) {
      listWrap.textContent = '';
      var errBox = document.createElement('div');
      errBox.className = 'placeholder-note is-error';
      errBox.textContent = err.message || 'Failed to load failure logs';
      listWrap.appendChild(errBox);
    });
  }

  loadFailures(keySelect.getValue());
}

function closeFailuresModal() {
  closeModal('failures-modal');
}


// bindKeysEvents wires the keys panel and the dialogs it owns (the key form, the
// date range picker and the failure viewer). The shell only calls it.
function bindKeysEvents() {
  initQuotaForm();

  bindClick('keys-add', openAddKeyForm);
  bindClick('keys-unit', cycleTokenUnit);
  bindClick('keys-clear', clearKeyFilter);

  var filter = byId('keys-filter');
  if (filter) {
    // The field starts read-only so credential managers have nothing to fill on
    // load; the first interaction restores normal typing.
    var unlockFilter = function () {
      if (filter.readOnly) {
        filter.readOnly = false;
      }
    };
    filter.addEventListener('focus', unlockFilter);
    filter.addEventListener('pointerdown', unlockFilter);
    filter.addEventListener('input', function () {
      renderKeys();
    });
  }

  bindClick('key-form-models-all', function () {
    if (modelChipGrid) {
      modelChipGrid.setSelected(state.availableModels || []);
    }
  });
  bindClick('key-form-models-none', function () {
    if (modelChipGrid) {
      modelChipGrid.setSelected([]);
    }
  });

  var quotaEnabled = byId('key-form-quota-enabled');
  if (quotaEnabled) {
    quotaEnabled.addEventListener('change', function () {
      var fields = byId('key-form-quota-fields');
      if (fields) {
        fields.hidden = !quotaEnabled.checked;
      }
    });
  }

  var limitValue = byId('key-form-quota-limit-val');
  if (limitValue) {
    limitValue.addEventListener('input', syncQuotaInput);
  }

  var durationValue = byId('key-form-quota-duration');
  if (durationValue) {
    durationValue.addEventListener('input', syncQuotaDuration);
  }

  var presetPills = document.querySelectorAll('.pill[data-val]');
  for (var index = 0; index < presetPills.length; index++) {
    presetPills[index].addEventListener('click', function () {
      var value = this.getAttribute('data-val');
      var unit = this.getAttribute('data-unit');
      if (limitValue) {
        limitValue.value = value === '0' ? '' : value;
      }
      if (quotaUnitSelect && unit) {
        quotaUnitSelect.setValue(unit);
      }
      syncQuotaInput();
    });
  }

  var presetButtons = document.querySelectorAll('.pill[data-preset]');
  for (var position = 0; position < presetButtons.length; position++) {
    presetButtons[position].addEventListener('click', function () {
      applyDateRangePreset(this.getAttribute('data-preset'));
    });
  }

  bindClick('failures-close', closeFailuresModal);
  bindClick('failures-open-logs', openHostLogs);
  bindBackdrop('failures-modal');

  bindClick('date-range-btn', openDateRangeModal);
  bindClick('date-range-cancel', closeDateRangeModal);
  bindClick('date-range-reset', resetDateRangeToday);
  bindClick('date-range-confirm', confirmDateRange);
  bindBackdrop('date-range-modal');
}

function openHostLogs() {
  closeFailuresModal();
  var hostWin = hostWindow();
  if (hostWin) {
    try {
      var links = hostWin.document.querySelectorAll('a, button, [role="button"]');
      for (var i = 0; i < links.length; i++) {
        var text = (links[i].textContent || '').trim();
        if (text === '日志查看' || text === 'Logs' || text === 'Log viewer') {
          links[i].click();
          return;
        }
      }
      if (hostWin.location) {
        hostWin.location.hash = '#/logs';
        return;
      }
    } catch (e) {
      // Fallback below
    }
  }
}
`,
}
