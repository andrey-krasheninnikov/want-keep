# Task-3.3 — isolated browser collector

Implementation base: `a4a77365b909080b48093bc5304216489526642c`; branch: `feat/task-3.3-isolated-browser-collector`. Dependencies task-1.5 and task-3.2 are included. Live platforms, user browser profiles and production were not changed.

## Proven result

The collector runs as a separate Node.js process with Playwright 1.63.0 and listens only on a mode-`0600` Unix socket. Its protocol contains `GET /ready`, `POST /v1/capabilities` and `POST /v1/read`. The server bounds request size and duration and runs one browser job at a time. Every job creates a new non-persistent `BrowserContext`; JavaScript, downloads and service workers are disabled and the context closes after every outcome.

Build-owned runtime configuration defines the complete D-43 binding, `admissionRevision`, capability manifest, exact origin and allowed actions. The collector matches every binding field and revision before browser launch. Its allowlist permits one `entry`, one `read` and an optional `request_statement` with exact `from`/`to` values. A subsequent read sends the server-issued cursor as the sole `cursor` query parameter; unknown origins, methods, paths, queries and payloads are blocked. The read is performed outside the page with filtered cookies and a bounded response stream; page scripts cannot forge it or send a second statement POST. Redirects, popups, downloads, WebSockets, service workers and the synthetic portal's payment route cannot cross the boundary. The input contract contains no URL, selector, JavaScript, upload or route rule. MFA, CAPTCHA and an expired session become typed provider failures without returning session state.

The Go Unix-socket client implements the existing `contract.RawGateway`. The worker constructs it only from a stored sync job and current admission, temporarily borrows `browser_session` through `credentials.Vault` and clears its copy after the job. The `external_started` marker is set immediately before `/v1/read`; a capability check does not set it. Confirmed `collector_busy`, `collector_session_invalid` and `collector_preflight_rejected` responses prove rejection before provider IO: the marker is cleared under the current lease, leaving the job waiting for the collector, owner sign-in or a failed request, respectively. HTTP 429 becomes a typed `rate_limited` outcome with bounded retry. Invalid 401/403 body bytes do not mask a sign-in request. A lost connection after provider IO starts produces `unresolved` without automatic provider replay. Pages and failures pass the existing admission, connection-generation, lease and cursor fences. A stale result is retained only in quarantine.

Raw evidence is encrypted through the existing connection keyring before PostgreSQL storage. AAD binds ciphertext to household, job, page and evidence reference. Migration 021 stores batch metadata and immutable evidence items; provider evidence IDs stay encrypted. A batch can move only from `staged` to one terminal disposition; the application has minimal grants and cannot read plaintext. Staged recovery after restart uses the existing terminal receipt without repeating browser IO or a financial effect.

## Verification matrix

Required commands are `make check`; `make test-collector FILTER=security`; `make test-integration AREA=collector`; `make test-integration AREA=all`; ingestion/jobs/storage/privacy integration and affected race suites; and `git diff --check`. The PostgreSQL suite uses isolated PostgreSQL 17.11 under the unprivileged role and fails without `WANT_KEEP_TEST_DATABASE_URL`. The browser suite installs the pinned Chromium and accesses only a local synthetic portal. Exact candidate and CI results are recorded in the PR.

Tests cover safe GET and statement POST, separate session cookies, MFA/CAPTCHA, HTTP 429, invalid 401 body bytes, Chromium restart, route normalization, page fetch replacement, duplicate statement prevention, WebRTC suppression, payment/redirect/popup/download/WebSocket/service-worker blocking and stale binding before browser IO. Go tests cover Unix-only transport, the `external_started` boundary, three proven pre-IO rejections, absence of session data in results and encryption/AAD. PostgreSQL tests cover ciphertext-only storage, absence of provider IDs from metadata, nanosecond time, household isolation, restart, idempotent terminal disposition, immutable items and staged recovery.

## Acceptance boundaries

| Criteria | Proven by task-3.3 | Downstream verification |
| --- | --- | --- |
| AC-040/041/048 | Read-only runtime, exact statement POST, typed MFA/CAPTCHA/reauthentication and one job | Live provider routes, history and owner reauthentication |
| AC-050/061/087 | Temporary session through the vault, context isolation and no plaintext in result/database/diagnostics | User session-supply flow and browser UI |
| AC-079/090/106 | Server-owned binding/principal, exact pre-read fence, encrypted evidence and commit-time quarantine | Provider/deployment evidence and production admission |
| AC-060/068 | Shared safe collector infrastructure and bounded synthetic egress | Complete live-source and operational scenarios |

The SDD remains **Ready for development**. Live portals needing page JavaScript require a separate vetted workflow. Personal Chrome/Arc profiles, production egress and provider deployment are not proven by this task.

After merging task-6.1, its `020_rates_valuation.sql` is retained; the unapplied collector migration is renamed to `021_collector_evidence.sql` without SQL changes. No production schema was changed.
