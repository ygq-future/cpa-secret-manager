package main

// controlPanelHTML emulates the official management center shell needed to
// develop the plugin page: it applies the CPA theme variables, persists the
// theme and language the same way the official UI does, stores the management
// key in the same obfuscated `cli-proxy-auth` envelope, and renders the plugin
// resource page inside a same-origin iframe.
//
// The theme token values mirror the official stylesheet so the plugin page
// mirrors values a real host actually publishes (see ADR-0003).
const controlPanelHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CPA Host Simulator</title>
<style>
*, *::before, *::after { box-sizing: border-box; }
html, body { height: 100%; margin: 0; }

:root {
  --bg-primary: #f0eee8;
  --bg-secondary: #faf9f5;
  --bg-tertiary: #e9e6df;
  --bg-quinary: #f6f4ee;
  --bg-hover: var(--bg-tertiary);
  --floating-surface: #fffdf9;
  --text-primary: #2d2a26;
  --text-secondary: #6d6760;
  --text-tertiary: #a29c95;
  --text-quaternary: #c0bab3;
  --text-muted: var(--text-tertiary);
  --border-color: #e3e1db;
  --border-primary: #d5d2cb;
  --border-hover: #cecac4;
  --primary-color: #8b8680;
  --primary-hover: #7f7a74;
  --primary-active: #726d67;
  --primary-contrast: #ffffff;
  --success-color: #10b981;
  --warning-color: #c65746;
  --error-color: #c65746;
  --danger-color: var(--error-color);
  --amber-color: #d97706;
  --quota-medium-color: #e0aa14;
  --success-badge-bg: #d1fae5;
  --success-badge-text: #065f46;
  --failure-badge-bg: rgba(198, 87, 70, 0.14);
  --failure-badge-text: #8a3a30;
  --shadow: 0 1px 2px 0 rgb(0 0 0 / 0.08);
  --shadow-lg: 0 10px 18px -3px rgb(0 0 0 / 0.1);
  --radius-md: 8px;
}

[data-theme='white'] {
  --bg-primary: #ffffff;
  --bg-secondary: #ffffff;
  --bg-tertiary: #f6f6f6;
  --bg-quinary: #ffffff;
  --bg-hover: var(--bg-tertiary);
  --floating-surface: #ffffff;
  --text-primary: #2d2a26;
  --text-secondary: #6d6760;
  --text-tertiary: #a29c95;
  --text-quaternary: #c0bab3;
  --text-muted: var(--text-tertiary);
  --border-color: #e5e5e5;
  --border-primary: #d9d9d9;
  --border-hover: #cccccc;
  --primary-color: #8b8680;
  --primary-hover: #7f7a74;
  --primary-active: #726d67;
  --primary-contrast: #ffffff;
  --success-color: #10b981;
  --warning-color: #c65746;
  --error-color: #c65746;
  --danger-color: var(--error-color);
  --amber-color: #d97706;
  --quota-medium-color: #e0aa14;
  --success-badge-bg: #d1fae5;
  --success-badge-text: #065f46;
  --failure-badge-bg: rgba(198, 87, 70, 0.14);
  --failure-badge-text: #8a3a30;
  --shadow: 0 1px 2px 0 rgb(0 0 0 / 0.08);
  --shadow-lg: 0 10px 18px -3px rgb(0 0 0 / 0.1);
  --radius-md: 8px;
}

[data-theme='dark'] {
  --bg-primary: #1d1b18;
  --bg-secondary: #151412;
  --bg-tertiary: #262320;
  --bg-quinary: #191714;
  --bg-hover: #2e2a26;
  --floating-surface: #2a2723;
  --text-primary: #f6f4f1;
  --text-secondary: #c9c3bb;
  --text-tertiary: #9c958d;
  --text-quaternary: #6f6962;
  --text-muted: var(--text-tertiary);
  --border-color: #3a3530;
  --border-primary: #4a453f;
  --border-hover: #5a544d;
  --primary-color: #8b8680;
  --primary-hover: #9a948e;
  --primary-active: #a6a099;
  --primary-contrast: #ffffff;
  --success-color: #10b981;
  --warning-color: #c65746;
  --error-color: #c65746;
  --danger-color: var(--error-color);
  --amber-color: #f59e0b;
  --quota-medium-color: #ffd862;
  --success-badge-bg: rgba(6, 78, 59, 0.3);
  --success-badge-text: #6ee7b7;
  --failure-badge-bg: rgba(198, 87, 70, 0.24);
  --failure-badge-text: #f1b0a6;
  --shadow: 0 1px 3px 0 rgb(0 0 0 / 0.3);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.3);
  --radius-md: 8px;
}

body {
  display: flex;
  flex-direction: column;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font: 13px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
}

header {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-primary);
}

header strong { font-size: 14px; }
header .tag {
  padding: 1px 7px;
  border: 1px solid var(--border-color);
  border-radius: 999px;
  color: var(--text-tertiary);
  font-size: 11px;
}
header .spacer { flex: 1; }

select, input, button {
  padding: 6px 9px;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  color: var(--text-primary);
  font: inherit;
}

button {
  background: var(--primary-color);
  border-color: transparent;
  color: var(--primary-contrast);
  cursor: pointer;
  font-weight: 600;
}

button.secondary {
  background: transparent;
  border-color: var(--border-primary);
  color: var(--text-primary);
}

.status { color: var(--text-tertiary); font-size: 12px; }

main { flex: 1; min-height: 0; padding: 0; }

iframe {
  width: 100%;
  height: 100%;
  border: 0;
  background: var(--bg-secondary);
}
</style>
</head>
<body>
<header>
  <strong>CPA Host Simulator</strong>
  <span class="tag" id="plugin-version">plugin</span>
  <select id="theme-select" title="Host theme">
    <option value="auto">theme: auto</option>
    <option value="light">theme: light</option>
    <option value="white">theme: white</option>
    <option value="dark">theme: dark</option>
  </select>
  <select id="language-select" title="Host language">
    <option value="zh-CN">language: zh-CN</option>
    <option value="en-US">language: en-US</option>
  </select>
  <input id="key-input" type="password" placeholder="management key" size="24">
  <button id="connect">Connect</button>
  <button id="seed" class="secondary">Seed demo data</button>
  <span class="spacer"></span>
  <span class="status" id="status">idle</span>
</header>
<main>
  <iframe id="plugin-frame" allow="clipboard-read; clipboard-write" referrerpolicy="no-referrer"></iframe>
</main>
<script>
(function () {
  'use strict';

  var STORAGE_SALT = 'cli-proxy-api-webui::secure-storage';
  var ENC_PREFIX = 'enc::v1::';

  function byId(id) {
    return document.getElementById(id);
  }

  function setStatus(message) {
    var node = byId('status');
    if (node) {
      node.textContent = message;
    }
  }

  function encodeText(value) {
    return new TextEncoder().encode(value);
  }

  function obfuscate(value) {
    var salt = encodeText(STORAGE_SALT + '|' + window.location.host + '|' + window.navigator.userAgent);
    var bytes = encodeText(value);
    for (var index = 0; index < bytes.length; index++) {
      bytes[index] = bytes[index] ^ salt[index % salt.length];
    }
    var binary = '';
    for (var byteIndex = 0; byteIndex < bytes.length; byteIndex++) {
      binary += String.fromCharCode(bytes[byteIndex]);
    }
    return ENC_PREFIX + window.btoa(binary);
  }

  function zustand(state) {
    return JSON.stringify({ state: state, version: 0 });
  }

  function currentTheme() {
    var select = byId('theme-select');
    return select ? select.value : 'auto';
  }

  function resolveTheme(theme) {
    if (theme === 'dark' || theme === 'white') {
      return theme;
    }
    if (theme === 'light') {
      return 'light';
    }
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'white';
  }

  function applyTheme() {
    var theme = currentTheme();
    var resolved = resolveTheme(theme);
    if (resolved === 'dark' || resolved === 'white') {
      document.documentElement.setAttribute('data-theme', resolved);
    } else {
      document.documentElement.removeAttribute('data-theme');
    }
    window.localStorage.setItem('cli-proxy-theme', zustand({ theme: theme, resolvedTheme: resolved }));
  }

  function applyLanguage() {
    var select = byId('language-select');
    var language = select ? select.value : 'zh-CN';
    window.localStorage.setItem('cli-proxy-language', zustand({ language: language }));
  }

  function storeManagementKey(key) {
    window.localStorage.setItem('cli-proxy-auth', obfuscate(zustand({
      apiBase: window.location.origin,
      managementKey: key,
      rememberPassword: true
    })));
    window.localStorage.setItem('isLoggedIn', 'true');
  }

  function requestJSON(path, options) {
    var settings = options || {};
    var headers = { Authorization: 'Bearer ' + settings.key };
    headers['X-Management-Key'] = settings.key;
    if (settings.body) {
      headers['Content-Type'] = 'application/json';
    }
    return window.fetch(path, {
      method: settings.method || 'GET',
      headers: headers,
      body: settings.body,
      cache: 'no-store'
    }).then(function (response) {
      if (!response.ok) {
        throw new Error('HTTP ' + response.status);
      }
      return response.text().then(function (text) {
        return text ? JSON.parse(text) : null;
      });
    });
  }

  function currentKey() {
    var input = byId('key-input');
    return input ? input.value.trim() : '';
  }

  function connect() {
    var key = currentKey();
    if (!key) {
      setStatus('enter a management key');
      return Promise.reject(new Error('management key required'));
    }
    storeManagementKey(key);
    return requestJSON('/v0/management/plugins', { key: key }).then(function (payload) {
      var plugins = payload && Array.isArray(payload.plugins) ? payload.plugins : [];
      var target = null;
      for (var index = 0; index < plugins.length; index++) {
        var plugin = plugins[index];
        if (!plugin.effectiveEnabled || !Array.isArray(plugin.menus) || plugin.menus.length === 0) {
          continue;
        }
        target = plugin;
        break;
      }
      if (!target) {
        throw new Error('no plugin menu available');
      }
      var versionNode = byId('plugin-version');
      if (versionNode) {
        versionNode.textContent = target.id + ' ' + (target.metadata && target.metadata.version ? target.metadata.version : '');
      }
      var frame = byId('plugin-frame');
      if (frame) {
        frame.src = target.menus[0].path;
      }
      setStatus('connected: ' + target.menus[0].menu);
      return null;
    }).catch(function (error) {
      setStatus('connection failed: ' + error.message);
    });
  }

  function seed() {
    var key = currentKey();
    if (!key) {
      setStatus('enter a management key');
      return;
    }
    requestJSON('/dev/seed?keys=3', { key: key, method: 'POST' }).then(function () {
      setStatus('demo data seeded');
      var frame = byId('plugin-frame');
      if (frame && frame.src) {
        frame.src = frame.src;
      }
    }).catch(function (error) {
      setStatus('seed failed: ' + error.message);
    });
  }

  function loadSettings() {
    try {
      var raw = window.localStorage.getItem('cli-proxy-theme');
      if (raw) {
        var stored = JSON.parse(raw);
        var theme = stored && stored.state ? stored.state.theme : '';
        var select = byId('theme-select');
        if (select && theme) {
          select.value = theme;
        }
      }
      var languageRaw = window.localStorage.getItem('cli-proxy-language');
      if (languageRaw) {
        var parsedLanguage = JSON.parse(languageRaw);
        var language = parsedLanguage && parsedLanguage.state ? parsedLanguage.state.language : '';
        var languageSelect = byId('language-select');
        if (languageSelect && language) {
          languageSelect.value = language;
        }
      }
      var keyInput = byId('key-input');
      if (keyInput && !keyInput.value) {
        keyInput.value = 'devkey';
      }
    } catch (error) {
      setStatus('settings restore failed');
    }
  }

  byId('theme-select').addEventListener('change', applyTheme);
  byId('language-select').addEventListener('change', function () {
    applyLanguage();
  });
  byId('connect').addEventListener('click', function () {
    connect();
  });
  byId('seed').addEventListener('click', seed);
  if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function () {
      if (currentTheme() === 'auto') {
        applyTheme();
      }
    });
  }

  loadSettings();
  applyTheme();
  applyLanguage();
  connect();
})();
</script>
</body>
</html>
`
