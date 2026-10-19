package management

// templateStyleShell holds the page layout: shell chrome, cards, tables, forms,
// badges and overlays.
//
// Surfaces, text and borders come from the host (--bg-primary canvas,
// --bg-card/--bg-subtle derived from the host's secondary/tertiary surfaces);
// every accent is the page's own token, so buttons and states look identical
// under any host theme. See ThemePaletteCSS and ADR-0003.
//
// The page is a fixed-height flex column: the header, tab bar, summary strip
// and card chrome stay put, and only the table body (or the settings/help
// scroller) scrolls. That mirrors the host dashboard layout and keeps the page
// free of a document-level scrollbar inside the plugin iframe.
const templateStyleShell = `
html, body {
  height: 100%;
  margin: 0;
  padding: 0;
  overflow: hidden;
}

body {
  background: var(--bg-primary);
  color: var(--text-primary);
  font: 14px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", "Helvetica Neue", "PingFang SC", "Microsoft YaHei", sans-serif;
  -webkit-font-smoothing: antialiased;
}

h1, h2, h3, p { margin: 0; }

.icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.muted { color: var(--text-tertiary); }

.app {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  max-width: 1240px;
  height: 100%;
  margin: 0 auto;
  padding: 14px 18px;
}

.app-header {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 0 0 auto;
}

.topbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: var(--radius-md);
  background: var(--accent-blue-subtle);
  color: var(--accent-blue-text);
}

.brand-mark .icon { width: 17px; height: 17px; }

.brand h1 {
  font-size: 19px;
  font-weight: 750;
  letter-spacing: -0.02em;
  white-space: nowrap;
}

.version-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--accent-blue-subtle);
  color: var(--accent-blue-text);
  font-size: 11.5px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.topbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-primary);
  font: inherit;
  font-size: 13px;
  font-weight: 650;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn:disabled { opacity: 0.6; cursor: not-allowed !important; }

.btn-primary {
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
  border-color: transparent;
  font-weight: 500;
  box-shadow: var(--shadow-sm);
}

.btn-primary:hover { background: var(--btn-primary-hover); }
.btn-secondary {
  background: var(--bg-card);
  border-color: var(--border-color);
  color: var(--text-primary);
}

.btn-secondary:hover { background: var(--bg-subtle); }

.btn-quiet {
  background: var(--bg-card);
  border-color: var(--border-color);
  color: var(--text-primary);
}

.btn-quiet:hover {
  background: var(--bg-subtle);
}

.btn-danger {
  background: var(--accent-red-subtle);
  color: var(--accent-red-text);
  border-color: rgba(239, 68, 68, 0.2);
}

.btn-danger:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.35);
}
.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 150ms cubic-bezier(0.23, 1, 0.32, 1), color 150ms, border-color 150ms;
}

.btn-icon:hover {
  background: var(--bg-hover);
  border-color: var(--border-color);
  color: var(--text-primary);
}

.btn-icon.is-danger:hover {
  background: var(--accent-red-subtle);
  border-color: transparent;
  color: var(--accent-red-text);
}

.btn-icon.copied { color: var(--accent-green); }

.btn-icon .icon-check { display: none; }
.btn-icon.copied .icon-check { display: block; }
.btn-icon.copied .icon-copy { display: none; }

.btn:focus-visible,
.btn-icon:focus-visible,
.tab:focus-visible,
.pill-btn:focus-visible {
  outline: none;
  border-color: var(--border-focus);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

input:focus-visible,
textarea:focus-visible,
select:focus-visible {
  outline: none;
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.tabs {
  display: flex;
  gap: 4px;
  flex: 0 0 auto;
  padding: 3px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--bg-subtle);
}

.tab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex: 1 1 0;
  min-width: 0;
  min-height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: calc(var(--radius-lg) - 5px);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 650;
  white-space: nowrap;
  overflow: hidden;
  cursor: pointer;
  transition: background 150ms cubic-bezier(0.23, 1, 0.32, 1), color 150ms, box-shadow 150ms;
}

.tab .icon { width: 14px; height: 14px; opacity: 0.85; }

.tab:hover { color: var(--text-primary); }

.tab.active {
  background: var(--bg-card);
  color: var(--text-primary);
  box-shadow: var(--shadow-sm);
}

.tab.active .icon { opacity: 1; }

main {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0;
}

.panel { display: none; }

.panel.active {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1 1 auto;
  min-height: 0;
}

.scroll-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding-right: 2px;
}

.card {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  overflow: hidden;
}

.scroll-area .card { flex: 0 0 auto; }

.panel > .card { flex: 1 1 auto; }

.card-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 750;
}

.card-sub {
  color: var(--text-tertiary);
  font-size: 12px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  flex: 0 0 auto;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 2px;
  justify-content: center;
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 10px;
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}

.stat-card:hover {
  box-shadow: var(--shadow-md);
  border-color: var(--border-subtle);
}

.stat-label {
  color: var(--text-tertiary);
  font-size: 12px;
  font-weight: 650;
}

.stat-value {
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.stat-hint {
  color: var(--text-tertiary);
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  flex: 0 0 auto;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-color);
}

.toolbar-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.search {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1 1 220px;
  max-width: 340px;
}

.search .icon {
  position: absolute;
  left: 11px;
  color: var(--text-tertiary);
  pointer-events: none;
}

.count-pill {
  color: var(--text-tertiary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

input[type="text"],
input[type="password"],
input[type="search"],
textarea,
select {
  width: 100%;
  min-height: 34px;
  padding: 7px 11px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-primary);
  font: inherit;
  font-size: 13px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
textarea {
  min-height: 76px;
  resize: vertical;
}

.search input { padding-left: 34px; }

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label,
.field-label {
  font-size: 12.5px;
  font-weight: 650;
}

.field .hint {
  color: var(--text-tertiary);
  font-size: 12px;
}

.field-row {
  display: flex;
  gap: 8px;
  align-items: stretch;
}

.field-row input { flex: 1; min-width: 0; }

.hint {
  color: var(--text-tertiary);
  font-size: 12px;
  line-height: 1.5;
}

.table-wrap {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
}

/* The page draws its own scrollbars: the standard scrollbar-* properties would
   make Chromium fall back to the native bar, which on Windows paints stepper
   arrows at both ends. Only the WebKit pseudo-elements are used, and every
   button variant is disabled explicitly. */
*::-webkit-scrollbar,
.table-wrap::-webkit-scrollbar,
.scroll-area::-webkit-scrollbar,
.modal::-webkit-scrollbar,
textarea::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

*::-webkit-scrollbar-track,
.table-wrap::-webkit-scrollbar-track,
.scroll-area::-webkit-scrollbar-track,
.modal::-webkit-scrollbar-track,
textarea::-webkit-scrollbar-track { background: transparent; }

*::-webkit-scrollbar-thumb,
.table-wrap::-webkit-scrollbar-thumb,
.scroll-area::-webkit-scrollbar-thumb,
.modal::-webkit-scrollbar-thumb,
textarea::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: var(--bg-hover);
  background-clip: content-box;
}

*::-webkit-scrollbar-thumb:hover,
.table-wrap::-webkit-scrollbar-thumb:hover,
.scroll-area::-webkit-scrollbar-thumb:hover,
.modal::-webkit-scrollbar-thumb:hover,
textarea::-webkit-scrollbar-thumb:hover {
  background: var(--text-tertiary);
  background-clip: content-box;
}

*::-webkit-scrollbar-button,
.table-wrap::-webkit-scrollbar-button,
.scroll-area::-webkit-scrollbar-button,
.table-wrap::-webkit-scrollbar-button:vertical:start:decrement,
.table-wrap::-webkit-scrollbar-button:vertical:end:increment,
.table-wrap::-webkit-scrollbar-button:horizontal:start:decrement,
.table-wrap::-webkit-scrollbar-button:horizontal:end:increment,
.scroll-area::-webkit-scrollbar-button:vertical:start:decrement,
.scroll-area::-webkit-scrollbar-button:vertical:end:increment,
.scroll-area::-webkit-scrollbar-button:horizontal:start:decrement,
.scroll-area::-webkit-scrollbar-button:horizontal:end:increment {
  display: none;
  width: 0;
  height: 0;
}

*::-webkit-scrollbar-corner,
.table-wrap::-webkit-scrollbar-corner,
.scroll-area::-webkit-scrollbar-corner,
.modal::-webkit-scrollbar-corner,
textarea::-webkit-scrollbar-corner { background: transparent; }

@supports (-moz-appearance: none) {
  *,
  .table-wrap,
  .scroll-area,
  .modal,
  textarea {
    scrollbar-width: thin;
    scrollbar-color: var(--bg-hover) transparent;
  }
}

table.data-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  font-size: 13px;
}

.data-table th,
.data-table td {
  padding: 10px 14px;
  text-align: left;
  vertical-align: middle;
  transition: background-color 0.18s cubic-bezier(0.4, 0, 0.2, 1), color 0.18s ease;
}
.data-table thead th {
  position: sticky;
  top: 0;
  z-index: 2;
  background: var(--bg-subtle);
  color: var(--text-tertiary);
  font-size: 12px;
  font-weight: 650;
  white-space: nowrap;
}

[lang="en-US"] .data-table thead th {
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.data-table tbody td { border-top: 1px solid var(--border-color); }

.data-table tbody tr:first-child td { border-top: 0; }

.data-table tbody tr {
  transition: background-color 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}

.data-table tbody tr:hover td { background-color: var(--bg-hover) !important; }

.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.num-strong { font-weight: 650; }

.key-value {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: var(--bg-subtle);
  font-family: var(--font-mono);
  font-size: 12.5px;
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.row-actions {
  display: inline-flex;
  gap: 2px;
  align-items: center;
}

.remark {
  max-width: 280px;
  word-break: break-word;
}

.pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 26px;
  padding: 0 9px;
  border: 1px solid var(--border-color);
  border-radius: 999px;
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-size: 12px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background 150ms cubic-bezier(0.23, 1, 0.32, 1), color 150ms, border-color 150ms;
}

.pill-btn:hover {
  background: var(--bg-hover);
  border-color: var(--border-focus);
  color: var(--text-primary);
}

.pill-btn[aria-expanded="true"] {
  background: var(--accent-blue-subtle);
  border-color: transparent;
  color: var(--accent-blue-text);
}

.pill-btn .icon { width: 12px; height: 12px; transition: transform 150ms cubic-bezier(0.23, 1, 0.32, 1); }
.pill-btn[aria-expanded="true"] .icon { transform: rotate(180deg); }

.empty-state {
  padding: 44px 20px;
  color: var(--text-tertiary);
  font-size: 13.5px;
  text-align: center;
}

.badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 7px;
  border-radius: var(--radius-sm);
  font-size: 11.5px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.badge-danger {
  background: var(--accent-red-subtle);
  color: var(--accent-red-text);
}

.badge-remark {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
  background: var(--accent-blue-subtle);
  color: var(--accent-blue-text);
  border: 1px solid rgba(59, 130, 246, 0.22);
}

.switch-row {
  display: flex;
  gap: 16px;
  align-items: center;
  justify-content: space-between;
  margin-top: 2px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.switch-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.switch-label { font-size: 13px; font-weight: 650; }

.switch-control {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.toggle-state {
  color: var(--text-secondary);
  font-size: 12.5px;
  font-weight: 650;
  white-space: nowrap;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  flex: 0 0 auto;
  width: 40px;
  height: 22px;
}

.toggle-switch input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
}

.toggle-slider {
  position: absolute;
  inset: 0;
  border-radius: 999px;
  background: var(--bg-subtle);
  box-shadow: inset 0 0 0 1px var(--border-color);
  cursor: pointer;
  transition: background 250ms cubic-bezier(0.23, 1, 0.32, 1), box-shadow 250ms;
}

.toggle-slider::before {
  content: "";
  position: absolute;
  left: 3px;
  bottom: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--switch-knob);
  box-shadow: var(--shadow-sm);
  transition: transform 250ms cubic-bezier(0.23, 1, 0.32, 1);
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent-green);
  box-shadow: inset 0 0 0 1px var(--accent-green);
}

.toggle-switch input:checked + .toggle-slider::before { transform: translateX(18px); }

.toggle-switch input:focus-visible + .toggle-slider {
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.kv {
  display: grid;
  grid-template-columns: minmax(120px, 180px) 1fr;
  gap: 10px 16px;
  margin: 0;
  font-size: 13px;
}

.kv dt {
  color: var(--text-tertiary);
  font-size: 12.5px;
  font-weight: 650;
}

.kv dd {
  margin: 0;
  word-break: break-all;
}

.kv dd.path { font-family: var(--font-mono); font-size: 12.5px; }

.alert {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: var(--accent-red-subtle);
  color: var(--accent-red-text);
  font-size: 12.5px;
  line-height: 1.5;
}

.alert .icon { margin-top: 1px; }

.status-bar {
  display: flex;
  align-items: center;
  gap: 7px;
  flex: 0 0 auto;
  min-height: 18px;
  color: var(--text-tertiary);
  font-size: 12px;
}

.status-bar::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.6;
}

.status-bar:empty::before { display: none; }

.status-bar.error-text { color: var(--accent-red-text); }

.help-list {
  display: flex;
  flex-direction: column;
  gap: 9px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.help-list li {
  position: relative;
  padding-left: 18px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.help-list li::before {
  content: "";
  position: absolute;
  top: 8px;
  left: 2px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--accent-blue);
  opacity: 0.7;
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: none;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: var(--bg-overlay);
}

.modal-backdrop.open {
  display: flex;
  animation: overlayIn 160ms ease-out;
}

.modal {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: min(560px, 100%);
  max-height: 86vh;
  overflow-y: auto;
  padding: 20px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  box-shadow: var(--shadow-lg);
  animation: modalIn 200ms cubic-bezier(0.16, 1, 0.3, 1);
}

.modal h2 {
  font-size: 16px;
  font-weight: 750;
  letter-spacing: -0.01em;
}

.modal form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.modal-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 4px;
}

.toast {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 60;
  max-width: min(420px, calc(100% - 40px));
  padding: 11px 15px;
  border: 1px solid var(--border-color);
  border-left: 4px solid var(--accent-blue);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  box-shadow: var(--shadow-lg);
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
  opacity: 0;
  transform: translateY(10px) scale(0.98);
  transition: opacity 200ms cubic-bezier(0.23, 1, 0.32, 1), transform 200ms cubic-bezier(0.23, 1, 0.32, 1);
  pointer-events: none;
}

.toast.visible {
  opacity: 1;
  transform: none;
}

.toast.success { border-left-color: var(--accent-green); }
.toast.error { border-left-color: var(--accent-red); }

@keyframes overlayIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes modalIn {
  from { opacity: 0; transform: translateY(12px) scale(0.97); }
  to { opacity: 1; transform: none; }
}

[hidden] { display: none !important; }
`

// templateStyleResponsive adapts the shell to narrow viewports.
const templateStyleResponsive = `
@media (max-width: 960px) {
  .stats-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}

@media (max-width: 720px) {
  .app { padding: 12px; gap: 10px; }
  .brand h1 { font-size: 16px; }
  .tab { padding: 0 8px; }
  .tab .icon { display: none; }
  .toolbar { align-items: stretch; }
  .search { max-width: none; }
  .toolbar-actions { justify-content: space-between; }
  .stats-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .stat-value { font-size: 19px; }
  .stat-hint { display: none; }
  .kv { grid-template-columns: 1fr; gap: 4px 0; }
  .kv dd { margin-bottom: 8px; }
  .data-table th, .data-table td { padding: 9px 11px; }
  .remark { max-width: 160px; }
}

@media (prefers-reduced-motion: reduce) {
  .btn, .btn-icon, .pill-btn, .tab, .data-table th, .data-table td,
  .toggle-slider, .toggle-slider::before, .toast, .modal, .modal-backdrop.open {
    animation: none;
    transition: none;
  }
}

input[readonly] {
  background: var(--bg-secondary) !important;
  color: var(--text-primary);
  cursor: default;
}

.badge-clickable {
  cursor: pointer;
  transition: transform 0.12s ease, box-shadow 0.12s ease;
}
.badge-clickable:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.failures-box {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 12px 14px;
  margin: 12px 0;
  font-size: 13px;
}
.failures-summary {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-weight: 600;
  color: var(--text-primary);
}
.failures-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 180px;
  overflow-y: auto;
}
.failures-list-item {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  font-size: 12.5px;
}
.failures-list-item:last-child {
  border-bottom: none;
}
.failures-tip {
  margin-top: 10px;
  font-size: 12px;
  color: var(--text-tertiary);
  line-height: 1.5;
}

.date-range-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 140px;
  font-variant-numeric: tabular-nums;
}

.date-range-popover {
  width: min(740px, calc(100vw - 32px));
  max-height: calc(100vh - 48px);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.45);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: modalIn 180ms cubic-bezier(0.2, 0.9, 0.3, 1);
}

.date-dialog-body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;
}

.date-dialog-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}

.date-dialog-title {
  font-size: 17px;
  font-weight: 650;
  margin: 0;
  color: var(--text-primary);
}

.date-selection-status {
  font-size: 13px;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
  margin-top: 4px;
}

.date-presets {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.btn-preset {
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 9999px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-preset:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.btn-preset.active {
  background: var(--accent-blue);
  border-color: var(--accent-blue);
  color: #fff;
}

.calendar-panels {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px;
}

@media (max-width: 640px) {
  .calendar-panels {
    grid-template-columns: 1fr;
  }
}

.calendar-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.calendar-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  text-align: center;
}

.calendar-week {
  font-size: 11.5px;
  color: var(--text-tertiary);
  padding: 4px 0;
  user-select: none;
}

.calendar-day {
  height: 32px;
  line-height: 32px;
  font-size: 12.5px;
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-primary);
  user-select: none;
  transition: background 0.12s ease;
}

.calendar-day:hover:not(.empty) {
  background: var(--bg-hover);
}

.calendar-day.empty {
  cursor: default;
  opacity: 0;
}

.calendar-day.range-start,
.calendar-day.range-end {
  background: var(--accent-blue) !important;
  color: #fff !important;
  font-weight: 600;
}

.calendar-day.in-range {
  background: rgba(99, 102, 241, 0.18);
  border-radius: 0;
}

.time-pickers-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.time-picker-col {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.time-picker-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
}

.time-input {
  font-variant-numeric: tabular-nums;
  text-align: center;
  font-size: 13px;
  height: 34px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.date-dialog-hint {
  font-size: 12px;
  color: var(--text-tertiary);
}

.date-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
.modal-header-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.modal-header-row h2 {
  margin: 0;
}

.failures-range-tag {
  font-size: 12.5px;
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

#failures-modal .modal {
  width: min(720px, calc(100vw - 32px));
  max-height: 88vh;
  overflow: visible;
}

.failures-filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: nowrap;
}

.custom-dropdown {
  position: relative;
  flex: 1 1 0;
  min-width: 0;
  max-width: 280px;
}

.custom-dropdown-btn {
  width: 100%;
  height: 32px;
  padding: 5px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12.5px;
  font-family: inherit;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  background-color: var(--bg-card);
  color: var(--text-primary);
  cursor: pointer;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
  user-select: none;
}

.custom-dropdown-btn:hover {
  border-color: var(--text-secondary);
}

.custom-dropdown.open .custom-dropdown-btn,
.custom-dropdown-btn:focus-visible {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--focus-ring);
  outline: none;
}

.custom-dropdown-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}

.custom-dropdown-arrow {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  color: var(--text-tertiary);
  transition: transform 0.15s ease;
}

.custom-dropdown.open .custom-dropdown-arrow {
  transform: rotate(180deg);
}

.custom-dropdown-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  min-width: 100%;
  width: max-content;
  max-width: 380px;
  z-index: 100;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.55);
  max-height: 220px;
  overflow-y: auto;
  padding: 4px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  box-sizing: border-box;
}

.custom-dropdown-option {
  padding: 8px 12px;
  font-size: 13px;
  line-height: 18px;
  min-height: 34px;
  box-sizing: border-box;
  border-radius: 4px;
  color: var(--text-primary);
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  user-select: none;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  transition: background 0.12s ease, color 0.12s ease;
}

.custom-dropdown-option:hover {
  background: var(--bg-hover);
}

.custom-dropdown-option.selected {
  background: rgba(99, 102, 241, 0.15);
  color: var(--accent-blue);
  font-weight: 600;
}
.failures-list-wrap {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 8px 0;
  max-height: 420px;
  overflow-y: auto;
  padding-right: 4px;
}

.failure-item {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  transition: background 0.12s ease;
}

.failure-item:hover {
  background: var(--bg-hover);
}

.failure-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.failure-meta-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.failure-time {
  font-size: 11.5px;
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
}

.failure-model-badge {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.failure-status-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 4px;
}

.failure-body {
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-primary);
  word-break: break-all;
  white-space: pre-wrap;
  font-family: var(--font-mono);
  max-height: 80px;
  overflow-y: auto;
}
`
