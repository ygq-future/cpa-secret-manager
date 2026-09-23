package management

// templateStyleTokens declares the CPA host theme variables used by the page.
//
// Names and fallback values mirror the official management center
// (src/styles/themes.scss) so the page looks correct standalone and is
// overridden at runtime by the host's live computed values, see ADR-0003.
// Every variable declared in :root must also be declared in both theme blocks;
// TestPageThemeTokensAreComplete enforces it.
const templateStyleTokens = `
*, *::before, *::after { box-sizing: border-box; }

:root {
  color-scheme: light;
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
  color-scheme: light;
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
  color-scheme: dark;
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
`
