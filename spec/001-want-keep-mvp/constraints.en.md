# Architecture constraints

[Русский](constraints.md)

The user selected the stack (D-17). task-1.1 provides reproducible manifests and a buildable foundation without a product runtime/API. The remainder of this document describes the target architecture.

Go 1.26.5 and Node.js 24.19.0 LTS are pinned. Web uses React 19.2.8, Vite 8.2.2, TypeScript 6.0.3, Base UI 1.8.0 and shadcn 4.21.0; the collector uses Playwright 1.63.0. Direct npm dependencies use exact versions and each package has its own lockfile.

## Behavior ownership and layout

A modular Go monolith with API/worker processes, managed PostgreSQL, React/TypeScript/Vite and a separate Playwright TypeScript collector. Production uses one German 2 vCPU/4 GB/50 GB application VPS and 1 vCPU/2 GB/20 GB managed PostgreSQL in the same private VPC without a public IP; Docker Compose owns the application, not the production DB. Development and integration use an isolated PostgreSQL container. Redis, Kafka, Kubernetes, a vector database and a Python service are not mandatory. Configuration and open runtime gates: [task-0.9 research](evidence/hosting.en.md).

| Planned area | Responsibility |
| --- | --- |
| `backend/internal/<feature>/` | Money/accounts/ledger/budget/goals/forecast/credit/savings/returns/trading/mining and application contracts. Organize by owner, not global helpers/utils. |
| `backend/internal/delivery/`, `api/openapi.yaml` | HTTP, auth boundary, schema validation and transport mapping. OpenAPI is source; do not hand-edit generated bindings. |
| `backend/internal/integrations/<provider>/` | Verified provider contract, DTOs, gateway/mapper, replay and error classification. No UI imports. |
| `backend/internal/storage/`, `backend/migrations/` | Transactions, persistence mapping and migrations; no business decisions hidden in SQL. |
| `backend/internal/ai/`, `gateways/openai/` | Application commands and bounded orchestration / external OpenAI SDK. Domain does not depend on the SDK. |
| `web/src/features/<feature>/` | Local screens/forms/state; cross-feature access through public APIs and narrow query subscriptions. |
| `collector/src/providers/`, `collector/contracts/` | Isolated portal-read workflows and collection transport contract. No classification or accounting ownership. |
| `deploy/`, `ops/` | Deployment, backup and recovery; keys/data stay outside Git. |

Money is a small independent domain contract. Add other shared modules only with proven shared ownership. Coordinate modules in application services through public contracts, never through another feature's private/data/presentation implementation.

## Flow and persistence

Source/chat → retained evidence → mapping and domain validation → atomic transactions/postings → revision-specific AI review → validated command or clarification → derived reports.

Original source records are immutable; provider changes create revisions. A transaction links multiple pieces of evidence. Financial posting and outbox/job share a transaction. Source IDs are unique within household + stable external-account identity + product/log namespace. Connection/session is provenance, not permanent account identity. Exact idempotency does not replace economic cross-source matching.

Persist exact decimals and transport strings; retain source precision. Separate native ledger and historical rate snapshots from formatting. One-sided/partial transfers, unsupported assets, unknown balances and pending events stay explicitly incomplete.

## Security and trust

- Separate users with household membership; one-time operator-controlled first-member bootstrap. Passkeys validate origin/RP/challenge/purpose/expiry. Recovery codes are hashed and single-use; recovery revokes only the recovering user’s old sessions/subscriptions.
- Browser sessions use Secure HttpOnly SameSite cookies; mutating HTTP requests have CSRF protection. Actor comes from the authenticated server context; ownership is a stored resource property and membership is checked independently of client JSON.
- Encrypt provider keys and bank sessions separately from data; keep master/recovery keys outside DB/logs/Git. Authorize every attachment download/preview.
- Isolate the collector by process, profiles, network and verified action/route/payload allowlists. A GET/POST allowlist alone is insufficient: validate action semantics. Block unknown actions and hand MFA/CAPTCHA to the owner. AI does not control the browser.
- Transaction descriptions, files and model outputs are untrusted. AI receives authorized data without secrets and invokes typed application commands only; no SQL, raw HTTP, shell, payment or trading tools.
- Validate files before AI: MIME/signature, size/page limits and safe parsing. Never execute active content or arbitrary links. Financial texts/documents do not enter telemetry.
- Corrections/links carry expectedRevision, evidence and audit. Changes to approved plans/goals require confirmation of the same proposal version.

## Failure and observability

Source and AI work use independent queues. Sync has leases, bounded retries, checkpoints and last-success time. One failed source does not stop others. Recovery replays idempotent read jobs; reconcile ambiguous paid/external effects first. An incomplete source yields `source_partial`; an identity collision yields `source_ambiguous`. A confirmed monetary effect remains posted while an unverified effect does not post. D-43 admission is checked before scheduling a sync job; a stale/missing binding yields `provider_not_admitted` without provider IO.

AI-reviewed does not mean posted. During AI outage confirmed source/manual transactions persist and calculations work; unknown classification is separate. Unknown amount/account/currency stays draft. Stale AI results never apply.

Logs contain codes, correlation IDs, source/job/transaction IDs, durations and error categories. Metrics cover sync freshness/completeness, AI backlog, clarifications, amount/match errors, usage/reserved spend and backup age. No keys, receipts or full messages.

## Command contract

The root Makefile implements the shared verification interface. `make check` runs only existing foundation checks; a future suite command fails until its implementation exists instead of reporting an empty success.

| Command | Required behavior |
| --- | --- |
| `make check` | Check-mode formatting, lint/typecheck, Go/web/collector unit checks, docs and contract-generation checks. |
| `make test-go PKG=./internal/<feature>/...` | Run the package tests from `backend/` with repo-pinned Go. |
| `make test-web FILTER=<feature>` | Run the matching headless web tests. |
| `make test-collector FILTER=<suite>` | Run bounded synthetic collector scenarios without external accounts. |
| `make test-integration AREA=<suite|all>` | Isolated PostgreSQL/services with fault/concurrency scenarios; never production. |
| `make test-contract PROVIDER=<name>` | Check versioned synthetic provider fixtures and normalization. |
| `make check-contracts` | Check OpenAPI/schema and reproducible generated output. |
| `make e2e SCENARIO=<name|all>` | Browser flows using test data; real-device evidence is separate. |
| `make eval-ai SUITE=<name|all>` | Offline fixtures by default; live runs require a key, explicit run cap and usage report. |
| `make check-deploy` | Validate Compose/config/resources/security without provisioning. |
| `make backup-check MODE=synthetic`, `make restore-check MODE=synthetic` | Isolated integrity/recovery checks with measured duration. |

Suite names in task cards form part of the runner contract. A missing suite fails rather than returning successful empty execution.

## Readiness, migrations and rollout

task-1.1 is complete as an independent technical foundation. task-0.10 resolved fundamental D-37–D-43 decisions and published the [Ready plan](plan.en.md). Implementation starts with task-1.2 and follows each task's dependencies. Unknown provider fields are never filled with invented endpoints or values: safe unknown/partial/ambiguous behavior is part of the completed contract.

SDD Ready and operational approval are separate. Every provider deployment is disabled by default. `backend/internal/connections/admission/` is the application owner of the aggregate/repository interface; task-1.3 implements its storage adapter and atomic transitions. task-4.x proves provider evidence and task-8.x proves host/deployment evidence; only the server-owned admission service combines them for the exact D-43 build/contract/allowlist/configuration/permission/environment binding. A binding change or failed/revoked check increments `admissionRevision`, returns it to `pending|blocked`, invalidates unstarted jobs and is rechecked by the collector before provider IO. A started result must pass commit-time revalidation in the source/posting/outbox transaction; a stale result remains in quarantine. A failed gate blocks only that connector deployment or production, not development of the domain and other adapters.

Add schema changes through new migrations; do not rewrite applied migrations. Check compatibility against existing APIs/data; use expand → backfill → switch → contract only where needed. Rollback never discards the ledger, files or owner corrections.

CI, synthetic integration, live-source readback, physical-device push and restore rehearsal are separate evidence. Only all mandatory ACs complete the MVP. Final review is read-only with independent fact-checking; the SDD owner fixes confirmed findings.

## Household scope and permissions

`backend/internal/household/` owns User/Household/Membership and a public membership/permission policy. Budget/goals/ledger/integrations owners apply it within their use cases alongside domain invariants. Current participants have the member role; personal/household scope and personalOwnerId are not replaced by a role. Add no arbitrary-role editor or first/second-partner fields.

All financial objects, files, chat retrieval, jobs, idempotency and matching are scoped to householdId. Principal comes from a session or validated job context; a report person filter cannot change it. Check household scope, permission, expectedRevision and mutation within one transactional boundary. UI permissions do not replace backend checks. A second test household verifies isolation, not a new public-SaaS feature.

Both edit transactions and connections; only the owner edits a personal goal or personal plan portion; any member edits joint resources. Correcting an expense fact grants no right to edit the other member’s plan. Command replay and revision conflicts differ. Disconnect changes connector generation/lease and prevents previous-generation results from applying. Bank-secret entry is isolated from the shared chat and other member.

Financial source identity includes household, provider and a verified real external account/product/log. Recreating a connection never resets deduplication; equal source IDs from distinct external accounts do not merge. When identity is insufficient, retain evidence without blindly creating another financial account.

## Desktop and presentation

Design, screens and navigation are UI contracts supplementing the API: [design](design.en.md), [screens](screens.en.md), [navigation](navigation.en.md). Design system narrowly owns tokens/primitives/motion; features own user tasks. Server returns amounts, explanations and statuses; client never repeats financial formulas. States and animation events do not control the ledger. The foundation pins Base UI/shadcn and dark base tokens; fonts, components and user screens belong to later UI tasks.

## PostgreSQL task-1.3

pgx v5.10.0 stays in storage; domain/application never import the driver or pgx.Tx. Architecture tests recognize connections/admission as an application boundary. Test PostgreSQL 17.11 is digest-pinned; task-8.1 confirms the production major. READ COMMITTED uses admission-before-household lock order; maintenance is separate from application. [Storage contract and execution](evidence/task-1.3-storage.en.md).

## Identity task-1.4

Auth uses domain/application, a WebAuthn adapter, delivery/identity and storage. D-45 and verification boundaries: [contract](contracts.en.md#task-14-sign-in-and-recovery-d-45), [evidence](evidence/task-1.4-identity.en.md).

Task-1.5 (D-46) adds separate keyrings and an isolated processor. Domain/application never import crypto storage, SQL, HTTP or decoders; only the credentials adapter handles secrets. Processor runtime has no network/keys/DB/shared directory. Protected-function failures do not stop identity/accounting. [Contract](contracts.en.md), [verification and operational handoff](evidence/task-1.5-privacy.en.md).

## Household task-1.6

An invitation does not assign principal or provide financial access before atomic enrollment. Identity lock precedes family/invitation lock; financial family scope and joining scope remain separate. Domain policies preserve current ownership, actor, payer and external owner. Migration 005 expands the schema; secret responses are never replayed. [Contract](contracts.en.md#task-16--invitations-and-household-permissions), [evidence](evidence/task-1.6-household.en.md).

### Accounts storage boundary (task-2.1)

The accounts API registers commands before session-bound execution and reuses the identity → household lock order. Reads use repeatable-read snapshots without financial write locks. Migration 007 preserves 001–006 and marks old unproven balances as legacy. Only admitted import transactions may create imported products, source observations or aliases; public commands cannot forge them. Ledger projections and immutable provider observations have separate persistence and semantics. Account ownership never grants ownership of a bank session. See [accounts contract](contracts.en.md#task-21--accounts-and-opening-balances) and [verification boundaries](evidence/task-2.1-accounts.en.md).

## Ingestion task-3.2

`collector/contracts/v10/ingestion.openapi.yaml` is the sole wire-format source for internal contract version 10. `make generate-contracts` updates Go and TypeScript, while `make check-contracts` compares both outputs against temporary generation. Generated DTOs remain in `integrations/contract` and the collector; `integrations/domain` depends on no transport, SQL, HTTP, Playwright or provider SDK.

`integrations/application` coordinates `ProviderGateway`, `EvidenceStore`, the accounts importer, ledger source writer and `connections/admission`. The gateway is created from server-owned configuration with the full immutable D-43 binding; the application compares it with the job binding before manifest/provider IO. Provider IO runs only after the pre-read fence and outside the financial transaction. Raw evidence is durable before `CommitPage`, is immediately bound to the server-derived household/job, and receives a durable disposition even when later validation fails; the callback performs no external IO. An account may be resolved without a balance observation; every balance/posting reference requires an account descriptor in the same self-contained page. Server principals, external owners, internal IDs, revisions and timestamps never come from the payload.

The contract fails closed: read capabilities only, strict JSON with no missing required fields or explicit `null` in non-nullable fields, canonical UTC instants with `Z`, aligned Unicode code-point/byte limits and an exact job/binding/revision/cursor echo for both pages and provider failures. Cursor is checked when a sync result commits, separately from immutable lease identity: a delayed outcome from an earlier page cannot change job state, while checkpoint advancement cannot break heartbeat. NUL and lone surrogates are rejected before identity/storage; composite keys use structural encoding. Gaps are unique and bounded, while an omitted lookback never collapses into explicit zero. The exact admitted build's manifest is bound to the provider and restricts product, log namespace, record kind and required read action before evidence storage. Confirmed provider mappings, including Raiffeisen/Ozon `RUR → RUB`, live in the accounts-owned policy and preserve the raw code; unknown symbols are never mapped heuristically. A card alias permits only a safe description and its own last four ASCII digits. Separate fee postings are supported, but free-form `feeId` cannot cross the v10 boundary without typed correspondence. Server-side account/source ambiguity preflights complete D-39 keys for the page, deduplicates identical facts, retains evidence, makes coverage partial and posts none of a conflicting group; independent records remain applied. Evidence receives a durable `staged` disposition before the financial commit. The financial effect and immutable receipt from migration 018 commit atomically; bounded readback completes a proven commit, while an unproven outcome remains `staged`. A separate lifecycle context closes only proven rejected pages or provider outcomes. A typed provider outcome atomically associates evidence with its household/job and preserves retry/reauthentication/terminal lifecycle under the sync-result fence. A stale or rejected result crosses only the evidence/quarantine boundary. Downstream tasks prove live provider routes, network isolation and permission; a synthetic gateway does not admit production deployment.

## Browser collector task-3.3

The collector is a separate Node.js process with no TCP listener. Go communicates with it only through a Unix socket. Runtime uses a build-owned exact binding/origin/action allowlist and a new non-persistent Playwright context for every job. JavaScript is disabled in that context; the authenticated read uses its filtered cookies from outside the page, with an exact target and no redirects. User Chrome/Arc profiles, URLs, selectors, JavaScript and route rules are absent from the input contract. Only reads and an approved statement-request POST are allowed; every other action and unknown network transition is closed. A live provider that requires page JavaScript needs a separately reviewed provider workflow and deployment admission.

The existing vault decrypts a session only for the job duration. `external_started` is recorded immediately before IO, and an unknown outcome is never replayed automatically. Evidence is encrypted through the existing connection keyring before PostgreSQL storage; provider evidence IDs remain inside ciphertext and AAD includes household/job/page/item. Migration 022 permits only a one-way disposition transition, gives the application no plaintext access and retains staged recovery. Provider-specific allowlists, egress and task-4.x/task-8.x permission remain mandatory before operation.

## OpenAI task-5.1

`ai/domain` owns exact cost, usage and states; `ai/application` owns reservation, execution and reconciliation; `gateways/openai` is the sole SDK import location; storage hides SQL. The worker composes these boundaries but never applies an AI proposal. Architecture checks reject SDK, HTTP and pgx imports in domain/application.

Migration 016 adds append-only attempts and state history only. The household lock serializes the family budget and two slots. An external-started marker commits before IO; provider IO always stays outside transactions. Every known provider outcome and its terminal/retry job state commit in one household transaction; an unknown outcome atomically leaves the job `unresolved`. A trusted resumer only moves eligible budget-waiting jobs back to `ready`; after Input Tokens counting, reservation, slot and the global unknown barrier are checked again under the same lock. The maintenance role cannot read or mutate AI/job tables directly: it can execute only the validated `SECURITY DEFINER` reconciliation transition for an expired or unresolved call. The application role cannot update immutable history. [Contract and verification](evidence/task-5.1-openai-gateway.en.md).

## Transaction review task-5.2

The new command schema/prompt does not inherit admission from historical task-0.8 qualification. Preserve task-5.1 model/effort, pricing and budget; `production_admitted=false`. Task-5.2 verification uses offline synthetic responses and PostgreSQL. It does not establish live qualification, OpenAI IO, receipt extraction, chat, product budget/goal handlers or browser acceptance. No new dependencies or public general-purpose execution tools are introduced. Automatic classification respects persisted field protections; user approval creates an ordinary audited decision supporting undo.
