package main

import (
	"encoding/json"
	"strings"

	"cpa-secret-manager/internal/management"
)

// hostShellTemplate is the invisible harness shell the simulator serves at
// /control-panel. It publishes the plugin's own theme palette on the parent
// document, stores the simulated management key in the same obfuscated
// `cli-proxy-auth` envelope the official UI writes, and renders the production
// resource page in a same-origin iframe that fills the viewport.
//
// It deliberately draws no chrome: what the simulator shows is exactly what the
// plugin page renders inside a real host, so layout and scroll behaviour can be
// judged 1:1. Theme and language are switched from the console, mirroring the
// `window.setTheme(...)` hook the sibling plugin exposes:
//
//	window.setTheme('auto' | 'light' | 'white' | 'dark')
//	window.setLanguage('zh-CN' | 'en-US')
//	window.seedDemo(3)
const hostShellTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CPA Secret Manager</title>
<style>
html, body { height: 100%; margin: 0; overflow: hidden; }
` + management.ThemePaletteCSS + `
iframe { display: block; width: 100%; height: 100%; border: 0; background: var(--bg-secondary); }
</style>
</head>
<body>
<iframe id="plugin-frame" allow="clipboard-read; clipboard-write" referrerpolicy="no-referrer"></iframe>
<script>
(function () {
  'use strict';

  var STORAGE_SALT = 'cli-proxy-api-webui::secure-storage';
  var ENC_PREFIX = 'enc::v1::';
  var MANAGEMENT_KEY = {{MANAGEMENT_KEY}};
  var PAGE_PATH = {{PAGE_PATH}};

  function byId(id) {
    return document.getElementById(id);
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

  function storeManagementKey() {
    window.localStorage.setItem('cli-proxy-auth', obfuscate(zustand({
      apiBase: window.location.origin,
      managementKey: MANAGEMENT_KEY,
      rememberPassword: true
    })));
    window.localStorage.setItem('isLoggedIn', 'true');
  }

  function reloadFrame() {
    var frame = byId('plugin-frame');
    if (frame) {
      frame.src = PAGE_PATH;
    }
  }

  // setTheme mirrors the official theme store: it marks the parent document the
  // same way the host does (dark/white attribute, default light) and keeps the
  // same cli-proxy-theme envelope, so the plugin page follows through its normal
  // host-variable path.
  window.setTheme = function (theme) {
    var preference = theme === 'dark' || theme === 'white' || theme === 'light' ? theme : 'auto';
    var resolved = preference;
    if (preference === 'auto') {
      resolved = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    if (resolved === 'dark' || resolved === 'white') {
      document.documentElement.setAttribute('data-theme', resolved);
    } else {
      document.documentElement.removeAttribute('data-theme');
    }
    window.localStorage.setItem('cli-proxy-theme', zustand({ theme: preference, resolvedTheme: resolved }));
    return resolved;
  };

  window.setLanguage = function (language) {
    var value = language === 'zh-CN' || language === 'en-US' ? language : 'zh-CN';
    window.localStorage.setItem('cli-proxy-language', zustand({ language: value }));
    reloadFrame();
    return value;
  };

  window.seedDemo = function (count) {
    var keys = typeof count === 'number' && count > 0 ? count : 3;
    return window.fetch('/dev/seed?keys=' + keys, {
      method: 'POST',
      headers: { 'X-Management-Key': MANAGEMENT_KEY }
    }).then(function (response) {
      if (!response.ok) {
        throw new Error('HTTP ' + response.status);
      }
      reloadFrame();
      return 'seeded ' + keys + ' keys';
    });
  };

  storeManagementKey();

  var stored = window.localStorage.getItem('cli-proxy-theme');
  var preference = 'auto';
  if (stored) {
    try {
      var parsed = JSON.parse(stored);
      var payload = parsed && parsed.state ? parsed.state : parsed;
      if (payload && payload.theme) {
        preference = payload.theme;
      }
    } catch (error) {
      preference = 'auto';
    }
  }
  window.setTheme(preference);

  reloadFrame();
})();
</script>
</body>
</html>
`

// hostShellHTML renders the harness shell for one management key.
func hostShellHTML(managementKey string) string {
	key, err := json.Marshal(managementKey)
	if err != nil {
		key = []byte(`""`)
	}
	page, err := json.Marshal("/v0/resource/plugins/" + management.PluginID + management.PathPage)
	if err != nil {
		page = []byte(`"/"`)
	}
	return strings.NewReplacer(
		"{{MANAGEMENT_KEY}}", string(key),
		"{{PAGE_PATH}}", string(page),
	).Replace(hostShellTemplate)
}
