package management

// keysFeature renders the managed proxy API key list with per-key usage.
var keysFeature = pageFeature{
	markup: `
    <section id="panel-keys" class="panel active">
      <div class="card">
        <div class="toolbar">
          <input id="keys-filter" class="filter-input" type="search" autocomplete="off" data-i18n-placeholder="keys.filter">
          <div class="toolbar-actions">
            <span id="keys-count" class="muted"></span>
            <button id="keys-add" class="btn" type="button" data-i18n="keys.add">Add key</button>
          </div>
        </div>
        <div class="table-wrap">
          <table>
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
        <p id="keys-note" class="note"></p>
      </div>
    </section>
`,
	styles: `
.models-toggle {
  padding: 2px 8px;
  border: 1px solid var(--border-primary);
  border-radius: 999px;
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.models-toggle:hover { background: var(--bg-hover); color: var(--text-primary); }

.detail-table { margin-top: 4px; }
`,
	scripts: `
function refreshAll(options) {
  var settings = options || {};
  if (state.busy) {
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
    state.unattributedRequests = payload && payload.unattributed_requests ? payload.unattributed_requests : 0;
    state.orphanRemarks = payload && payload.orphan_remarks ? payload.orphan_remarks : 0;
    state.orphanUsage = payload && payload.orphan_usage ? payload.orphan_usage : 0;
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
      body.appendChild(buildUsageDetailRow(item.usage));
    }
  }

  if (shown === 0) {
    body.appendChild(buildEmptyRow(filter ? t('keys.empty_filter') : t('keys.empty')));
  }

  var count = byId('keys-count');
  if (count) {
    count.textContent = tf('keys.count', { shown: formatNumber(shown), total: formatNumber(state.keys.length) });
  }
  renderKeysNote();
}

function buildKeyRow(index, value, remark, usage) {
  var row = document.createElement('tr');

  var keyCell = document.createElement('td');
  var keyText = document.createElement('span');
  keyText.className = 'key-value';
  keyText.textContent = state.visible[index] ? value : maskKey(value);
  keyCell.appendChild(keyText);

  var toggle = document.createElement('button');
  toggle.className = 'btn-link';
  toggle.type = 'button';
  toggle.textContent = state.visible[index] ? t('keys.hide') : t('keys.show');
  toggle.addEventListener('click', function () {
    state.visible[index] = !state.visible[index];
    renderKeys();
  });
  keyCell.appendChild(toggle);

  var copy = document.createElement('button');
  copy.className = 'btn-link';
  copy.type = 'button';
  copy.textContent = t('keys.copy');
  copy.addEventListener('click', function () {
    copyKeyValue(index);
  });
  keyCell.appendChild(copy);
  row.appendChild(keyCell);

  var remarkCell = document.createElement('td');
  remarkCell.className = 'remark';
  remarkCell.textContent = remark || '—';
  row.appendChild(remarkCell);

  row.appendChild(buildNumberCell(usage ? usage.requests : 0, !usage));
  row.appendChild(buildNumberCell(usage ? usage.failed : 0, !usage));
  row.appendChild(buildNumberCell(usage ? usage.input : 0, !usage));
  row.appendChild(buildNumberCell(usage ? usage.output : 0, !usage));
  row.appendChild(buildNumberCell(usage ? usage.total : 0, !usage));

  var modelsCell = document.createElement('td');
  modelsCell.className = 'num';
  var models = usage && Array.isArray(usage.models) ? usage.models : [];
  if (models.length === 0) {
    modelsCell.textContent = '—';
  } else {
    var modelsButton = document.createElement('button');
    modelsButton.className = 'models-toggle';
    modelsButton.type = 'button';
    modelsButton.textContent = formatNumber(models.length);
    modelsButton.addEventListener('click', function () {
      state.expanded[index] = !state.expanded[index];
      renderKeys();
    });
    modelsCell.appendChild(modelsButton);
  }
  row.appendChild(modelsCell);

  var actions = document.createElement('td');
  var actionsWrap = document.createElement('div');
  actionsWrap.className = 'row-actions';

  var edit = document.createElement('button');
  edit.className = 'btn-link';
  edit.type = 'button';
  edit.textContent = t('keys.edit');
  edit.addEventListener('click', function () {
    openEditKeyForm(index);
  });
  actionsWrap.appendChild(edit);

  var remove = document.createElement('button');
  remove.className = 'btn-link';
  remove.type = 'button';
  remove.textContent = t('keys.delete');
  remove.addEventListener('click', function () {
    confirmDeleteKey(index);
  });
  actionsWrap.appendChild(remove);

  actions.appendChild(actionsWrap);
  row.appendChild(actions);
  return row;
}

function buildNumberCell(value, empty) {
  var cell = document.createElement('td');
  cell.className = 'num';
  cell.textContent = empty ? '—' : formatNumber(value);
  return cell;
}

function buildUsageDetailRow(usage) {
  var row = document.createElement('tr');
  row.className = 'detail-row';
  var cell = document.createElement('td');
  cell.colSpan = 9;

  var body = document.createElement('div');
  body.className = 'detail-body';
  var title = document.createElement('h3');
  title.textContent = t('keys.detail_title');
  body.appendChild(title);

  var models = usage && Array.isArray(usage.models) ? usage.models : [];
  if (models.length === 0) {
    var empty = document.createElement('p');
    empty.className = 'hint';
    empty.textContent = t('keys.detail_empty');
    body.appendChild(empty);
  } else {
    body.appendChild(buildModelTable(models));
  }

  cell.appendChild(body);
  row.appendChild(cell);
  return row;
}

function buildModelTable(models) {
  var columns = [
    ['model', 'keys.column.model'],
    ['requests', 'keys.column.requests'],
    ['failed', 'keys.column.failed'],
    ['input', 'keys.column.input'],
    ['output', 'keys.column.output'],
    ['reasoning', 'keys.column.reasoning'],
    ['cache_read', 'keys.column.cache_read'],
    ['cache_creation', 'keys.column.cache_write'],
    ['total', 'keys.column.total']
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
        cell.className = 'key-value';
        cell.textContent = model[key] ? String(model[key]) : t('keys.unknown_model');
      } else {
        cell.className = 'num';
        cell.textContent = formatNumber(model[key] || 0);
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
  var cell = document.createElement('td');
  cell.colSpan = 9;
  cell.className = 'muted';
  cell.textContent = message;
  row.appendChild(cell);
  return row;
}

function renderKeysNote() {
  var note = byId('keys-note');
  if (!note) {
    return;
  }
  note.textContent = tf('keys.note', {
    unattributed: formatNumber(state.unattributedRequests),
    remarks: formatNumber(state.orphanRemarks),
    usage: formatNumber(state.orphanUsage)
  });
}

function copyKeyValue(index) {
  var value = state.keys[index] || '';
  if (!value) {
    return;
  }
  var done = function () {
    showToast(t('keys.copied'), 'success');
  };
  if (window.navigator.clipboard && window.navigator.clipboard.writeText) {
    window.navigator.clipboard.writeText(value).then(done, function () {
      showToast(t('error.copy_failed'), 'error');
    });
    return;
  }
  showToast(t('error.copy_failed'), 'error');
}

function openAddKeyForm() {
  state.editingIndex = -1;
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
  state.editingIndex = index;
  var title = byId('key-form-title');
  if (title) {
    title.textContent = t('key_form.edit_title');
  }
  var value = byId('key-form-value');
  if (value) {
    value.value = state.keys[index] || '';
  }
  var entry = state.entries[index] || {};
  var remark = byId('key-form-remark');
  if (remark) {
    remark.value = typeof entry.remark === 'string' ? entry.remark : '';
  }
  var generate = byId('key-form-generate');
  if (generate) {
    generate.disabled = true;
  }
  openModal('key-form-modal');
  if (value) {
    value.focus();
  }
}

function closeKeyForm() {
  state.editingIndex = -1;
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
  var valueInput = byId('key-form-value');
  var remarkInput = byId('key-form-remark');
  var value = valueInput ? valueInput.value.trim() : '';
  var remark = remarkInput ? remarkInput.value.trim() : '';
  if (!value) {
    showToast(t('error.key_required'), 'error');
    return;
  }

  var index = state.editingIndex;
  var previous = index >= 0 ? state.keys[index] : '';
  var duplicate = state.keys.some(function (existing, position) {
    return existing === value && position !== index;
  });
  if (duplicate) {
    showToast(t('error.duplicate_key'), 'error');
    return;
  }

  var next = state.keys.slice();
  if (index >= 0) {
    next[index] = value;
  } else {
    next.push(value);
  }

  setBusy(true);
  persistKeys(next)
    .then(function () {
      return saveRemark(value, remark);
    })
    .then(function () {
      if (index >= 0 && previous && previous !== value) {
        return saveRemark(previous, '');
      }
      return null;
    })
    .then(function () {
      closeKeyForm();
      showToast(index >= 0 ? t('keys.updated') : t('keys.added'), 'success');
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
  openConfirm(tf('keys.delete_confirm', { key: maskKey(value) }), function () {
    var next = state.keys.slice();
    next.splice(index, 1);
    state.expanded = {};
    state.visible = {};
    setBusy(true);
    persistKeys(next)
      .then(function () {
        return saveRemark(value, '');
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
