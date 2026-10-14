# API-key usage

Mode: Operate. Build: code-first, confirmed by the user.

Inherit the application's existing type, themes, controls, and chart language; redesign only the user surface. First viewport: a clear usage title and masked key, compact Overview / Models & access navigation, quota windows and concurrency, then spending and usage measurements. Trends follow; prices and model permissions move to the secondary view. Use a flat, aligned measurement strip rather than repeated floating cards. Admin surfaces retain their existing structure.

Signature interaction: change the shared time range to update spending, trend, and model breakdown together. Preserve existing chart and quota-fill motion and reduced-motion behavior. Login names API-key access directly, with no role switch on the user page. Desktop provides a bounded reading width and asymmetric quota/concurrency row; narrow screens stack the same information without horizontal page scrolling. Verify login, role boundaries, refresh, empty/error states, both themes, and desktop/narrow layouts.

## Checked implementation evidence

Compared `internal/plugin/web/ui.html`, the usage-page rules in `style.css`, and the summary renderer in `analysis.js` with the six supplied desktop/narrow screenshots: Overview in light and dark themes, and Models & access in light theme. The surface uses existing theme variables, sans-serif headings, monospace measurements, controls, and chart colors. Quota and concurrency precede the flat spending strip; model permissions and prices occupy the secondary tab. The desktop quota row uses a 3:1 grid inside a 1180px maximum width; at 768px and below it stacks, with measurements in two columns. The screenshots preserve this order and show contained charts and wrapping access chips.

Reviewer disposition: ship; no material visual fixes. These are surface-specific composition choices, not new application tokens or a replacement visual system. Screenshot comparison does not establish authentication, API role enforcement, refresh, or empty/error behavior; those require separate functional verification.

Credential access is admin-only. The user routing response exposes model permissions only and never loads or returns upstream credential inventory.
