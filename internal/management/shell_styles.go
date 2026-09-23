package management

// templateStyleShell holds the page layout: shell chrome, cards, tables, forms,
// badges and overlays. All colours come from the CPA host theme variables.
const templateStyleShell = `
html, body {
  margin: 0;
  padding: 0;
  min-height: 100%;
}

body {
  background: var(--bg-secondary);
  color: var(--text-primary);
  font: 14px/1.55 -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  -webkit-font-smoothing: antialiased;
}

.app {
  max-width: 1240px;
  margin: 0 auto;
  padding: 20px 18px 40px;
}

.topbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-color);
}

.brand {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
}

.brand h1 {
  margin: 0;
  font-size: 19px;
  font-weight: 650;
  letter-spacing: 0.01em;
}

.version-badge {
  padding: 1px 7px;
  border: 1px solid var(--border-color);
  border-radius: 999px;
  color: var(--text-tertiary);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.topbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.btn {
  padding: 7px 13px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: var(--primary-color);
  color: var(--primary-contrast);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  transition: background 200ms cubic-bezier(0.23, 1, 0.32, 1), border-color 200ms;
}

.btn:hover { background: var(--primary-hover); }
.btn:active { background: var(--primary-active); }

.btn-ghost {
  background: transparent;
  border-color: var(--border-primary);
  color: var(--text-primary);
}

.btn-ghost:hover { background: var(--bg-hover); }

.btn-danger {
  background: transparent;
  border-color: var(--border-primary);
  color: var(--danger-color);
}

.btn-danger:hover { background: var(--bg-hover); }

.btn-link {
  padding: 2px 4px;
  border: 0;
  background: transparent;
  color: var(--primary-color);
  font: inherit;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.btn-link:hover { color: var(--primary-active); }

.tabs {
  display: flex;
  gap: 4px;
  margin: 16px 0 18px;
  padding: 3px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-primary);
  overflow-x: auto;
}

.tab {
  flex: 0 0 auto;
  padding: 7px 15px;
  border: 0;
  border-radius: calc(var(--radius-md) - 2px);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.tab:hover { color: var(--text-primary); }

.tab.active {
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: var(--shadow);
}

.panel { display: none; }
.panel.active { display: block; }

.card {
  padding: 16px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-primary);
  box-shadow: var(--shadow);
}

.card + .card { margin-top: 14px; }

.card h2 {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 650;
}

.card .hint {
  margin: 0 0 12px;
  color: var(--text-tertiary);
  font-size: 12px;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.toolbar-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

input[type="text"], input[type="password"], input[type="search"], textarea, select {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  background: var(--bg-secondary);
  color: var(--text-primary);
  font: inherit;
}

input:focus, textarea:focus, select:focus {
  outline: none;
  border-color: var(--primary-active);
}

textarea {
  min-height: 70px;
  resize: vertical;
}

.filter-input { max-width: 320px; }

.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 12px;
}

.field label {
  font-weight: 600;
  font-size: 13px;
}

.field .hint {
  margin: 0;
  color: var(--text-tertiary);
  font-size: 12px;
}

.field-row {
  display: flex;
  gap: 8px;
  align-items: stretch;
}

.field-row input { flex: 1; min-width: 0; }

.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

th, td {
  padding: 9px 11px;
  border-bottom: 1px solid var(--border-color);
  text-align: left;
  vertical-align: top;
}

thead th {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 650;
  white-space: nowrap;
}

tbody tr:last-child td { border-bottom: 0; }

tbody tr:hover td { background: var(--bg-hover); }

.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.key-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  word-break: break-all;
}

.row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  white-space: nowrap;
}

.remark {
  max-width: 260px;
  word-break: break-word;
}

.muted { color: var(--text-tertiary); }
.error-text { color: var(--danger-color); }

.detail-row td {
  background: var(--bg-secondary);
  padding: 0;
}

.detail-body {
  padding: 12px 14px;
}

.detail-body h3 {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 650;
}

.badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
}

.badge-success {
  background: var(--success-badge-bg);
  color: var(--success-badge-text);
}

.badge-failure {
  background: var(--failure-badge-bg);
  color: var(--failure-badge-text);
}

.status-bar {
  margin-top: 14px;
  min-height: 20px;
  color: var(--text-tertiary);
  font-size: 12px;
}

.note {
  margin: 10px 0 0;
  color: var(--text-tertiary);
  font-size: 12px;
}

.switch-row {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  justify-content: space-between;
  padding: 10px 0;
  border-top: 1px solid var(--border-color);
}

.switch-row:first-of-type { border-top: 0; }

.switch-row .switch-label { font-weight: 600; }

.switch {
  position: relative;
  flex: 0 0 auto;
  width: 42px;
  height: 24px;
  border: 1px solid var(--border-primary);
  border-radius: 999px;
  background: var(--bg-tertiary);
  cursor: pointer;
  transition: background 200ms cubic-bezier(0.23, 1, 0.32, 1);
}

.switch::after {
  content: "";
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--bg-secondary);
  box-shadow: var(--shadow);
  transition: transform 200ms cubic-bezier(0.23, 1, 0.32, 1);
}

.switch[aria-checked="true"] {
  background: var(--success-color);
  border-color: var(--success-color);
}

.switch[aria-checked="true"]::after { transform: translateX(18px); }

.modal-backdrop {
  position: fixed;
  inset: 0;
  display: none;
  align-items: center;
  justify-content: center;
  padding: 18px;
  background: rgb(0 0 0 / 0.42);
  z-index: 40;
}

.modal-backdrop.open { display: flex; }

.modal {
  width: min(560px, 100%);
  max-height: 86vh;
  overflow-y: auto;
  padding: 20px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--floating-surface);
  box-shadow: var(--shadow-lg);
}

.modal h2 {
  margin: 0 0 12px;
  font-size: 16px;
}

.modal-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 16px;
}

.toast {
  position: fixed;
  right: 18px;
  bottom: 18px;
  max-width: min(420px, calc(100% - 36px));
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--floating-surface);
  box-shadow: var(--shadow-lg);
  color: var(--text-primary);
  font-size: 13px;
  opacity: 0;
  transform: translateY(8px);
  transition: opacity 200ms cubic-bezier(0.23, 1, 0.32, 1), transform 200ms cubic-bezier(0.23, 1, 0.32, 1);
  pointer-events: none;
  z-index: 60;
}

.toast.visible {
  opacity: 1;
  transform: translateY(0);
}

.toast.error { border-color: var(--danger-color); color: var(--danger-color); }

.help-list {
  margin: 0;
  padding-left: 20px;
  color: var(--text-secondary);
}

.help-list li { margin-bottom: 6px; }

.kv {
  display: grid;
  grid-template-columns: minmax(120px, 200px) 1fr;
  gap: 6px 14px;
  margin: 0;
  font-size: 13px;
}

.kv dt { color: var(--text-secondary); font-weight: 600; }
.kv dd { margin: 0; word-break: break-all; }
`

// templateStyleResponsive adapts the shell to narrow viewports.
const templateStyleResponsive = `
@media (max-width: 720px) {
  .app { padding: 14px 12px 32px; }
  .brand h1 { font-size: 17px; }
  .toolbar { align-items: stretch; flex-direction: column; }
  .filter-input { max-width: none; }
  .kv { grid-template-columns: 1fr; }
  .kv dt { margin-top: 6px; }
  th, td { padding: 8px 9px; }
}

@media (prefers-reduced-motion: reduce) {
  .btn, .switch, .switch::after, .toast { transition: none; }
}
`
