<!-- want-keep-task: task-4.2 -->
# task-4.2 — Реализовать коннектор Райффайзенбанк РФ / Implement Raiffeisenbank Russia connector

## RU

Автоматически получать согласованные остатки и движения расчётного счёта ИП через RBO API по D-35.

**Состояние:** Backend и нативный RBO/CAMT-контракт реализованы; live-чтение одного счёта подтверждено. Допуск provider/host остаётся отдельным deployment gate.

**Зависимости:** `task-0.2`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать RBO API для расчётного счёта ИП по D-35. Account UUID и number/accountKeys хранить отдельно; RUR явно переводить в RUB. CAMT.053 допускает 1:N между entry и transaction details без двойной проводки. Identity: для каждого проводимого detail вычислять достаточный versioned `camtCrossReportFingerprint` из полей, доказанно неизменных между перекрывающимися camt.052/camt.053: непустых structured references, party/account/remittance и bank-code fields в документированном порядке. Он всегда становится canonical providerRecordId, даже если присутствует NtryRef/AcctSvcrRef/EndToEndId. Эти optional ID и statement/report ID сохраняются как aliases/provenance; aliases регистрируются атомарно с canonical source record. Amount/time исключены. Недостаточный fingerprint, один alias у разных fingerprints или один fingerprint для разных фактов даёт `source_ambiguous`, сохраняет evidence и не создаёт новую проводку. Corrections/reversals создают revisions. `no-statements` не означает нулевой остаток. При недоступном camt.052 показывать последний подтверждённый CLBD с `asOf`/coverage; available/locked/balance/fee без evidence остаются unknown. До provider deployment проверить Code Flow/rotation, allowlist, полноту истории, reauth, два аккаунта и stale jobs. Playwright допустим только при доказанном API-пробеле. Provider evidence публикуется admission service для точного D-43 binding; conformance до admission идёт в quarantine без source record/проводки, а смена binding снова закрывает sync. Нативный Go gateway использует только allowlist legacy RBO. Пользовательские create/sync/reauth/disconnect сохраняют команду и результат; authorization/callback — отдельная одноразовая церемония владельца с session/state/nonce/PKCE и проверкой signed ID token по доверенному issuer/JWKS. Refresh и создание отчёта фиксируют намерение до IO; неизвестный результат не повторяется автоматически. История загружается дневными страницами, текущий день через camt.052, следующие синхронизации перекрывают семь дней. Raw digest доказывает evidence, но не изменяет семантический hash факта из-за нового report ID. Изменённый fingerprint-факт признаётся correction только при ранее сохранённой сильной банковской reference и более новом sourceAsOf; иначе source_ambiguous. Нативный adapter проходит те же vault/admission/generation/lease/CommitPage fences. Live conformance сохраняет ответы приватно вне Git, без journal effect и без выдачи допуска.

### Границы изменений

- `backend/internal/integrations/raiffeisen/`
- `backend/internal/connections/`
- `backend/internal/delivery/connections/`
- `backend/internal/storage/`
- `backend/migrations/023_raiffeisen_connector.sql`
- `backend/cmd/raiffeisen-conformance/`
- `api/paths/connections.yaml`
- `api/schemas/connections.yaml`
- `backend/test/integration/raiffeisen/`
- `spec/001-want-keep-mvp/evidence/task-4.2-raiffeisen.md`
- `spec/001-want-keep-mvp/evidence/task-4.2-raiffeisen.en.md`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-036:** Вознаграждения майнинга отделены от переводов между собственными кошельками.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-043:** Raiffeisen через RBO API читает только расчётный счёт ИП: остатки, поступления, списания, комиссии и историю (D-35).
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-043

- **Дано:** Подключён разрешённый расчётный счёт ИП в RBO API.
- **Когда:** Запрошены остатки и движения, повторный импорт и intraday no-statements.
- **Тогда:** Данные совпадают с источником; дублей нет, комиссии учтены отдельно. Неизвестный текущий остаток не равен нулю: видны последний подтверждённый остаток, его дата и пробел покрытия.
- **Уровень:** `contract+manual`.

#### AC-040

- **Дано:** Два источника доступны, третий требует повторного входа.
- **Когда:** Срабатывает расписание и одновременно нажата кнопка обновления.
- **Тогда:** Нет параллельного дублирования одного задания; доступные источники обновлены, проблемный имеет отдельный статус и старый timestamp.
- **Уровень:** `integration`.

#### AC-041

- **Дано:** Источник выдаёт несколько страниц с ограничением глубины; второй запрос завершился ошибкой.
- **Когда:** Импорт возобновляется.
- **Тогда:** Подтверждённые страницы сохранены без дублей; курсор не перескакивает пропуск; неполная история и её границы видны.
- **Уровень:** `integration`.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-063

- **Дано:** Приходящая сторона USDT 100 уже импортирована; исходящая RUB 9 000 и комиссия ещё отсутствуют.
- **Когда:** Приходят поздняя сторона, исправление комиссии и повтор старой страницы.
- **Тогда:** Состояние ожидания связи сменяется проверенным обменом; доход/расход основной суммы не удваивается, устаревшая комиссия не восстанавливается.
- **Уровень:** `integration`.

#### AC-071

- **Дано:** Источник различает gross P&L, net P&L, fee, funding и reward/transfer.
- **Когда:** Одна экономическая операция встречается в нескольких журналах.
- **Тогда:** Происхождение показателей сохранено; комиссия и доход не удваиваются; выбор net/gross подтверждён контрактом.
- **Уровень:** `contract+integration`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-106

- **Дано:** Подключение авторизовано, но provider/host gate неполон либо прошлый admission относится к другой версии binding.
- **Когда:** Участник или scheduler запрашивает sync, либо меняются build, contract, allowlist, config, permission или environment.
- **Тогда:** Если binding уже неполон или устарел, сервер возвращает `provider_not_admitted` без job, collector IO и проводки. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding. Job/result несёт неизменяемые binding и `admissionRevision`; смена binding во время read отменяет работу best effort, а обязательная commit-time revalidation сохраняет stale result в quarantine без source record или проводки.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make test-contract PROVIDER=raiffeisen && make test-integration AREA=raiffeisen
```

Синтетические проверки покрывают CAMT 1:N, точные суммы, optional aliases, unknown fee/balance, OIDC signature/binding, replay выписки через worker, OAuth owner/session/claim/readback и commit-time quarantine. Live 2026-10-03: список одного счёта и новая выписка за август, 7 нормализованных details. Полнота истории, браузерный Code Flow, реальные два аккаунта и production admission не доказаны.

Согласованный результат #30 — код и live-чтение текущего расчётного счёта ИП. Разрешения, доверенный issuer/JWKS, Code Flow, повторный вход, второй реальный аккаунт и точный provider/host binding подтверждаются до deployment; без них sync не разрешается. См. evidence/task-4.2-raiffeisen.md.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Automatically retrieve consistent balances and movements of the individual entrepreneur current account through RBO API under D-35.

**Status:** Backend and native RBO/CAMT contract implemented; live reading of one account confirmed. Provider/host admission remains a separate deployment gate.

**Dependencies:** `task-0.2`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

Implement the RBO API for the individual entrepreneur current account under D-35. Keep Account UUID and number/accountKeys separate; map RUR explicitly to RUB. CAMT.053 permits a 1:N relation between an entry and transaction details without double posting. Identity: for every postable detail compute a sufficient versioned `camtCrossReportFingerprint` from fields proven invariant across overlapping camt.052/camt.053: non-empty structured references, party/account/remittance and bank-code fields in documented order. It is always the canonical providerRecordId even when NtryRef/AcctSvcrRef/EndToEndId is present. Those optional IDs and statement/report ID remain aliases/provenance; aliases are registered atomically with the canonical source record. Amount/time are excluded. An insufficient fingerprint, one alias mapped to different fingerprints or one fingerprint covering different facts yields `source_ambiguous`, retains evidence and creates no new posting. Corrections/reversals create revisions. `no-statements` is not a zero balance. When camt.052 is unavailable, show the last confirmed CLBD with `asOf`/coverage; available/locked/balance/fee without evidence stay unknown. Before provider deployment verify Code Flow/rotation, allowlist, history completeness, reauthentication, two accounts and stale jobs. Playwright is allowed only for a proven API gap. Provider evidence is supplied to the admission service for the exact D-43 binding; pre-admission conformance runs in quarantine without source records/postings, and any binding change closes sync again. The native Go gateway uses only the legacy RBO allowlist. User create/sync/reauth/disconnect persist command results; authorization/callback is a separate one-use owner ceremony bound to session/state/nonce/PKCE with signed ID-token validation against trusted issuer/JWKS. Refresh and report creation persist intent before IO; unknown outcomes are never retried automatically. History uses daily pages, camt.052 for the current day, and seven-day overlap on later syncs. A raw digest proves evidence but does not change the semantic fact hash because of a new report ID. A changed fingerprint fact becomes a correction only with a previously stored strong bank reference and newer sourceAsOf; otherwise it is source_ambiguous. The native adapter uses the same vault/admission/generation/lease/CommitPage fences. Live conformance retains responses privately outside Git, without a journal effect or admission grant.

### Change boundaries

- `backend/internal/integrations/raiffeisen/`
- `backend/internal/connections/`
- `backend/internal/delivery/connections/`
- `backend/internal/storage/`
- `backend/migrations/023_raiffeisen_connector.sql`
- `backend/cmd/raiffeisen-conformance/`
- `api/paths/connections.yaml`
- `api/schemas/connections.yaml`
- `backend/test/integration/raiffeisen/`
- `spec/001-want-keep-mvp/evidence/task-4.2-raiffeisen.md`
- `spec/001-want-keep-mvp/evidence/task-4.2-raiffeisen.en.md`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-036:** Mining rewards are separate from transfers between owned wallets.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-043:** Raiffeisen RBO API reads only the entrepreneur current account: balances, receipts, debits, fees and history (D-35).
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-043

- **Given:** An authorized entrepreneur current account is connected through RBO API.
- **When:** Request balances, movements, repeated import and intraday no-statements.
- **Then:** Data matches the source; no duplicates, fees recorded separately. Unknown current balance is not zero: show the last verified balance, its date and the coverage gap.
- **Level:** `contract+manual`.

#### AC-040

- **Given:** Two sources are available and a third requires sign-in again.
- **When:** The schedule fires while the refresh button is pressed.
- **Then:** The same job is not duplicated concurrently; available sources refresh and the failing source has its own status and old timestamp.
- **Level:** `integration`.

#### AC-041

- **Given:** A source provides paginated history with a retention limit; the second request fails.
- **When:** Import resumes.
- **Then:** Confirmed pages remain without duplicates; the cursor does not skip the gap; incomplete history and its boundaries are visible.
- **Level:** `integration`.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-063

- **Given:** The incoming USDT 100 leg is imported; outgoing RUB 9,000 and fee are missing.
- **When:** The late leg, fee correction and replayed old page arrive.
- **Then:** Pending matching becomes a verified exchange; principal is not double-counted and the stale fee is not restored.
- **Level:** `integration`.

#### AC-071

- **Given:** A source distinguishes gross P&L, net P&L, fee, funding and reward/transfer.
- **When:** One economic event appears in several logs.
- **Then:** Metric provenance is preserved; fees and income are not doubled; net/gross semantics are contract-verified.
- **Level:** `contract+integration`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-106

- **Given:** A connection is authenticated, but the provider/host gate is incomplete or the prior admission belongs to a different binding revision.
- **When:** A member or scheduler requests sync, or the build, contract, allowlist, configuration, permission or environment changes.
- **Then:** If the binding is already incomplete or stale, the server returns `provider_not_admitted` with no job, collector IO or posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding. Each job/result carries immutable binding and `admissionRevision`; a binding change during a read cancels work best effort, while mandatory commit-time revalidation retains a stale result in quarantine without a source record or posting.
- **Level:** `integration+security`.

### Verification

```sh
make test-contract PROVIDER=raiffeisen && make test-integration AREA=raiffeisen
```

Synthetic checks cover CAMT 1:N, exact amounts, optional aliases, unknown fee/balance, OIDC signature/binding, statement replay through the worker, OAuth owner/session/claim/readback and commit-time quarantine. Live 2026-10-03: one account and a new August statement with 7 normalized details. History completeness, browser Code Flow, two real accounts and production admission are not proven.

The agreed #30 outcome is code and live reading of the existing entrepreneur current account. Permission, trusted issuer/JWKS, Code Flow, reauthentication, a second real account and the exact provider/host binding must be confirmed before deployment; otherwise sync remains denied. See evidence/task-4.2-raiffeisen.en.md.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
