# want-desyst migration: task-7.16 / #97

[Русский](issue-97-want-desyst.md) · [Issue](https://github.com/andrey-krasheninnikov/want-keep/issues/97)

D-49 replaces former local visual decisions with the complete want-desyst dark theme. Foundation/react 0.2.1 are published on npm and pinned with integrity in the lockfile. Upstream [PR #9](https://github.com/andrey-krasheninnikov/want-desyst/pull/9) fixes years 0001–0099 and calendar bounds; tag v0.2.1 points to the merged main result.

Migration covers the existing shell, sign-in, bootstrap, invitation, recovery, security, household panel, cash accounts and both development catalogs. [design-components.en.md](../design-components.en.md) maps ownership and public interfaces. Base UI, DayPicker and shadcn are removed, together with former palette/focus/font-face overrides and Pixelify Sans. Local compositions use shared controls while preserving product input, labeling and financial states.

Local checks: 108 web tests, TypeScript, lint, Vite build with production CSS/font/development-catalog checks, and docs-check (62 tests). Automated Chromium covers all 26 design scenarios, including contrast, money input, RU/EN, keyboard, overlays, 17 states, reduced motion, font failure, years 0001/9999 and three timezones. Successful scenarios were reused; corrected cases ran separately. The access scenario with temporary PostgreSQL confirms closed household setup, two independent passkeys, a cash command, invitation and personal recovery. Review and publication outcomes are recorded on GitHub.

Vite reports a main JS chunk above 500 kB (546.51 kB, 170.31 kB gzip); this is a warning, not a build failure. The shared theme applies without copied primitives. Invalid fields explicitly use the shared foreground for readable input while their label and border retain status feedback.

The owner confirmed manual scenarios in macOS Chrome and Arc: RU/English, Tab and visible focus, a form with 29.02.2024 and BTC, Escape/focus restoration, notification, a smaller window and 200% zoom, and the sign-in screen. This is owner-reported evidence; automated Chromium is tracked separately. The available Chrome-control tool timed out, so direct manual verification through that tool is not claimed.

Exact monetary strings, calendar dates, APIs, authority, CSRF and command-outcome recovery remain with their existing owners. Backend/database schema are unchanged. Full backend/banking suites, live bank portals, Touch ID, push delivery and production were not checked. New screens and event triggers in #62 remain separate work. SDD Ready for development does not establish operational readiness.
