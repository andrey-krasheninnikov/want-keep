<!-- want-keep-task: task-4.1 -->
# task-4.1 — Реализовать коннектор Альфа-Банк / Implement Alfa-Bank connector

## RU

Автоматически получать согласованные данные текущих счетов, дебетовых карт, операций и кэшбэка Alfa по D-47.

**Состояние:** В работе: зависимости закрыты. Первый живой read-прогон одного аккаунта завершён; финансовый mapping, конец истории, связь выплаты кэшбэка и второй аккаунт ещё не подтверждены. Коннектор не реализован, admission не включён. Отчёт: spec/001-want-keep-mvp/evidence/task-4.1-alfa.md.

**Зависимости:** `task-0.1`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать D-47: текущие счета, дебетовые карты, операции и кэшбэк. Накопительные счета, вклады и кредиты Alfa исключены по решению владельца от 2026-10-28; остальные провайдеры и общие финансовые функции сохраняются. Подтверждённая денежная выплата кэшбэка создаёт один доход в дату выплаты; ожидаемое начисление не создаёт проводку. До deployment доказать разрешённый структурированный read path, точный allowlist, стабильную account/source identity D-39, пагинацию/coverage, revisions/statuses/fees/cashback lifecycle, reauth и два независимых аккаунта. Browser collector допустим только для подтверждённого API-пробела. Неполная история даёт `source_partial`; неизвестное обязательное поле остаётся unknown; коллизия даёт `source_ambiguous` без проводки. Reconnect связывается с тем же внешним счётом; stale job после disconnect не применяется. Target-host route/DNS/TLS проверяются до deployment. Provider evidence передаётся admission service для точного D-43 binding; conformance до admission не создаёт source record или проводок. Смена binding закрывает sync.

### Границы изменений

- `backend/internal/integrations/alfa/`
- `collector/src/providers/alfa/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-036:** Вознаграждения майнинга отделены от переводов между собственными кошельками.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-042:** Alfa автоматически читает текущие счета, дебетовые карты, операции и кэшбэк по проверенному контракту.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-042

- **Дано:** Подключён разрешённый аккаунт Alfa с продуктами D-47.
- **Когда:** Запрошены текущие счета, остатки, дебетовые карты, операции и кэшбэк.
- **Тогда:** Данные совпадают с source evidence; накопительные счета, вклады и кредиты Alfa исключены. Unknown блокирует только deployment коннектора.
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

#### AC-070

- **Дано:** Банк передаёт баланс, но не условия грейса; ставка Earn имеет неизвестную базу начисления.
- **Когда:** Открываются прогнозы.
- **Тогда:** Баланс отображается; льгота и точный прогноз имеют причину недоступности; AI не извлекает гарантированную бизнес-логику из рекламной формулировки.
- **Уровень:** `contract+end-to-end`.

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
make test-contract PROVIDER=alfa && make test-integration AREA=alfa
```

Продукты D-47 проходят синтетические fixtures и отдельный разрешённый live readback; доказаны два аккаунта, reauth, pagination, revisions и deployment gate. Накопительные счета, вклады и кредиты Alfa не требуются. Неизвестные поля и gaps не проводят деньги.

Общие команды Make реализованы; suites PROVIDER=alfa и AREA=alfa добавляются этой задачей. Синтетические проверки, живое чтение, повторный вход и второй независимый аккаунт фиксируются отдельно. Без обязательного live-доказательства задача остаётся открытой, а deployment admission не включается.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Automatically retrieve consistent Alfa current-account, debit-card, transaction and cashback data under D-47.

**Status:** In progress: dependencies are closed. The first live read of one account is complete; financial mapping, history end, cashback payout links and a second account are not verified. The connector is not implemented and admission is disabled. Report: spec/001-want-keep-mvp/evidence/task-4.1-alfa.en.md.

**Dependencies:** `task-0.1`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

Implement D-47: current accounts, debit cards, transactions and cashback. Alfa savings accounts, deposits and credit products are excluded by the owner decision of 2026-10-28; other providers and shared financial features remain in scope. A confirmed cash cashback payout creates one income entry on its payout date; an expected accrual creates no posting. Before deployment, prove an authorized structured read path, an exact allowlist, stable D-39 account/source identity, pagination/coverage, revisions/statuses/fees/cashback lifecycle, reauthentication and two independent accounts. A browser collector is allowed only for a proven API gap. Incomplete history yields `source_partial`; an unknown mandatory field stays unknown; a collision yields `source_ambiguous` without posting. Reconnection links the same external account; a stale job cannot apply after disconnect. Target-host route/DNS/TLS are checked before deployment. Provider evidence goes to the admission service for the exact D-43 binding; pre-admission conformance creates no source records or postings. A binding change closes sync.

### Change boundaries

- `backend/internal/integrations/alfa/`
- `collector/src/providers/alfa/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-036:** Mining rewards are separate from transfers between owned wallets.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-042:** Alfa automatically reads current accounts, debit cards, transactions and cashback under a verified contract.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-042

- **Given:** An authorized Alfa account with D-47 products is connected.
- **When:** Current accounts, balances, debit cards, transactions and cashback are requested.
- **Then:** Data matches source evidence; Alfa savings accounts, deposits and credit products are excluded. Unknown blocks only connector deployment.
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

#### AC-070

- **Given:** A bank exposes balance but no grace terms; an Earn rate has an unknown accrual basis.
- **When:** Forecasts are opened.
- **Then:** Balance is shown; grace eligibility and exact forecasts explain unavailability; AI does not turn marketing wording into guaranteed business rules.
- **Level:** `contract+end-to-end`.

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
make test-contract PROVIDER=alfa && make test-integration AREA=alfa
```

D-47 products pass synthetic fixtures and a separate authorized live readback; two accounts, reauthentication, pagination, revisions and the deployment gate are proven. Alfa savings accounts, deposits and credit products are not required. Unknown fields and gaps do not post money.

Shared Make commands are implemented; this task adds the PROVIDER=alfa and AREA=alfa suites. Synthetic checks, live reads, reauthentication and a second independent account are recorded separately. Missing mandatory live evidence keeps the task open and deployment admission disabled.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
