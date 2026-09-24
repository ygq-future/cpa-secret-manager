package management

import "strings"

// templateScriptShellCoreTemplate provides the page runtime shared by all tabs:
// CPA theme following, management key resolution, the authenticated fetch
// wrapper, internationalization, tab switching and shared overlays.
const templateScriptShellCoreTemplate = `
var PLUGIN_ID = '{{PLUGIN_ID}}';
// HOST_THEME_VARIABLES are the host tokens the page consumes: surfaces, text
// and borders. Accents are the page's own tokens (see ThemePaletteCSS), so a
// host theme can never repaint the page's buttons or states.
var HOST_THEME_VARIABLES = [
  '--bg-primary', '--bg-secondary', '--bg-tertiary', '--bg-surface',
  '--bg-hover', '--text-primary', '--text-secondary', '--text-tertiary',
  '--text-muted', '--border-color', '--border-subtle'
];

var REFRESH_INTERVAL_MS = 15000;
var LANGUAGE_STORAGE_KEY = 'cpa-secret-manager-language';
var TOKEN_UNIT_STORAGE_KEY = 'cpa-secret-manager-token-unit';
var MANUAL_KEY_STORAGE = 'cpa-secret-manager-key';
var HOST_AUTH_STORAGE_KEY = 'cli-proxy-auth';
var HOST_THEME_STORAGE_KEY = 'cli-proxy-theme';
var HOST_LANGUAGE_STORAGE_KEY = 'cli-proxy-language';
var ENC_PREFIX = 'enc::v1::';
var STORAGE_SALT = 'cli-proxy-api-webui::secure-storage';

var API_BASE = '/v0/management';
var KEYS_PATH = API_BASE + '/api-keys';
var PLUGIN_BASE = API_BASE + '/plugins/' + PLUGIN_ID;
var SETTINGS_PATH = PLUGIN_BASE + '/settings';
var RESOLVE_PATH = PLUGIN_BASE + '/resolve';
var FORGET_PATH = PLUGIN_BASE + '/forget';
var REMARKS_PATH = PLUGIN_BASE + '/remarks';
var GENERATE_PATH = PLUGIN_BASE + '/keys/generate';

var state = {
  managementKey: '',
  keys: [],
  entries: [],
  expanded: {},
  settings: null,
  pendingStale: [],
  collapseTimers: {},
  editingKey: '',
  activeTab: 'keys',
  authBlocked: false,
  tokenUnit: 0,
  busy: false,
  pendingRefresh: null,
  refreshTimer: 0,
  language: 'zh-CN',
  toastTimer: 0,
  confirmHandler: null,
  themeObserver: null
};

function byId(id) {
  return document.getElementById(id);
}

function hostWindow() {
  try {
    if (window.parent && window.parent !== window) {
      return window.parent;
    }
  } catch (error) {
    return null;
  }
  return null;
}

function hostDocument() {
  var parent = hostWindow();
  if (!parent) {
    return null;
  }
  try {
    return parent.document;
  } catch (error) {
    return null;
  }
}

function readStorage(source, kind, key) {
  if (!source) {
    return '';
  }
  try {
    var storage = source[kind];
    var value = storage ? storage.getItem(key) : null;
    return typeof value === 'string' ? value : '';
  } catch (error) {
    return '';
  }
}

function writeStorage(source, kind, key, value) {
  if (!source) {
    return false;
  }
  try {
    var storage = source[kind];
    if (!storage) {
      return false;
    }
    if (value === '') {
      storage.removeItem(key);
    } else {
      storage.setItem(key, value);
    }
    return true;
  } catch (error) {
    return false;
  }
}
function readStoredTokenUnit() {
  var stored = readStorage(window, 'localStorage', TOKEN_UNIT_STORAGE_KEY);
  if (!stored) {
    return 0;
  }
  var parsed = parseInt(stored, 10);
  if (isFinite(parsed) && parsed >= 0 && parsed <= 3) {
    return parsed;
  }
  return 0;
}


function normalizeKey(value) {
  if (typeof value !== 'string') {
    return '';
  }
  var trimmed = value.trim();
  if (trimmed.slice(0, 7).toLowerCase() === 'bearer ') {
    trimmed = trimmed.slice(7).trim();
  }
  return trimmed;
}

function decodeOfficialValue(raw) {
  if (raw.slice(0, ENC_PREFIX.length) !== ENC_PREFIX) {
    return raw;
  }
  try {
    var binary = window.atob(raw.slice(ENC_PREFIX.length));
    var bytes = new Uint8Array(binary.length);
    for (var index = 0; index < binary.length; index++) {
      bytes[index] = binary.charCodeAt(index);
    }
    var salt = STORAGE_SALT + '|' + window.location.host + '|' + window.navigator.userAgent;
    var saltBytes = new window.TextEncoder().encode(salt);
    for (var byteIndex = 0; byteIndex < bytes.length; byteIndex++) {
      bytes[byteIndex] = bytes[byteIndex] ^ saltBytes[byteIndex % saltBytes.length];
    }
    return new window.TextDecoder().decode(bytes);
  } catch (error) {
    return '';
  }
}

function extractKeyFromStored(raw) {
  if (!raw) {
    return '';
  }
  var decoded = decodeOfficialValue(raw);
  if (!decoded) {
    return '';
  }
  var trimmed = decoded.trim();
  var first = trimmed.charAt(0);
  if (first !== '{' && first !== '[' && first !== '"') {
    return normalizeKey(trimmed);
  }
  try {
    var parsed = JSON.parse(trimmed);
    if (typeof parsed === 'string') {
      return normalizeKey(parsed);
    }
    if (parsed && typeof parsed === 'object') {
      var payload = parsed.state && typeof parsed.state === 'object' ? parsed.state : parsed;
      if (typeof payload.managementKey === 'string') {
        return normalizeKey(payload.managementKey);
      }
      if (typeof payload.key === 'string') {
        return normalizeKey(payload.key);
      }
    }
  } catch (error) {
    return '';
  }
  return '';
}

function readOfficialManagementKey() {
  var candidates = [hostWindow(), window];
  for (var index = 0; index < candidates.length; index++) {
    var value = extractKeyFromStored(readStorage(candidates[index], 'localStorage', HOST_AUTH_STORAGE_KEY));
    if (value) {
      return value;
    }
  }
  return '';
}

function readPlainManagementKey() {
  var candidates = [hostWindow(), window];
  var names = ['managementKey', 'management_key', 'management-key', 'cpa-management-key'];
  for (var index = 0; index < candidates.length; index++) {
    for (var nameIndex = 0; nameIndex < names.length; nameIndex++) {
      var value = extractKeyFromStored(readStorage(candidates[index], 'localStorage', names[nameIndex]));
      if (value) {
        return value;
      }
      value = extractKeyFromStored(readStorage(candidates[index], 'sessionStorage', names[nameIndex]));
      if (value) {
        return value;
      }
    }
  }
  return '';
}

function readKeyFromURL() {
  var sources = [];
  var parent = hostWindow();
  if (parent) {
    try {
      sources.push(parent.location.search);
    } catch (error) {
      /* cross-origin parent */
    }
  }
  sources.push(window.location.search);
  for (var index = 0; index < sources.length; index++) {
    if (!sources[index]) {
      continue;
    }
    var params = new window.URLSearchParams(sources[index]);
    var value = normalizeKey(params.get('key') || params.get('management_key') || params.get('management-key') || '');
    if (value) {
      return value;
    }
  }
  return '';
}

function resolveManagementKey() {
  var key = readOfficialManagementKey();
  if (!key) {
    key = readPlainManagementKey();
  }
  if (!key) {
    key = readKeyFromURL();
  }
  if (!key) {
    key = normalizeKey(readStorage(window, 'sessionStorage', MANUAL_KEY_STORAGE));
  }
  state.managementKey = key;
  return key;
}

function openKeyModal() {
  var input = byId('key-modal-input');
  if (input) {
    input.value = state.managementKey;
  }
  openModal('key-modal');
  if (input) {
    input.focus();
  }
}

function closeKeyModal() {
  closeModal('key-modal');
}

function saveManagementKey() {
  var input = byId('key-modal-input');
  var value = normalizeKey(input ? input.value : '');
  if (!value) {
    showToast(t('error.key_required'), 'error');
    return;
  }
  state.managementKey = value;
  state.authBlocked = false;
  writeStorage(window, 'sessionStorage', MANUAL_KEY_STORAGE, value);
  closeKeyModal();
  refreshAll({});
}

function readHostThemePreference() {
  var raw = readStorage(hostWindow(), 'localStorage', HOST_THEME_STORAGE_KEY);
  if (!raw) {
    raw = readStorage(window, 'localStorage', HOST_THEME_STORAGE_KEY);
  }
  if (!raw) {
    return '';
  }
  try {
    var parsed = JSON.parse(raw);
    var payload = parsed && parsed.state ? parsed.state : parsed;
    return payload && payload.theme ? String(payload.theme) : '';
  } catch (error) {
    return '';
  }
}

function readHostTheme() {
  var hostDoc = hostDocument();
  if (!hostDoc || !hostDoc.documentElement) {
    return '';
  }
  var root = hostDoc.documentElement;
  var body = hostDoc.body;
  var marker = root.getAttribute('data-theme') || (body && body.getAttribute('data-theme'));
  if (marker === 'dark' || marker === 'white' || marker === 'light') {
    return marker;
  }
  if (root.classList.contains('dark') || (body && body.classList.contains('dark'))) {
    return 'dark';
  }
  return 'light';
}

function fallbackTheme() {
  var preference = readHostThemePreference();
  if (preference === 'dark' || preference === 'white' || preference === 'light') {
    return preference;
  }
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }
  return 'light';
}

function cleanInlineThemeStyles() {
  var root = document.documentElement;
  for (var index = 0; index < HOST_THEME_VARIABLES.length; index++) {
    root.style.removeProperty(HOST_THEME_VARIABLES[index]);
  }
  root.style.removeProperty('--bg-surface');
  root.style.removeProperty('--bg-card');
  root.style.removeProperty('--bg-subtle');
}

function applyTheme(theme) {
  var root = document.documentElement;
  cleanInlineThemeStyles();
  if (theme === 'dark' || theme === 'white') {
    root.setAttribute('data-theme', theme);
    return;
  }
  root.removeAttribute('data-theme');
}

function copyHostVariables() {
  var hostDoc = hostDocument();
  if (!hostDoc || !hostDoc.documentElement) {
    return;
  }
  var computed = null;
  try {
    computed = hostWindow().getComputedStyle(hostDoc.documentElement);
  } catch (error) {
    return;
  }
  if (!computed) {
    return;
  }
  var root = document.documentElement;
  for (var index = 0; index < HOST_THEME_VARIABLES.length; index++) {
    var name = HOST_THEME_VARIABLES[index];
    var value = readHostValue(computed, name);
    if (value) {
      root.style.setProperty(name, value);
    } else {
      root.style.removeProperty(name);
    }
  }

  var sec = readHostValue(computed, '--bg-secondary') || readHostValue(computed, '--bg-primary');
  var tert = readHostValue(computed, '--bg-tertiary');
  if (sec) {
    root.style.setProperty('--bg-surface', sec);
    root.style.setProperty('--bg-card', sec);
  }
  if (tert) {
    root.style.setProperty('--bg-subtle', tert);
  }
}

function readHostValue(computed, name) {
  var value = computed.getPropertyValue(name);
  return value && value.trim() ? value.trim() : '';
}

function syncTheme() {
  var theme = readHostTheme();
  var resolved = theme || fallbackTheme();
  applyTheme(resolved);
  if (theme) {
    copyHostVariables();
  }
}
function watchTheme() {
  if (state.themeObserver) {
    state.themeObserver.disconnect();
    state.themeObserver = null;
  }
  var hostDoc = hostDocument();
  if (!hostDoc || !window.MutationObserver) {
    return;
  }
  state.themeObserver = new window.MutationObserver(function () {
    syncTheme();
  });
  state.themeObserver.observe(hostDoc.documentElement, { attributes: true, attributeFilter: ['data-theme', 'class', 'style'] });
  if (hostDoc.body) {
    state.themeObserver.observe(hostDoc.body, { attributes: true, attributeFilter: ['data-theme', 'class', 'style'] });
  }
}

function setTheme(theme) {
  var preference = theme === 'dark' || theme === 'white' || theme === 'light' ? theme : 'auto';
  applyTheme(preference);
  // Standalone and devserver use have no host to remember the choice; keep the
  // host's own envelope so a reload restores it. Inside a host iframe the host
  // owns this key and keeps winning.
  writeStorage(window, 'localStorage', HOST_THEME_STORAGE_KEY, JSON.stringify({ state: { theme: preference }, version: 0 }));
  return preference;
}

function readHostLanguage() {
  var raw = readStorage(hostWindow(), 'localStorage', HOST_LANGUAGE_STORAGE_KEY);
  if (!raw) {
    raw = readStorage(window, 'localStorage', HOST_LANGUAGE_STORAGE_KEY);
  }
  if (!raw) {
    return '';
  }
  var trimmed = raw.trim();
  if (trimmed.charAt(0) !== '{') {
    return trimmed;
  }
  try {
    var parsed = JSON.parse(trimmed);
    var payload = parsed && parsed.state ? parsed.state : parsed;
    return payload && typeof payload.language === 'string' ? payload.language : '';
  } catch (error) {
    return '';
  }
}

function detectLanguage() {
  var stored = readStorage(window, 'localStorage', LANGUAGE_STORAGE_KEY);
  if (stored === 'zh-CN' || stored === 'en-US') {
    return stored;
  }
  var hostLanguage = readHostLanguage();
  if (hostLanguage.indexOf('zh') === 0) {
    return 'zh-CN';
  }
  if (hostLanguage) {
    return 'en-US';
  }
  var browserLanguage = window.navigator.language || '';
  return browserLanguage.indexOf('zh') === 0 ? 'zh-CN' : 'en-US';
}

function t(key) {
  var dictionary = I18N[state.language] || I18N['zh-CN'];
  if (dictionary && dictionary[key]) {
    return dictionary[key];
  }
  var fallback = I18N['zh-CN'] || {};
  return fallback[key] || key;
}

function tf(key, values) {
  var text = t(key);
  if (!values) {
    return text;
  }
  var names = Object.keys(values);
  for (var index = 0; index < names.length; index++) {
    text = text.split('{' + names[index] + '}').join(String(values[names[index]]));
  }
  return text;
}

function applyLanguage() {
  document.documentElement.lang = state.language;
  var nodes = document.querySelectorAll('[data-i18n]');
  for (var index = 0; index < nodes.length; index++) {
    nodes[index].textContent = t(nodes[index].getAttribute('data-i18n'));
  }
  var placeholders = document.querySelectorAll('[data-i18n-placeholder]');
  for (var placeholderIndex = 0; placeholderIndex < placeholders.length; placeholderIndex++) {
    placeholders[placeholderIndex].setAttribute('placeholder', t(placeholders[placeholderIndex].getAttribute('data-i18n-placeholder')));
  }
  var titles = document.querySelectorAll('[data-i18n-title]');
  for (var titleIndex = 0; titleIndex < titles.length; titleIndex++) {
    titles[titleIndex].setAttribute('title', t(titles[titleIndex].getAttribute('data-i18n-title')));
  }
  var ariaLabels = document.querySelectorAll('[data-i18n-aria-label]');
  for (var ariaIndex = 0; ariaIndex < ariaLabels.length; ariaIndex++) {
    ariaLabels[ariaIndex].setAttribute('aria-label', t(ariaLabels[ariaIndex].getAttribute('data-i18n-aria-label')));
  }
  var toggle = byId('lang-toggle');
  if (toggle) {
    toggle.textContent = state.language === 'zh-CN' ? 'EN' : '中文';
  }
}

function toggleLanguage() {
  state.language = state.language === 'zh-CN' ? 'en-US' : 'zh-CN';
  writeStorage(window, 'localStorage', LANGUAGE_STORAGE_KEY, state.language);
  applyLanguage();
  renderKeys();
  renderSettings();
}

function activateTab(name) {
  state.activeTab = name;
  var panels = ['keys', 'settings', 'help'];
  for (var index = 0; index < panels.length; index++) {
    var tab = byId('tab-' + panels[index]);
    var panel = byId('panel-' + panels[index]);
    var active = panels[index] === name;
    if (tab) {
      tab.classList.toggle('active', active);
      tab.setAttribute('aria-selected', active ? 'true' : 'false');
    }
    if (panel) {
      panel.classList.toggle('active', active);
    }
  }
  if (name === 'settings') {
    renderSettings();
  }
}

function openModal(id) {
  var node = byId(id);
  if (node) {
    node.classList.add('open');
  }
}

function closeModal(id) {
  var node = byId(id);
  if (node) {
    node.classList.remove('open');
  }
}

function modalDismiss(event, id) {
  var node = byId(id);
  if (node && event && event.target === node) {
    closeModal(id);
  }
}

function handleEscape(event) {
  if (event.key !== 'Escape') {
    return;
  }
  closeModal('key-modal');
  closeKeyForm();
  closeConfirm();
}

function showToast(message, type) {
  var toast = byId('toast');
  if (!toast) {
    return;
  }
  var variant = type === 'error' ? ' error' : (type === 'success' ? ' success' : '');
  toast.textContent = message;
  toast.className = 'toast visible' + variant;
  if (state.toastTimer) {
    window.clearTimeout(state.toastTimer);
  }
  state.toastTimer = window.setTimeout(function () {
    toast.className = 'toast';
  }, 4200);
}

function setStatus(message, isError) {
  var bar = byId('status-bar');
  if (!bar) {
    return;
  }
  bar.textContent = message || '';
  bar.className = isError ? 'status-bar error-text' : 'status-bar';
}

function formatNumber(value) {
  var number = typeof value === 'number' ? value : Number(value);
  if (!isFinite(number)) {
    return '0';
  }
  return number.toLocaleString();
}

function maskKey(value) {
  if (value.length <= 12) {
    return '••••••••';
  }
  return value.slice(0, 6) + '••••' + value.slice(-4);
}

function apiFetch(path, options) {
  var settings = options || {};
  if (state.authBlocked) {
    openKeyModal();
    return Promise.reject(new Error(t('error.auth_required')));
  }
  if (!state.managementKey) {
    openKeyModal();
    return Promise.reject(new Error(t('error.auth_required')));
  }
  var headers = {
    Authorization: 'Bearer ' + state.managementKey,
    'X-Management-Key': state.managementKey
  };
  if (settings.body) {
    headers['Content-Type'] = 'application/json';
  }
  return window.fetch(path, {
    method: settings.method || 'GET',
    headers: headers,
    body: settings.body,
    cache: 'no-store'
  }).then(function (response) {
    if (response.status === 401 || response.status === 403) {
      state.authBlocked = true;
      openKeyModal();
      throw new Error(t('error.auth_failed'));
    }
    return response.text().then(function (text) {
      var payload = null;
      if (text) {
        try {
          payload = JSON.parse(text);
        } catch (error) {
          payload = null;
        }
      }
      if (!response.ok) {
        var detail = payload && (payload.message || payload.error) ? (payload.message || payload.error) : 'HTTP ' + response.status;
        throw new Error(detail);
      }
      return payload;
    });
  });
}

function openConfirm(message, handler) {
  state.confirmHandler = handler;
  var node = byId('confirm-message');
  if (node) {
    node.textContent = message;
  }
  openModal('confirm-modal');
}

function closeConfirm() {
  state.confirmHandler = null;
  closeModal('confirm-modal');
}

function acceptConfirm() {
  var handler = state.confirmHandler;
  closeConfirm();
  if (typeof handler === 'function') {
    handler();
  }
}

function bindClick(id, handler) {
  var node = byId(id);
  if (node) {
    node.addEventListener('click', handler);
  }
}

function bindSubmit(id, handler) {
  var node = byId(id);
  if (node) {
    node.addEventListener('submit', function (event) {
      event.preventDefault();
      handler(event);
    });
  }
}

function bindBackdrop(id) {
  var node = byId(id);
  if (!node) {
    return;
  }
  var startedOnBackdrop = false;
  node.addEventListener('mousedown', function (event) {
    startedOnBackdrop = event.target === node;
  });
  node.addEventListener('click', function (event) {
    if (startedOnBackdrop && event.target === node) {
      modalDismiss(event, id);
    }
    startedOnBackdrop = false;
  });
}

function bindGlobalEvents() {
  document.addEventListener('keydown', handleEscape);

  bindClick('lang-toggle', toggleLanguage);
  bindClick('key-button', openKeyModal);
  bindClick('refresh-button', function () {
    refreshAll({});
  });
  bindClick('tab-keys', function () {
    activateTab('keys');
  });
  bindClick('tab-settings', function () {
    activateTab('settings');
  });
  bindClick('tab-help', function () {
    activateTab('help');
  });

  bindClick('key-modal-cancel', closeKeyModal);
  bindClick('key-modal-save', saveManagementKey);
  bindBackdrop('key-modal');

  bindSubmit('key-form', saveKeyForm);
  bindClick('key-form-generate', generateKeyIntoForm);
  bindClick('key-form-cancel', closeKeyForm);
  bindBackdrop('key-form-modal');

  bindClick('confirm-cancel', closeConfirm);
  bindClick('confirm-accept', acceptConfirm);
  bindBackdrop('confirm-modal');

  bindClick('keys-add', openAddKeyForm);
  bindClick('keys-unit', cycleTokenUnit);
  var filter = byId('keys-filter');
  if (filter) {
    filter.addEventListener('input', function () {
      renderKeys();
    });
  }

  bindClick('settings-usage-toggle', toggleUsageEnabled);

  if (window.matchMedia) {
    var query = window.matchMedia('(prefers-color-scheme: dark)');
    var listener = function () {
      if (!hostWindow()) {
        syncTheme();
      }
    };
    if (query.addEventListener) {
      query.addEventListener('change', listener);
    }
  }
}

function startAutoRefresh() {
  if (state.refreshTimer) {
    window.clearInterval(state.refreshTimer);
  }
  state.refreshTimer = window.setInterval(function () {
    if (document.hidden || !state.managementKey || state.authBlocked) {
      return;
    }
    refreshAll({ silent: true });
  }, REFRESH_INTERVAL_MS);
}

function initializeApp() {
  state.tokenUnit = readStoredTokenUnit();
  state.language = detectLanguage();
  applyLanguage();
  syncTheme();
  watchTheme();
  resolveManagementKey();
  bindGlobalEvents();
  activateTab('keys');
  refreshAll({ silent: true });
  startAutoRefresh();
}

window.setTheme = setTheme;
`

// templateScriptShellCore is the page runtime with the plugin id resolved.
var templateScriptShellCore = strings.Replace(templateScriptShellCoreTemplate, "{{PLUGIN_ID}}", PluginID, 1)
