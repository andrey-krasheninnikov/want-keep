# want-desyst migration: task-7.16 / #97

[Русский](issue-97-want-desyst.md) · [Issue](https://github.com/andrey-krasheninnikov/want-keep/issues/97)

D-49 replaces former local visual decisions with the complete want-desyst dark theme. Foundation/react 0.2.1 are published on npm and pinned with integrity in the lockfile. Upstream [PR #9](https://github.com/andrey-krasheninnikov/want-desyst/pull/9) fixes years 0001–0099 and calendar bounds; tag v0.2.1 points to the merged main result.

Migration covers the existing shell, sign-in, bootstrap, invitation, recovery, security, household panel, cash accounts and both development catalogs. [design-components.en.md](../design-components.en.md) maps ownership and public interfaces. Base UI, DayPicker and shadcn are removed, together with former palette/focus/font-face overrides and Pixelify Sans. Local compositions use shared controls while preserving product input, labeling and financial states.

Local checks: 108 web tests, TypeScript, lint, Vite build with production CSS/font/development-catalog checks, and docs-check (62 tests). Automated Chromium covers all 28 design scenarios, including contrast, money input, RU/EN, keyboard, overlays, 17 states, reduced motion, font failure, years 0001/9999 and three timezones. Successful scenarios were reused; corrected cases ran separately. The access scenario with temporary PostgreSQL confirms closed household setup, two independent passkeys, a cash command, invitation and personal recovery. Review and publication outcomes are recorded on GitHub.

Vite reports a main JS chunk above 500 kB (546.72 kB, 170.36 kB gzip); this is a warning, not a build failure. The shared theme applies without copied primitives. Invalid fields explicitly use the shared foreground for readable input while their label and border retain status feedback.

The owner confirmed manual scenarios in macOS Chrome and Arc: RU/English, Tab and visible focus, a form with 29.02.2024 and BTC, Escape/focus restoration, notification, a smaller window and 200% zoom, the sign-in screen, reduced motion and reduced transparency. Locally installed versions: macOS 26.6.2 (25G83), Chrome 156.0.8078.48 and Arc 1.170.0. This is owner-reported evidence; automated Chromium is tracked separately. The available Chrome-control tool timed out, so direct manual verification through that tool is not claimed.

Review found two adaptation defects: clearing an invalid date draft and restoring focus from a triggerless leave warning. An invalid draft stays distinct from empty and canonical picker values, including pasted ISO-shaped text; LeaveGuard captures and restores the previous focus. Checks cover Clear and day selection after ISO-shaped invalid input in RU/EN, Stay and Escape.

The access/design-system web jobs use Ubuntu 22.04, matching the collector runner with a verified Chromium sandbox. On ubuntu-latest, startup failed with No usable sandbox. Sandboxing remains enabled; current CI outcomes are recorded on the PR.

Exact monetary strings, calendar dates, APIs, authority, CSRF and command-outcome recovery remain with their existing owners. Backend/database schema are unchanged. Full backend/banking suites were not repeated locally. Live bank portals, Touch ID, push delivery and production were not checked. New screens and event triggers in #62 remain separate work. SDD Ready for development does not establish operational readiness.
