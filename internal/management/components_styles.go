package management

// This file declares the page's visual layer as two strings: the element
// normalization that keeps native widgets from ever showing through, and the
// styles of every reusable component. Components never hardcode a colour: they
// consume the tokens declared in ThemePaletteCSS, and TestPage_RulesUseTokensOnly
// keeps the rules free of colour literals.

const templateStyleBase = `
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

button,
input,
textarea {
  font: inherit;
  color: inherit;
}

button {
  margin: 0;
  padding: 0;
  border: 0;
  background: none;
  cursor: pointer;
}

input[type="number"]::-webkit-outer-spin-button,
input[type="number"]::-webkit-inner-spin-button {
  -webkit-appearance: none;
  appearance: none;
  margin: 0;
}

input[type="number"] {
  -moz-appearance: textfield;
  appearance: textfield;
}

input[type="search"]::-webkit-search-decoration,
input[type="search"]::-webkit-search-cancel-button,
input[type="search"]::-webkit-search-results-button,
input[type="search"]::-webkit-search-results-decoration {
  -webkit-appearance: none;
  appearance: none;
  display: none;
}

input[type="search"] {
  -webkit-appearance: none;
  appearance: none;
}

input[type="password"]::-webkit-credentials-auto-fill-button,
input[type="password"]::-webkit-contacts-auto-fill-button {
  display: none !important;
  visibility: hidden;
  pointer-events: none;
}

select {
  -webkit-appearance: none;
  appearance: none;
}

input:-webkit-autofill,
input:-webkit-autofill:hover,
input:-webkit-autofill:focus {
  -webkit-text-fill-color: var(--text-primary);
  -webkit-box-shadow: 0 0 0 100px var(--bg-card) inset;
  box-shadow: 0 0 0 100px var(--bg-card) inset;
  caret-color: var(--text-primary);
}

::placeholder {
  color: var(--text-tertiary);
  opacity: 1;
}

::selection {
  background: var(--accent-blue-soft);
  color: var(--text-primary);
}

[hidden] { display: none !important; }

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
  border-radius: var(--radius-pill);
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
`

const templateStyleComponents = `
.icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.muted { color: var(--text-tertiary); }

.hint {
  color: var(--text-tertiary);
  font-size: 12px;
  line-height: 1.5;
}

.is-invisible { visibility: hidden; }

.row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

/* --- buttons --- */
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
  font-size: 13px;
  font-weight: 650;
  line-height: 1;
  white-space: nowrap;
  transition: background 150ms var(--ease-emphasized), border-color 150ms, color 150ms, box-shadow 150ms;
}

.btn:disabled { opacity: 0.6; cursor: not-allowed !important; }

.btn-primary {
  background: var(--btn-primary-bg);
  color: var(--btn-primary-text);
  border-color: transparent;
  box-shadow: var(--shadow-sm);
}

.btn-primary:hover { background: var(--btn-primary-hover); }

.btn-secondary {
  background: var(--bg-card);
  border-color: var(--border-color);
  color: var(--text-primary);
}

.btn-secondary:hover { background: var(--bg-subtle); }

.btn-danger {
  background: var(--accent-red-subtle);
  color: var(--accent-red-text);
  border-color: var(--accent-red-border);
}

.btn-danger:hover {
  background: var(--accent-red-soft);
  border-color: var(--accent-red-border);
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  transition: background 150ms var(--ease-emphasized), color 150ms, border-color 150ms;
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

.btn-text {
  color: var(--accent-blue-text);
  font-size: 12px;
  font-weight: 650;
  transition: opacity 120ms;
}

.btn-text:hover { opacity: 0.75; }

a.btn { text-decoration: none; }

/* A button that owns an in-flight request swaps its glyph for the shared
   spinner, so refresh feedback never depends on a toast or the status bar. */
.btn .spinner {
  display: none;
  width: 13px;
  height: 13px;
  border-width: 2px;
}

.btn.is-loading .spinner { display: block; }
.btn.is-loading .icon { display: none; }

/* --- pills: small rounded actions and preset selectors --- */
.pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 26px;
  padding: 0 11px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-pill);
  background: var(--bg-subtle);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  transition: background 150ms var(--ease-emphasized), color 150ms, border-color 150ms, transform 150ms;
}

.pill:hover {
  background: var(--bg-hover);
  border-color: var(--border-focus);
  color: var(--text-primary);
}

.pill:active { transform: scale(0.96); }

.pill.is-active {
  background: var(--accent-blue-subtle);
  border-color: transparent;
  color: var(--accent-blue-text);
  box-shadow: var(--shadow-sm);
}

.action-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 26px;
  padding: 0 9px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-pill);
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  transition: background 150ms var(--ease-emphasized), color 150ms, border-color 150ms;
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

.pill-btn .icon { width: 12px; height: 12px; transition: transform 150ms var(--ease-emphasized); }
.pill-btn[aria-expanded="true"] .icon { transform: rotate(180deg); }

/* --- tabs --- */
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
  border-radius: calc(var(--radius-lg) - 5px);
  background: transparent;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 650;
  white-space: nowrap;
  overflow: hidden;
  transition: background 150ms var(--ease-emphasized), color 150ms, box-shadow 150ms;
}

.tab .icon { width: 14px; height: 14px; opacity: 0.85; }

.tab:hover { color: var(--text-primary); }

.tab.active {
  background: var(--bg-card);
  color: var(--text-primary);
  box-shadow: var(--shadow-sm);
}

.tab.active .icon { opacity: 1; }

/* --- focus rings --- */
.btn:focus-visible,
.btn-icon:focus-visible,
.tab:focus-visible,
.pill:focus-visible,
.pill-btn:focus-visible,
.chip:focus-visible,
.btn-text:focus-visible,
.search-clear:focus-visible,
.select-trigger:focus-visible,
.select-option:focus-visible,
.calendar-day:focus-visible {
  outline: none;
  border-color: var(--border-focus);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

input:focus-visible,
textarea:focus-visible {
  outline: none;
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

/* --- cards --- */
.card {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  overflow: hidden;
}

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

.quota-card {
  margin-top: 14px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  overflow: hidden;
  transition: border-color 150ms, box-shadow 150ms;
}

.quota-switch-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 0;
  padding: 12px 14px;
  background: var(--bg-subtle);
  border: none;
}

.quota-switch-header .switch-label {
  font-size: 13px;
  font-weight: 650;
  color: var(--text-primary);
}

.quota-switch-header .hint {
  margin: 0;
  font-size: 11.5px;
}

.quota-fields {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  border-top: 1px solid var(--border-color);
}

.quota-status-banner {
  padding: 9px 12px;
  border-radius: var(--radius-md);
  font-size: 12px;
  line-height: 1.4;
  background: var(--bg-subtle);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
}

.quota-status-banner.is-exceeded {
  background: var(--accent-red-soft);
  border-color: var(--accent-red-border);
  color: var(--accent-red-text);
  font-weight: 650;
}

/* --- stats --- */
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

.stat-card.is-actionable { cursor: pointer; }

.stat-card.is-danger .stat-value { color: var(--accent-red-text); }

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

/* --- fields --- */
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

.field .hint { font-size: 12px; }

.field-row {
  display: flex;
  gap: 8px;
  align-items: stretch;
}

.field-row input { flex: 1; min-width: 0; }

.flex-1 {
  flex: 1 1 0;
  min-width: 0;
}

input[type="text"],
input[type="password"],
input[type="search"],
input[type="number"],
textarea {
  width: 100%;
  min-height: 34px;
  padding: 7px 11px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 13px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

textarea {
  min-height: 76px;
  resize: vertical;
}

input[readonly] {
  background: var(--bg-secondary) !important;
  color: var(--text-primary);
  cursor: default;
}

/* --- search field --- */
.search {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1 1 220px;
  max-width: 340px;
}

/* Only the leading glyph is pinned; the clear button centres its own cross.
   A descendant selector would absolutely position that cross too and drag it
   out of the button. */
.search > .icon {
  position: absolute;
  left: 11px;
  color: var(--text-tertiary);
  pointer-events: none;
}

.search input { padding-left: 34px; padding-right: 32px; }

/* The filter is read-only until first interaction to block credential
   autofill; that guard must not change how the field looks. */
.search input[readonly] {
  background: var(--bg-card) !important;
  cursor: text;
}

.search-clear {
  position: absolute;
  right: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: var(--radius-pill);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  transition: background 120ms, color 120ms;
}

.search-clear:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.search-clear .icon {
  width: 11px;
  height: 11px;
  position: static;
  color: inherit;
}

/* --- input with unit --- */
.input-unit-group {
  display: flex;
  align-items: stretch;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  transition: border-color 150ms, box-shadow 150ms;
}

.input-unit-group:focus-within {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.input-unit-group .unit-input {
  flex: 1 1 auto;
  min-width: 0;
  border: none;
  border-radius: var(--radius-md) 0 0 var(--radius-md);
  background: transparent;
  padding: 8px 12px;
  font-size: 14px;
  box-shadow: none;
}

.input-unit-group .unit-input:focus-visible { box-shadow: none; }

.input-unit-group .unit-suffix {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  padding: 0 12px;
  border-left: 1px solid var(--border-color);
  background: var(--bg-subtle);
  color: var(--text-secondary);
  font-size: 12.5px;
  font-weight: 600;
  white-space: nowrap;
  border-radius: 0 var(--radius-md) var(--radius-md) 0;
}

.input-unit-group .select {
  flex: 0 0 auto;
  display: flex;
  align-items: stretch;
}

.input-unit-group .select-trigger {
  height: 100%;
  padding: 0 10px 0 12px;
  border: 0;
  border-left: 1px solid var(--border-color);
  border-radius: 0 var(--radius-md) var(--radius-md) 0;
  background: var(--bg-subtle);
  color: var(--text-primary);
  font-size: 12.5px;
  font-weight: 650;
  gap: 6px;
}

.input-unit-group .select-trigger:hover { background: var(--bg-tertiary); }

.input-unit-group .select-menu {
  left: auto;
  right: 0;
  min-width: 120px;
}

/* --- quota limit row & inline calc --- */
.quota-limit-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.quota-limit-row .input-unit-group {
  flex: 1 1 auto;
  min-width: 0;
}

.quota-calc-inline {
  flex: 0 0 auto;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 550;
  white-space: nowrap;
}

.quota-calc-inline[hidden] {
  display: none !important;
}
/* --- listbox (the page's own <select>) --- */
.select {
  position: relative;
  min-width: 0;
}

.select-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  min-height: 34px;
  padding: 5px 10px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 12.5px;
  transition: border-color 150ms, box-shadow 150ms, background 150ms;
  user-select: none;
}

.select-trigger:hover { border-color: var(--text-secondary); }

.select.is-open .select-trigger {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.select-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}

.select-arrow {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  color: var(--text-tertiary);
  transition: transform 150ms var(--ease-emphasized);
}

.select.is-open .select-arrow { transform: rotate(180deg); }

.select-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 100%;
  width: max-content;
  max-width: 380px;
  max-height: 220px;
  overflow-y: auto;
  padding: 4px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  box-shadow: var(--shadow-lg);
}

.select-menu.is-dropup {
  top: auto;
  bottom: calc(100% + 4px);
}

.select-option {
  display: flex;
  align-items: center;
  min-height: 34px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: 13px;
  line-height: 18px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
  user-select: none;
  transition: background 120ms, color 120ms;
}

.select-option:hover,
.select-option.is-active { background: var(--bg-hover); }

.select-option.is-selected {
  background: var(--accent-blue-soft);
  color: var(--accent-blue-text);
  font-weight: 650;
}

/* --- toggle switch --- */
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
  border-radius: var(--radius-pill);
  /* The track owns its own colours: --bg-subtle would vanish on the subtler
     surfaces a switch is placed on (the quota header is --bg-subtle). */
  background: var(--switch-track);
  box-shadow: inset 0 0 0 1px var(--switch-track-border);
  cursor: pointer;
  transition: background 250ms var(--ease-emphasized), box-shadow 250ms;
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
  transition: transform 250ms var(--ease-emphasized);
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent-green);
  box-shadow: inset 0 0 0 1px var(--accent-green);
}

.toggle-switch input:checked + .toggle-slider::before { transform: translateX(18px); }

.toggle-switch input:focus-visible + .toggle-slider {
  box-shadow: 0 0 0 3px var(--focus-ring);
}

/* --- multi-select searchable combobox --- */
.multi-select {
  position: relative;
  min-width: 0;
  width: 100%;
}

.multi-select-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  height: 38px;
  min-height: 38px;
  padding: 0 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: border-color 150ms, box-shadow 150ms, background 150ms;
  user-select: none;
}

.multi-select-trigger:hover {
  border-color: var(--text-secondary);
}

.multi-select.is-open .multi-select-trigger {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.multi-select-display {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.multi-select-label {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary);
}

.multi-select-label.is-placeholder {
  color: var(--text-tertiary);
}

.multi-select-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 7px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 650;
  background: var(--accent-blue-soft);
  color: var(--accent-blue-text);
  white-space: nowrap;
  flex: 0 0 auto;
}

.multi-select-arrow {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  color: var(--text-tertiary);
  transition: transform 150ms var(--ease-emphasized);
}

.multi-select.is-open .multi-select-arrow {
  transform: rotate(180deg);
}

.multi-select-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  z-index: 120;
  display: flex;
  flex-direction: column;
  padding: 8px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  box-shadow: var(--shadow-lg);
  gap: 6px;
}

.multi-select-menu.is-dropup {
  top: auto;
  bottom: calc(100% + 4px);
}

.multi-select-search-wrap {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
}

.multi-select-search-wrap > .icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  width: 14px;
  height: 14px;
  color: var(--text-tertiary);
  pointer-events: none;
}

.multi-select-search-wrap input[type="text"].multi-select-search-input,
input[type="text"].multi-select-search-input {
  width: 100%;
  height: 32px;
  min-height: 32px;
  padding: 0 30px 0 32px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-subtle);
  color: var(--text-primary);
  font-size: 12.5px;
  outline: none;
  transition: border-color 150ms, box-shadow 150ms, background 150ms;
}

.multi-select-search-wrap input[type="text"].multi-select-search-input:focus,
input[type="text"].multi-select-search-input:focus {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 2px var(--focus-ring);
  background: var(--bg-card);
}

.multi-select-clear-btn {
  position: absolute;
  right: 7px;
  top: 50%;
  transform: translateY(-50%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-pill);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: background 120ms, color 120ms;
}

.multi-select-clear-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.multi-select-clear-btn .icon {
  width: 10px;
  height: 10px;
  position: static;
  transform: none;
  color: inherit;
}

.multi-select-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 4px 4px;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 11.5px;
  color: var(--text-secondary);
}

.multi-select-bar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.multi-select-options {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 160px;
  overflow-y: auto;
  padding-right: 2px;
}

.multi-select-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: 12.5px;
  cursor: pointer;
  user-select: none;
  transition: background 120ms, color 120ms;
}

.multi-select-option:hover {
  background: var(--bg-hover);
}

.multi-select-option.is-selected {
  background: var(--accent-blue-soft);
  color: var(--accent-blue-text);
  font-weight: 550;
}

.multi-select-checkbox {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: 1px solid var(--border-color);
  border-radius: 3px;
  background: var(--bg-card);
  flex: 0 0 auto;
  transition: border-color 120ms, background 120ms;
}

.multi-select-option.is-selected .multi-select-checkbox {
  border-color: var(--accent-blue);
  background: var(--accent-blue);
  color: var(--text-inverse);
}

.multi-select-check-icon {
  width: 11px;
  height: 11px;
  display: none;
}

.multi-select-option.is-selected .multi-select-check-icon {
  display: block;
}

.multi-select-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1 1 auto;
}

.multi-select-empty {
  padding: 16px 8px;
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
}

/* --- chips --- */
.chip-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-height: 160px;
  overflow-y: auto;
  padding: 8px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

.chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 26px;
  padding: 0 10px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.2;
  user-select: none;
  cursor: pointer;
  transition: background 120ms, border-color 120ms, color 120ms, box-shadow 120ms;
}

.chip:hover {
  border-color: var(--accent-blue);
  color: var(--text-primary);
  background: var(--bg-hover);
}

.chip.is-selected {
  background: var(--accent-blue-soft);
  border-color: var(--accent-blue);
  color: var(--accent-blue-text);
  font-weight: 600;
}

.chip-check {
  display: inline-block;
  width: 0;
  max-width: 0;
  margin-right: 0;
  font-size: 11px;
  line-height: 1;
  opacity: 0;
  overflow: hidden;
  transform: scale(0.6);
  transition: width 140ms ease, margin-right 140ms ease, opacity 120ms ease, transform 140ms ease;
}

.chip.is-selected .chip-check {
  width: 11px;
  max-width: 11px;
  margin-right: 5px;
  opacity: 1;
  transform: scale(1);
}

/* --- badges --- */
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

/* The remark column renders one row of tags. All four share one shape — same
   height, padding, font and full-round corners — so the row reads as a set
   instead of four look-alikes with drifting metrics. */
.tag-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.badge-remark,
.badge-quota,
.badge-reset,
.badge-models {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 26px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: var(--radius-pill);
  font-size: 13px;
  font-weight: 650;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.badge-remark {
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  background: var(--accent-blue-subtle);
  color: var(--accent-blue-text);
  border-color: var(--accent-blue-border);
}

.badge-quota {
  color: var(--accent-amber-text);
  background: var(--accent-amber-soft);
  border-color: var(--accent-amber-border);
}

.badge-reset {
  color: var(--accent-green-text);
  background: var(--accent-green-soft);
  border-color: var(--accent-green-border);
}

.badge-models {
  color: var(--accent-violet-text);
  background: var(--accent-violet-soft);
  border-color: var(--accent-violet-border);
}

.badge-quota .icon,
.badge-reset .icon {
  width: 13px;
  height: 13px;
  opacity: 0.9;
}

/* UI fonts reserve descent space below the baseline that short labels never use,
   so centring the em box leaves the glyphs about a pixel low. Lift the text only:
   the glyph icons are drawn centred in their own view box. */
.badge-remark > span,
.badge-quota > span,
.badge-reset > span,
.badge-models > span {
  transform: translateY(-0.5px);
}

.badge-quota.is-exceeded {
  background: var(--accent-red-soft);
  border-color: var(--accent-red-border);
  color: var(--accent-red-text);
}

.badge-clickable {
  cursor: pointer;
  transition: transform 0.12s ease, box-shadow 0.12s ease;
}

.badge-clickable:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

/* --- key/value list --- */
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

/* --- tables --- */
.table-wrap {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
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

.data-table tbody tr:not(.detail-row):hover td { background-color: var(--bg-hover) !important; }

.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* The remark column is the table's flexible column: every other column hugs its
   content, so the leftover width lands on the remark instead of empty gutters. */
.data-table th.col-remark { width: 100%; }

.data-table th:not(.col-remark) { width: 1%; }

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
  max-width: none;
  word-break: break-word;
}

.empty-state {
  padding: 44px 20px;
  color: var(--text-tertiary);
  font-size: 13.5px;
  text-align: center;
}

.placeholder-note {
  padding: 32px 16px;
  color: var(--text-tertiary);
  font-size: 13px;
  text-align: center;
}

.placeholder-note.is-error { color: var(--accent-red-text); }

.placeholder-note.is-compact { padding: 24px 16px; }

.table-loading {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--bg-card);
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1), visibility 0.2s ease;
}

.spinner {
  width: 30px;
  height: 30px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.75s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.table-loading-text {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 550;
  letter-spacing: 0.02em;
}

.copy-shuttle {
  position: fixed;
  top: -9999px;
  left: -9999px;
  opacity: 0;
}

/* --- overlays --- */
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

.modal-header-row {
  align-items: baseline;
  margin-bottom: 12px;
}

.modal-header-row h2 { margin: 0; }

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
  font-weight: 650;
  opacity: 0;
  transform: translateY(10px) scale(0.98);
  transition: opacity 200ms var(--ease-emphasized), transform 200ms var(--ease-emphasized);
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
`
