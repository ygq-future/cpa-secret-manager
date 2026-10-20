package management

// ThemePaletteCSS declares every CSS custom property the page consumes.
//
// The architecture mirrors the verified sibling plugin `antigravity-priority`:
//
//  1. the page owns a design vocabulary (--bg-card, --bg-subtle, --bg-overlay,
//     --accent-*, --shadow-*, radii, monospace stack) that never changes with the
//     host, so buttons, badges and focus states look identical everywhere;
//  2. the host only supplies surfaces, text and borders. Those keep the host's
//     own variable names so the runtime can copy the host's live values onto the
//     page root (ADR-0003): --bg-primary is the canvas, --bg-secondary the card
//     surface and --bg-tertiary the inset surface, exactly how the host stacks
//     them;
//  3. the runtime then re-points --bg-card/--bg-subtle at the host's values.
//
// The three static palettes below are the fallback set (standalone page and the
// local host simulator), taken from the same verified palette. Every property
// declared in :root must also be declared in both theme blocks;
// TestPageThemeTokensAreComplete enforces it, and TestPage_ReferencedVariables-
// AreDefined keeps the stylesheet free of undefined references.
const ThemePaletteCSS = `
*, *::before, *::after { box-sizing: border-box; }

:root {
  color-scheme: light dark;
  --bg-primary: #f8fafc;
  --bg-secondary: #f1f5f9;
  --bg-tertiary: #e2e8f0;
  --bg-surface: #ffffff;
  --bg-inset: #eef2f7;
  --bg-card: #ffffff;
  --bg-subtle: #f1f5f9;
  --bg-hover: rgba(15, 23, 42, 0.04);
  --bg-overlay: rgba(15, 23, 42, 0.45);
  --border-color: #e2e8f0;
  --border-subtle: #f1f5f9;
  --border-focus: #3b82f6;
  --text-primary: #0f172a;
  --text-secondary: #475569;
  --text-tertiary: #94a3b8;
  --text-muted: #94a3b8;
  --text-inverse: #ffffff;
  --accent-blue: #3b82f6;
  --accent-blue-hover: #2563eb;
  --accent-blue-subtle: #eff6ff;
  --accent-blue-text: #1d4ed8;
  --accent-green: #10b981;
  --accent-red: #ef4444;
  --accent-red-subtle: #fef2f2;
  --accent-red-text: #b91c1c;
  --accent-blue-soft: rgba(59, 130, 246, 0.10);
  --accent-blue-border: rgba(59, 130, 246, 0.22);
  --accent-red-soft: rgba(239, 68, 68, 0.10);
  --accent-red-border: rgba(239, 68, 68, 0.28);
  --accent-amber-soft: rgba(245, 158, 11, 0.12);
  --accent-amber-border: rgba(245, 158, 11, 0.30);
  --accent-amber-text: #b45309;
  --accent-green-soft: rgba(16, 185, 129, 0.12);
  --accent-green-border: rgba(16, 185, 129, 0.28);
  --accent-green-text: #047857;
  --accent-violet-soft: rgba(139, 92, 246, 0.10);
  --accent-violet-border: rgba(139, 92, 246, 0.25);
  --accent-violet-text: #6d28d9;
  --on-accent: #ffffff;
  --btn-primary-bg: #0f172a;
  --btn-primary-hover: #1e293b;
  --btn-primary-text: #ffffff;
  --focus-ring: rgba(59, 130, 246, 0.15);
  --switch-knob: #ffffff;
  --switch-track: #cbd5e1;
  --switch-track-border: #64748b;
  --shadow-sm: 0 1px 3px 0 rgba(0, 0, 0, 0.04), 0 1px 2px -1px rgba(0, 0, 0, 0.03);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -2px rgba(0, 0, 0, 0.03);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.06), 0 4px 6px -4px rgba(0, 0, 0, 0.03);
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-pill: 999px;
  --ease-emphasized: cubic-bezier(0.23, 1, 0.32, 1);
  --font-mono: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace;
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme='light']):not([data-theme='white']) {
    --bg-primary: #121214;
    --bg-secondary: #18181b;
    --bg-tertiary: #27272a;
    --bg-surface: #18181b;
    --bg-inset: var(--bg-primary);
    --bg-card: #1f1f23;
    --bg-subtle: #27272a;
    --bg-hover: rgba(255, 255, 255, 0.055);
    --bg-overlay: rgba(0, 0, 0, 0.7);
    --border-color: #27272a;
    --border-subtle: #27272a;
    --border-focus: #a1a1aa;
    --text-primary: #f4f4f5;
    --text-secondary: #a1a1aa;
    --text-tertiary: #71717a;
    --text-muted: #71717a;
    --text-inverse: #0f172a;
    --accent-blue: #3b82f6;
    --accent-blue-hover: #60a5fa;
    --accent-blue-subtle: #1e293b;
    --accent-blue-text: #60a5fa;
    --accent-green: #22c55e;
    --accent-red: #ef4444;
    --accent-red-subtle: rgba(127, 29, 29, 0.25);
    --accent-red-text: #f87171;
    --accent-blue-soft: rgba(59, 130, 246, 0.18);
    --accent-blue-border: rgba(96, 165, 250, 0.35);
    --accent-red-soft: rgba(239, 68, 68, 0.16);
    --accent-red-border: rgba(248, 113, 113, 0.35);
    --accent-amber-soft: rgba(245, 158, 11, 0.16);
    --accent-amber-border: rgba(245, 158, 11, 0.35);
    --accent-amber-text: #fbbf24;
    --accent-green-soft: rgba(34, 197, 94, 0.16);
    --accent-green-border: rgba(34, 197, 94, 0.35);
    --accent-green-text: #4ade80;
    --accent-violet-soft: rgba(167, 139, 250, 0.16);
    --accent-violet-border: rgba(167, 139, 250, 0.35);
    --accent-violet-text: #c4b5fd;
    --on-accent: #ffffff;
    --btn-primary-bg: #f4f4f5;
    --btn-primary-hover: #e4e4e7;
    --btn-primary-text: #09090b;
    --focus-ring: rgba(59, 130, 246, 0.25);
    --switch-knob: #ffffff;
    --switch-track: #52525b;
    --switch-track-border: #71717a;
    --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.4);
    --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -2px rgba(0, 0, 0, 0.4);
    --shadow-lg: 0 10px 18px -3px rgba(0, 0, 0, 0.5), 0 4px 6px -4px rgba(0, 0, 0, 0.5);
  }
}

[data-theme='white'] {
  color-scheme: light;
  --bg-primary: #ffffff;
  --bg-secondary: #ffffff;
  --bg-tertiary: #f4f4f5;
  --bg-surface: #ffffff;
  --bg-inset: #f4f4f5;
  --bg-card: #ffffff;
  --bg-subtle: #f4f4f5;
  --bg-hover: rgba(15, 23, 42, 0.04);
  --bg-overlay: rgba(15, 23, 42, 0.45);
  --border-color: #e4e4e7;
  --border-subtle: #f4f4f5;
  --border-focus: #3b82f6;
  --text-primary: #0f172a;
  --text-secondary: #475569;
  --text-tertiary: #94a3b8;
  --text-muted: #94a3b8;
  --text-inverse: #ffffff;
  --accent-blue: #3b82f6;
  --accent-blue-hover: #2563eb;
  --accent-blue-subtle: #eff6ff;
  --accent-blue-text: #1d4ed8;
  --accent-green: #10b981;
  --accent-red: #ef4444;
  --accent-red-subtle: #fef2f2;
  --accent-red-text: #b91c1c;
  --accent-blue-soft: rgba(59, 130, 246, 0.10);
  --accent-blue-border: rgba(59, 130, 246, 0.22);
  --accent-red-soft: rgba(239, 68, 68, 0.10);
  --accent-red-border: rgba(239, 68, 68, 0.28);
  --accent-amber-soft: rgba(245, 158, 11, 0.12);
  --accent-amber-border: rgba(245, 158, 11, 0.30);
  --accent-amber-text: #b45309;
  --accent-green-soft: rgba(16, 185, 129, 0.12);
  --accent-green-border: rgba(16, 185, 129, 0.28);
  --accent-green-text: #047857;
  --accent-violet-soft: rgba(139, 92, 246, 0.10);
  --accent-violet-border: rgba(139, 92, 246, 0.25);
  --accent-violet-text: #6d28d9;
  --on-accent: #ffffff;
  --btn-primary-bg: #0f172a;
  --btn-primary-hover: #1e293b;
  --btn-primary-text: #ffffff;
  --focus-ring: rgba(59, 130, 246, 0.15);
  --switch-knob: #ffffff;
  --switch-track: #cbd5e1;
  --switch-track-border: #64748b;
  --shadow-sm: 0 1px 3px 0 rgba(0, 0, 0, 0.04), 0 1px 2px -1px rgba(0, 0, 0, 0.03);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -2px rgba(0, 0, 0, 0.03);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.06), 0 4px 6px -4px rgba(0, 0, 0, 0.03);
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-pill: 999px;
  --ease-emphasized: cubic-bezier(0.23, 1, 0.32, 1);
  --font-mono: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace;
}

[data-theme='dark'] {
  color-scheme: dark;
  --bg-primary: #121214;
  --bg-secondary: #18181b;
  --bg-tertiary: #27272a;
  --bg-surface: #18181b;
  --bg-inset: var(--bg-primary);
  --bg-card: #1f1f23;
  --bg-subtle: #27272a;
  --bg-hover: rgba(255, 255, 255, 0.055);
  --bg-overlay: rgba(0, 0, 0, 0.7);
  --border-color: #27272a;
  --border-subtle: #27272a;
  --border-focus: #a1a1aa;
  --text-primary: #f4f4f5;
  --text-secondary: #a1a1aa;
  --text-tertiary: #71717a;
  --text-muted: #71717a;
  --text-inverse: #0f172a;
  --accent-blue: #3b82f6;
  --accent-blue-hover: #60a5fa;
  --accent-blue-subtle: #1e293b;
  --accent-blue-text: #60a5fa;
  --accent-green: #22c55e;
  --accent-red: #ef4444;
  --accent-red-subtle: rgba(127, 29, 29, 0.25);
  --accent-red-text: #f87171;
  --accent-blue-soft: rgba(59, 130, 246, 0.18);
  --accent-blue-border: rgba(96, 165, 250, 0.35);
  --accent-red-soft: rgba(239, 68, 68, 0.16);
  --accent-red-border: rgba(248, 113, 113, 0.35);
  --accent-amber-soft: rgba(245, 158, 11, 0.16);
  --accent-amber-border: rgba(245, 158, 11, 0.35);
  --accent-amber-text: #fbbf24;
  --accent-green-soft: rgba(34, 197, 94, 0.16);
  --accent-green-border: rgba(34, 197, 94, 0.35);
  --accent-green-text: #4ade80;
  --accent-violet-soft: rgba(167, 139, 250, 0.16);
  --accent-violet-border: rgba(167, 139, 250, 0.35);
  --accent-violet-text: #c4b5fd;
  --on-accent: #ffffff;
  --btn-primary-bg: #f4f4f5;
  --btn-primary-hover: #e4e4e7;
  --btn-primary-text: #09090b;
  --focus-ring: rgba(59, 130, 246, 0.25);
  --switch-knob: #ffffff;
  --switch-track: #52525b;
  --switch-track-border: #71717a;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.4);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -2px rgba(0, 0, 0, 0.4);
  --shadow-lg: 0 10px 18px -3px rgba(0, 0, 0, 0.5), 0 4px 6px -4px rgba(0, 0, 0, 0.5);
  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-pill: 999px;
  --ease-emphasized: cubic-bezier(0.23, 1, 0.32, 1);
  --font-mono: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace;
}
`
