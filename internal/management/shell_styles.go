package management

// This file owns the page layout: app chrome, panels and the responsive
// contract, plus the styles of the page-level overlays (failure records and the
// date range dialog) declared by shellModals.
//
// Everything reusable lives in components.go; a feature only declares styles for
// markup that exists nowhere else.

// templateStyleLayout holds the fixed-height page shell: header, tab bar, panel
// scroll areas and the status strip. Only the table body (or the settings/help
// scroller) scrolls, which keeps the page free of a document-level scrollbar
// inside the plugin iframe.
const templateStyleLayout = `
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
  border-radius: var(--radius-pill);
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

.scroll-area .card { flex: 0 0 auto; }

.panel > .card { flex: 1 1 auto; }

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

.count-pill {
  color: var(--text-tertiary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

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
`

// templateStyleOverlays styles the two page-level dialogs declared by
// shellModals: the failure records viewer and the date range picker.
const templateStyleOverlays = `
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
  box-shadow: var(--shadow-lg);
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

.date-dialog-heading {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.date-dialog-title {
  font-size: 17px;
  font-weight: 650;
  color: var(--text-primary);
}

.date-selection-status {
  font-size: 13px;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
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
  font-weight: 650;
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
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-primary);
  user-select: none;
  transition: background 120ms ease;
}

.calendar-day:hover:not(.empty) { background: var(--bg-hover); }

.calendar-day.empty {
  cursor: default;
  opacity: 0;
}

/* Days before the retention window hold no data at all, so they are shown but
   cannot be picked. */
.calendar-day.is-disabled {
  color: var(--text-muted);
  opacity: 0.4;
  cursor: not-allowed;
  text-decoration: line-through;
}

.calendar-day.is-disabled:hover { background: transparent; }

.calendar-day.range-start,
.calendar-day.range-end {
  background: var(--accent-blue) !important;
  color: var(--on-accent) !important;
  font-weight: 650;
}

.calendar-day.in-range {
  background: var(--accent-blue-soft);
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
  font-weight: 650;
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

.failures-filter-bar .select {
  flex: 1 1 0;
  min-width: 0;
  max-width: 280px;
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
  transition: background 120ms ease;
}

.failure-item:hover { background: var(--bg-hover); }

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
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.failure-status-badge {
  font-size: 11px;
  font-weight: 650;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
}

.failure-status-badge.is-error {
  background: var(--accent-red-soft);
  border-color: var(--accent-red-border);
  color: var(--accent-red-text);
}

.failure-status-badge.is-warning {
  background: var(--accent-amber-soft);
  border-color: var(--accent-amber-border);
  color: var(--accent-amber-text);
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

// templateStyleResponsive adapts the page to narrow viewports and to reduced
// motion. It only carries media queries: every component rule lives with its
// component.
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
}

@media (max-width: 640px) {
  .calendar-panels { grid-template-columns: 1fr; }
  .time-pickers-row { grid-template-columns: 1fr; }
}

@media (prefers-reduced-motion: reduce) {
  .btn, .btn-icon, .pill, .pill-btn, .chip, .tab, .search-clear, .select-trigger,
  .select-arrow, .select-option, .badge-clickable, .data-table th, .data-table td,
  .toggle-slider, .toggle-slider::before, .toast, .modal, .modal-backdrop.open,
  .table-loading, .calendar-day, .failure-item {
    animation: none;
    transition: none;
  }
  .spinner { animation: none; }
}
`
