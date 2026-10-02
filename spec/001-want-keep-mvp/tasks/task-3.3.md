<!-- want-keep-task: task-3.3 -->
# task-3.3 — Создать изолированный браузерный сборщик / Create an isolated browser collector

## RU

Читать кабинеты по разрешённым сценариям и передавать нормализуемые данные Go-приложению.

**Состояние:** Реализованы изолированный Unix-socket collector, build-owned read allowlist, отдельный BrowserContext для job, typed provider challenges, worker/vault integration и зашифрованное durable evidence. Реальные кабинеты, пользовательские Chrome/Arc-профили, production egress и provider admission не проверены.

**Зависимости:** `task-3.2`, `task-1.5`.

**Тип:** `implementation`.

### Изменение и контракты

Playwright runtime работает отдельным Node.js-процессом и слушает только Unix socket с правами 0600. Вход ограничен ready/capabilities/read, одним job, лимитами размера/времени и server-issued SyncRequest. Build-owned конфигурация задаёт exact D-43 binding, admissionRevision, manifest, origin и allowlist entry/read/request_statement; job не может передать URL, selector, JavaScript или route rule. Каждый job использует новый непостоянный BrowserContext с отключённым JavaScript; разрешённое чтение выполняется вне страницы с cookies для точного адреса. Popup, download, WebSocket, service worker, redirect, неизвестные routes/payload и мутации блокируются. MFA/CAPTCHA/reauth возвращаются типизированно. Worker временно получает browser_session через encrypted vault, фиксирует external_started непосредственно перед browser IO и не повторяет неизвестный outcome. Result проходит прежние admission/generation/lease/cursor fences; stale result попадает только в quarantine. Raw evidence, включая provider evidence ID, шифруется connection keyring с household/job/page/item AAD и хранится миграцией 021 как неизменяемые items с однонаправленным disposition. Реальные provider workflows и production admission остаются task-4.x/task-8.x.

### Границы изменений

- `collector/src/runtime/`
- `collector/tests/security.browser.test.ts`
- `backend/internal/connections/collector/`
- `backend/internal/integrations/`
- `backend/internal/jobs/`
- `backend/internal/storage/collector_evidence.go`
- `backend/cmd/worker/`
- `backend/migrations/021_collector_evidence.sql`
- `backend/test/integration/collector/`
- `.github/workflows/ci.yml`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-015:** Для отправки чека требуется счёт списания; фото/PDF остаётся связанным с результатом обработки.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

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

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-060

- **Дано:** В PDF или описании операции есть инструкция раскрыть ключ либо сделать перевод.
- **Когда:** Документ обрабатывается AI.
- **Тогда:** Инструкция считается данными; секреты и платёжные инструменты недоступны; недопустимая команда отклонена и не меняет учёт.
- **Уровень:** `integration`.

#### AC-061

- **Дано:** Процесс падает между сохранением записи и подтверждением задания.
- **Когда:** Задание повторяется, одновременно приходит правка владельца.
- **Тогда:** Применён один эффект, правка защищена версией, незавершённое состояние восстанавливается; внешняя неоднозначность не вызывает слепой повтор.
- **Уровень:** `integration`.

#### AC-068

- **Дано:** Загружаются повреждённый PDF, неверно обозначенный тип, чрезмерный файл и чек с вредоносным текстом.
- **Когда:** Срабатывают проверка файла и обработка.
- **Тогда:** Файл с ошибкой не проводится; нет выполнения вложенного кода, произвольного скачивания URL или публичного доступа; понятная причина/уточнение видна в чате.
- **Уровень:** `integration`.

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
make test-collector FILTER=security && make test-integration AREA=collector
```

Synthetic browser suite подтверждает безопасное чтение и statement POST, изоляцию сессий, typed MFA/CAPTCHA и блокировку payment/redirect/popup/download/WebSocket/service worker. PostgreSQL suite подтверждает ciphertext-only evidence, AAD, семейную изоляцию, рестарт, staged recovery и terminal disposition. Admission/generation/lease fences не допускают устаревший финансовый результат.

Доказательства и границы: evidence/task-3.3-collector.md. Обязательны make check, collector security/integration, полная integration matrix и затронутые ingestion/jobs/storage/privacy race suites. Browser-тесты используют только локальный синтетический портал.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Read portals through authorized workflows and return normalizable data to Go.

**Status:** The isolated Unix-socket collector, build-owned read allowlist, per-job BrowserContext, typed provider challenges, worker/vault integration and encrypted durable evidence are implemented. Live portals, user Chrome/Arc profiles, production egress and provider admission are not verified.

**Dependencies:** `task-3.2`, `task-1.5`.

**Kind:** `implementation`.

### Change and contracts

The Playwright runtime is a separate Node.js process listening only on a mode-0600 Unix socket. Input is limited to ready/capabilities/read, one job, size/time limits and a server-issued SyncRequest. Build-owned configuration defines the exact D-43 binding, admissionRevision, manifest, origin and entry/read/request_statement allowlist; a job cannot supply a URL, selector, JavaScript or route rule. Every job uses a new non-persistent BrowserContext with JavaScript disabled; the approved read runs outside the page with cookies for the exact target. Popups, downloads, WebSockets, service workers, redirects, unknown routes/payloads and mutations are blocked. MFA/CAPTCHA/reauthentication return typed outcomes. The worker temporarily borrows browser_session through the encrypted vault, records external_started immediately before browser IO and never replays an unknown outcome. Results retain the existing admission/generation/lease/cursor fences and stale results enter quarantine only. Raw evidence, including provider evidence IDs, is encrypted through the connection keyring with household/job/page/item AAD and migration 021 stores immutable items with one-way disposition. Live provider workflows and production admission remain task-4.x/task-8.x.

### Change boundaries

- `collector/src/runtime/`
- `collector/tests/security.browser.test.ts`
- `backend/internal/connections/collector/`
- `backend/internal/integrations/`
- `backend/internal/jobs/`
- `backend/internal/storage/collector_evidence.go`
- `backend/cmd/worker/`
- `backend/migrations/021_collector_evidence.sql`
- `backend/test/integration/collector/`
- `.github/workflows/ci.yml`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-015:** Receipt submission requires a debit account; the photo/PDF stays linked to the processing result.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

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

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-060

- **Given:** A PDF or transaction description instructs the agent to reveal a key or transfer funds.
- **When:** AI processes the document.
- **Then:** The instruction is treated as data; secrets and payment tools are unavailable; an invalid command is rejected without changing accounting.
- **Level:** `integration`.

#### AC-061

- **Given:** A process crashes between persisting a record and acknowledging its job.
- **When:** The job is retried while the owner submits a correction.
- **Then:** One effect is applied, the correction is version-protected and incomplete state recovers; an ambiguous external outcome is not blindly retried.
- **Level:** `integration`.

#### AC-068

- **Given:** A corrupt PDF, mislabeled type, oversized file and prompt-injected receipt are uploaded.
- **When:** File validation and processing run.
- **Then:** Invalid files do not post; embedded code, arbitrary URL fetching and public access are unavailable; chat shows an understandable reason or clarification.
- **Level:** `integration`.

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
make test-collector FILTER=security && make test-integration AREA=collector
```

The synthetic browser suite proves safe reads and statement POSTs, session isolation, typed MFA/CAPTCHA and payment/redirect/popup/download/WebSocket/service-worker blocking. The PostgreSQL suite proves ciphertext-only evidence, AAD, household isolation, restart, staged recovery and terminal disposition. Admission/generation/lease fences prevent stale financial results.

Evidence and boundaries: evidence/task-3.3-collector.en.md. Require make check, collector security/integration, the full integration matrix and affected ingestion/jobs/storage/privacy race suites. Browser tests use only the local synthetic portal.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
