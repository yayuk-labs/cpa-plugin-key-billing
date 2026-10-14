# API-key usage

Mode: Operate. Build: code-first, confirmed by the user.

THESIS: Overview answers how much quota remains and how much has been used in each current quota window. Historical spend is a separate task.

OWN-WORLD: Preserve the application's existing light/dark theme variables, sans-serif typography, controls, icons, and charts. This is a scoped layout correction inside the incumbent visual world.

STORY: Overview shows independent quota windows. Usage history retains period-filtered totals, trends, and model breakdown. Models & access retains prices and model permissions. Credential access, diagnostics, and Source breakdown remain admin-only; user APIs do not return these identities.

FIRST VIEWPORT: A full-width quota panel shows plan/status, each window's name, prominent remaining balances, explicit used/limit text, remaining bars, reset state, and visible model scope. Concurrency is a compact footer row. No historical totals or date-range selector compete with current quota.

FORM: Thin separators group real quota windows without nested cards. Every monetary/token/request balance remains independent; never sum overlapping windows or infer values from history. Read remaining, used/limit, then scope; reset stays next to the window name. Narrow layouts stack balances and wrap metadata/scopes in the same DOM order. Use the existing compact navigation select when tabs do not fit. This precisely specified inherited-world refinement requires no concept roll or comp.

Verification: Test the user's daily $5 remaining and monthly $9.53 remaining/$0.47 used example with synthetic data, multi-dimension windows, exhausted/unlimited states, English/Chinese, desktop/mobile, and retained admin layouts. Quota Overview loads subscription state only; history loads analysis on entry. Source preferences in user history fall back to Model.

## Checked implementation evidence

Read `account.js` (`renderAccountQuotaWindows`), the quota/history sections in `ui.html`, and the quota rules in `style.css`; compared `quota-desktop.png`, `quota-mobile.png`, `quota-desktop-dark.png`, and `quota-mobile-dark.png` under `/tmp/cpa-usage-preview/`. All four show independent daily/monthly windows, explicit remaining and used/limit labels, visible model scopes, reset state, thin separators, and compact concurrency below the balances. The sample reads $5 remaining/Used $0 of $5 and $9.53 remaining/Used $0.47 of $10. Narrow captures stack metadata and balances, wrap scopes, and use the existing navigation select. History has its own tab and panel; no historical totals or time selector appear on Overview.

Reviewer disposition: ship; no material visual fixes. The coordinating agent reports passing browser checks for history navigation, no history API load on Overview, user-hidden/admin-retained Source controls, and multi-limit English/Chinese, exhausted, and unlimited states at 1440px, 390px, and 320px; tests, Go vet/race, and billing end-to-end verification against v7.2.143 also passed. Deployment remains blocked by missing `.env`. This documentation pass independently checked source and the four required captures only. The correction inherits existing themes and controls; it establishes no new application-wide tokens or visual system.
