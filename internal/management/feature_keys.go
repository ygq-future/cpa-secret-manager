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
          <label class="search">
            <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="7"></circle><path d="M20 20l-3.6-3.6"></path></svg>
            <input id="keys-filter" type="search" autocomplete="off" data-i18n-placeholder="keys.filter" data-i18n-aria-label="keys.filter">
          </label>
          <div class="toolbar-actions">
            <button id="keys-unit" class="btn btn-secondary" type="button" data-i18n-title="keys.unit_title">
              <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 9h16M4 15h16M10 3 8 21M16 3l-2 18"></path></svg>
              <span id="keys-unit-label"></span>
            </button>
            <span id="keys-count" class="count-pill"></span>
            <button id="keys-add" class="btn btn-primary" type="button">
              <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>
              <span data-i18n="keys.add">Add key</span>
            </button>
          </div>
        </div>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th data-i18n="keys.column.key">Key</th>
                <th data-i18n="keys.column.remark">Remark</th>
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
.key-cell {
  display: flex;
  gap: 8px;
  align-items: center;
}

.empty-row:hover td { background: transparent; }

.data-table tbody tr.detail-row:hover td { background: inherit; }

.detail-row > td {
  padding: 0;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.detail-panel {
  padding: 14px;
  animation: detailIn 200ms cubic-bezier(0.16, 1, 0.3, 1);
}

.detail-row.closing .detail-panel {
  animation: detailOut 160ms ease-in forwards;
}

@keyframes detailIn {
  from { opacity: 0; transform: translateY(-6px); }
  to { opacity: 1; transform: none; }
}

@keyframes detailOut {
  from { opacity: 1; transform: none; }
  to { opacity: 0; transform: translateY(-6px); }
}

@media (prefers-reduced-motion: reduce) {
  .detail-panel, .detail-row.closing .detail-panel { animation: none; }
}

.detail-head {
  display: flex;
  gap: 8px;
  align-items: baseline;
  margin-bottom: 10px;
}

.detail-head h3 {
  font-size: 12.5px;
  font-weight: 700;
}

table.detail-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  overflow: hidden;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-primary);
  font-size: 12.5px;
}

.detail-table th,
.detail-table td {
  padding: 8px 12px;
  text-align: left;
  vertical-align: middle;
  white-space: nowrap;
}

.detail-table thead th {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 11.5px;
  font-weight: 650;
}

.detail-table tbody td { border-top: 1px solid var(--border-color); }
.detail-table tbody tr:first-child td { border-top: 0; }

.detail-table .model-name {
  font-family: var(--font-mono);
  font-size: 12px;
}
`,
	scripts: `
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
  if (!settings.silent) {
    setStatus(t('shell.loading'), false);
  }
  return loadSettings()
    .then(loadKeys)
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
    state.keys = list.map(function (value) {
      return String(value);
    });
    return loadEntries();
  }).then(function () {
    renderKeys();
  });
}

function loadEntries() {
  return apiFetch(RESOLVE_PATH, { method: 'POST', body: JSON.stringify({ keys: state.keys }) }).then(function (payload) {
    state.entries = payload && Array.isArray(payload.items) ? payload.items : [];
    var stale = payload && Array.isArray(payload.stale_hashes) ? payload.stale_hashes : [];
    return adoptStaleHashes(stale.map(function (value) {
      return String(value);
    }));
  });
}

function persistKeys(keys) {
  return apiFetch(KEYS_PATH, { method: 'PUT', body: JSON.stringify(keys) });
}

function saveRemark(key, remark) {
  return apiFetch(REMARKS_PATH, { method: 'PUT', body: JSON.stringify({ key: key, remark: remark }) });
}

function setBusy(busy) {
  state.busy = busy;
  var button = byId('key-form-save');
  if (button) {
    button.disabled = busy;
  }
}

function readFilter() {
  var filter = byId('keys-filter');
  return filter ? filter.value.trim().toLowerCase() : '';
}

// TOKEN_UNITS cycles the display of every token counter: exact numbers, then
// K/M/B. Request and failure counters stay exact - they are counts, not tokens.
var TOKEN_UNITS = ['', 'K', 'M', 'B'];
var DETAIL_COLLAPSE_MS = 160;

function cycleTokenUnit() {
  state.tokenUnit = (state.tokenUnit + 1) % TOKEN_UNITS.length;
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
    failedCard.classList.toggle('is-danger', failed > 0);
  }
}

function setStatText(id, value, formatter) {
  var node = byId(id);
  if (node) {
    node.textContent = (formatter || formatNumber)(value);
  }
}

function renderKeys() {
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
    body.appendChild(buildKeyRow(index, value, remark, item.usage));
    if (state.expanded[index]) {
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
  renderStats();
}

// iconMarkup returns one inline SVG from the page's own glyph set. Callers pass
// only literals, so no page data ever reaches innerHTML.
function iconMarkup(name, className) {
  var glyphs = {
    'copy': '<rect x="9" y="9" width="12" height="12" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>',
    'check': '<polyline points="20 6 9 17 4 12"></polyline>',
    'edit': '<path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path>',
    'trash': '<polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>',
    'chevron': '<polyline points="6 9 12 15 18 9"></polyline>'
  };
  var body = glyphs[name];
  if (!body) {
    return null;
  }
  var wrapper = document.createElement('span');
  var cssClass = className ? 'icon ' + className : 'icon';
  wrapper.innerHTML = '<svg class="' + cssClass + '" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' + body + '</svg>';
  return wrapper.firstChild;
}

function buildIconButton(className, iconName, label, handler) {
  var button = document.createElement('button');
  button.className = 'btn-icon' + (className ? ' ' + className : '');
  button.type = 'button';
  button.title = label;
  button.setAttribute('aria-label', label);
  button.appendChild(iconMarkup(iconName, ''));
  button.addEventListener('click', handler);
  return button;
}

function buildKeyRow(index, value, remark, usage) {
  var row = document.createElement('tr');

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
  remarkCell.className = 'remark' + (remark ? '' : ' muted');
  remarkCell.textContent = remark || '—';
  row.appendChild(remarkCell);

  row.appendChild(buildNumberCell(usage ? usage.requests : 0, !usage));
  row.appendChild(buildFailedCell(usage));
  row.appendChild(buildValueCell(usage ? usage.input : 0, !usage, false, formatTokenValue));
  row.appendChild(buildValueCell(usage ? usage.output : 0, !usage, false, formatTokenValue));
  row.appendChild(buildValueCell(usage ? usage.total : 0, !usage, true, formatTokenValue));

  var modelsCell = document.createElement('td');
  modelsCell.className = 'num';
  var models = usage && Array.isArray(usage.models) ? usage.models : [];
  if (models.length === 0) {
    modelsCell.className = 'num muted';
    modelsCell.textContent = '—';
  } else {
    var modelsButton = document.createElement('button');
    modelsButton.className = 'pill-btn';
    modelsButton.type = 'button';
    modelsButton.title = t('keys.models_title');
    modelsButton.setAttribute('aria-label', tf('keys.models_a11y', { count: formatNumber(models.length) }));
    modelsButton.setAttribute('aria-expanded', state.expanded[index] ? 'true' : 'false');
    var modelsCount = document.createElement('span');
    modelsCount.textContent = formatNumber(models.length);
    modelsButton.appendChild(modelsCount);
    modelsButton.appendChild(iconMarkup('chevron', 'icon-chevron'));
    modelsButton.addEventListener('click', function () {
      toggleModels(index);
    });
    modelsCell.appendChild(modelsButton);
  }
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

function buildFailedCell(usage) {
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
    badge.className = 'badge badge-danger';
    badge.textContent = formatNumber(failed);
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
  if (state.collapseTimers[index]) {
    window.clearTimeout(state.collapseTimers[index]);
    delete state.collapseTimers[index];
  }
  if (!state.expanded[index]) {
    state.expanded[index] = true;
    renderKeys();
    return;
  }
  var row = findDetailRow(index);
  if (!row) {
    state.expanded[index] = false;
    renderKeys();
    return;
  }
  row.classList.add('closing');
  if (state.collapseTimers[index]) {
    window.clearTimeout(state.collapseTimers[index]);
  }
  state.collapseTimers[index] = window.setTimeout(function () {
    delete state.collapseTimers[index];
    state.expanded[index] = false;
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

function buildUsageDetailRow(index, usage) {
  var row = document.createElement('tr');
  row.className = 'detail-row';
  row.setAttribute('data-index', String(index));
  var cell = document.createElement('td');
  cell.colSpan = 9;

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
  cell.colSpan = 9;
  var box = document.createElement('div');
  box.className = 'empty-state';
  box.textContent = message;
  cell.appendChild(box);
  row.appendChild(cell);
  return row;
}

function copyKeyValue(index, button) {
  var value = state.keys[index] || '';
  if (!value) {
    return;
  }
  if (window.navigator.clipboard && window.navigator.clipboard.writeText) {
    window.navigator.clipboard.writeText(value).then(function () {
      flashCopied(button);
      showToast(t('keys.copied'), 'success');
    }, function () {
      showToast(t('error.copy_failed'), 'error');
    });
    return;
  }
  showToast(t('error.copy_failed'), 'error');
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
  if (valueField) {
    valueField.hidden = mode === 'edit';
  }
  if (chipField) {
    chipField.hidden = mode !== 'edit';
  }
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
  }
  var remark = byId('key-form-remark');
  if (remark) {
    remark.value = '';
  }
  var generate = byId('key-form-generate');
  if (generate) {
    generate.disabled = false;
  }
  openModal('key-form-modal');
  if (value) {
    value.focus();
  }
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
  openModal('key-form-modal');
  if (remark) {
    remark.focus();
  }
}

function closeKeyForm() {
  state.editingKey = '';
  closeModal('key-form-modal');
}

function generateKeyIntoForm() {
  var generate = byId('key-form-generate');
  if (generate) {
    generate.disabled = true;
  }
  apiFetch(GENERATE_PATH, { method: 'POST' }).then(function (payload) {
    var value = payload && payload.key ? String(payload.key) : '';
    var input = byId('key-form-value');
    if (input && value) {
      input.value = value;
      input.focus();
      input.select();
    }
    return copyToClipboard(value);
  }).then(function () {
    showToast(t('keys.generated'), 'success');
  }).catch(function (error) {
    showToast(error.message, 'error');
  }).then(function () {
    if (generate) {
      generate.disabled = false;
    }
  });
}

function copyToClipboard(value) {
  if (!value || !window.navigator.clipboard || !window.navigator.clipboard.writeText) {
    return Promise.resolve(false);
  }
  return window.navigator.clipboard.writeText(value).then(function () {
    return true;
  }, function () {
    return false;
  });
}

function saveKeyForm() {
  var remarkInput = byId('key-form-remark');
  var remark = remarkInput ? remarkInput.value.trim() : '';

  if (state.editingKey) {
    // Editing never touches the key list: the host owns the key, the plugin
    // owns the remark.
    setBusy(true);
    saveRemark(state.editingKey, remark)
      .then(function () {
        closeKeyForm();
        showToast(t('keys.updated'), 'success');
        return refreshAll({ silent: true });
      })
      .catch(function (error) {
        showToast(error.message, 'error');
      })
      .then(function () {
        setBusy(false);
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

  setBusy(true);
  persistKeys(next)
    .then(function () {
      return saveRemark(value, remark);
    })
    .then(function () {
      closeKeyForm();
      showToast(t('keys.added'), 'success');
      return refreshAll({ silent: true });
    })
    .catch(function (error) {
      showToast(error.message, 'error');
    })
    .then(function () {
      setBusy(false);
    });
}

function confirmDeleteKey(index) {
  var value = state.keys[index] || '';
  var entry = state.entries[index] || {};
  openConfirm(tf('keys.delete_confirm', { key: maskKey(value) }), function () {
    var next = state.keys.slice();
    next.splice(index, 1);
    state.expanded = {};
    setBusy(true);
    persistKeys(next)
      .then(function () {
        // The host owns the key; the plugin owns its remark and counters. Both
        // disappear with one explicit call.
        return apiFetch(FORGET_PATH, { method: 'POST', body: JSON.stringify({ hashes: [entry.hash] }) });
      })
      .then(function () {
        showToast(t('keys.deleted'), 'success');
        return refreshAll({ silent: true });
      })
      .catch(function (error) {
        showToast(error.message, 'error');
      })
      .then(function () {
        setBusy(false);
      });
  });
}
`,
}
