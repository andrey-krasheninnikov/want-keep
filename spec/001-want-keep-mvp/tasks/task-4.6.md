<!-- want-keep-task: task-4.6 -->
# task-4.6 — Реализовать коннектор EMCD / Implement EMCD connector

## RU

Автоматически читать кошелёк USDT, используемые Grow/криптокарты и архив P2P EMCD по D-34.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-0.6`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Исследовательские контракты и общая инфраструктура импорта, collector и admission включены в базу. Они не подтверждают финансовый mapping или live-допуск конкретного провайдера.

### Требуемое поведение

Реализовать D-34: кошелёк USDT, Grow, используемые криптокарты и историю P2P. До provider deployment получить разрешённые структурированные fixtures каждого журнала, read allowlist, стабильную D-39 identity, pagination/coverage, revisions/statuses/fees, card/Grow/P2P lifecycle, reauth и два аккаунта. UI-текст не создаёт проводку. Разделить aggregate, wallet, Grow, card owned/available/reserve; reward/capitalization/payout и card authorization/clearing/refund/reversal связывать без двойного эффекта. USDT funding, USD card и приблизительная оценка EUR-покупки — разные факты. Unknown fee/owner side/legs и history gaps остаются явными; коллизия даёт `source_ambiguous` без проводки. Stale job после disconnect не применяется. Майнинг и другие неиспользуемые продукты отложены без блокировки. Provider evidence публикуется admission service для точного D-43 binding; conformance до admission идёт в quarantine без source record/проводки, а смена binding снова закрывает sync.

### Ключевые контракты

Read-контракты EMCD wallet/Grow/card/P2P, source revisions, нормализация сумм и admission.

### Вне объёма

Не добавлять изменяющие банковские или торговые действия, несогласованные продукты и обход admission. Не заменять автоматический импорт ручной выпиской или UI-сводкой.

### Блокирующие зависимости

- [#6](https://github.com/andrey-krasheninnikov/want-keep/issues/6) (`task-0.6`): выполнена.
- [#28](https://github.com/andrey-krasheninnikov/want-keep/issues/28) (`task-3.3`): выполнена.
- [#20](https://github.com/andrey-krasheninnikov/want-keep/issues/20) (`task-2.4`): выполнена.
- [#21](https://github.com/andrey-krasheninnikov/want-keep/issues/21) (`task-2.5`): выполнена.

### Границы изменений

- `backend/internal/integrations/emcd/`
- `collector/src/providers/emcd/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-047:** Интеграция EMCD автоматически читает используемые криптокарты, Coinhold/Grow, кошелёк USDT и исторические P2P-ордера по D-34; майнинг никогда не использовался и вместе с другими неиспользуемыми продуктами отложен без блокировки.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-047

- **Дано:** Подключён разрешённый личный аккаунт EMCD с кошельком USDT, действующими Grow, существующими картами, включая заблокированную, и историей P2P. Майнинг не использовался ни сейчас, ни ранее.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** По каждому включённому продукту подтверждены сопоставимые с источником данные и автоматическое чтение: сводки не дублируют дочерние остатки, начисление/капитализация/выплата не утраивают доход, отказ карты не расход по основной сумме, P2P связан с денежными сторонами без дубля. Неизвестные поля/история отмечены явно; отсутствие контракта выбранных продуктов блокирует адаптер. Майнинг и другие неиспользуемые продукты не требуются.
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
make test-contract PROVIDER=emcd && make test-integration AREA=emcd
```

кошелёк USDT, Grow, используемые криптокарты и P2P проходят синтетические fixtures и отдельный разрешённый live readback; доказаны identity, полный обход страниц, revisions, reauth, два аккаунта и deployment gate. Unknown и коллизии не проводят деньги.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Automatically read the EMCD USDT wallet, used Grow/crypto cards and P2P archive under D-34.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-0.6`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Research contracts and shared ingestion, collector and admission infrastructure are included in the base. They do not establish the provider-specific financial mapping or live admission.

### Desired behavior

Implement D-34: the USDT wallet, Grow, existing crypto cards and P2P history. Before provider deployment obtain authorized structured fixtures for every log, a read allowlist, stable D-39 identity, pagination/coverage, revisions/statuses/fees, card/Grow/P2P lifecycle, reauthentication and two accounts. UI text never creates a posting. Separate aggregate, wallet, Grow and card owned/available/reserve; link reward/capitalization/payout and card authorization/clearing/refund/reversal without duplicate effects. USDT funding, the USD card and an approximate valuation of an EUR purchase are separate facts. Unknown fee/owner side/legs and history gaps stay explicit; a collision yields `source_ambiguous` without posting. A stale job cannot apply after disconnect. Mining and other unused products are deferred without blocking. Provider evidence is supplied to the admission service for the exact D-43 binding; pre-admission conformance runs in quarantine without source records/postings, and any binding change closes sync again.

### Key interfaces

EMCD wallet/Grow/card/P2P read contracts, source revisions, amount normalization and admission.

### Out of scope

Do not add bank or trading mutations, unapproved products or admission bypasses. Do not replace automated ingestion with manual statements or UI summaries.

### Blocked by

- [#6](https://github.com/andrey-krasheninnikov/want-keep/issues/6) (`task-0.6`): completed.
- [#28](https://github.com/andrey-krasheninnikov/want-keep/issues/28) (`task-3.3`): completed.
- [#20](https://github.com/andrey-krasheninnikov/want-keep/issues/20) (`task-2.4`): completed.
- [#21](https://github.com/andrey-krasheninnikov/want-keep/issues/21) (`task-2.5`): completed.

### Change boundaries

- `backend/internal/integrations/emcd/`
- `collector/src/providers/emcd/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-047:** The EMCD integration automatically reads used crypto cards, Coinhold/Grow, the USDT wallet and historical P2P orders under D-34; mining has never been used and is deferred with other unused products without blocking readiness.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-047

- **Given:** An authorized personal EMCD account has a USDT wallet, existing Grow deposits, existing cards including a blocked card, and P2P history. Mining has never been used.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Each included product has source-matching data and verified automatic reading: aggregates do not duplicate child balances; accrual/capitalization/payout do not triple income; declined card principal is not an expense; P2P links to monetary legs without duplicates. Unknown fields/history are explicit; missing selected-product contracts block the adapter. Mining and other unused products are not required.
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
make test-contract PROVIDER=emcd && make test-integration AREA=emcd
```

The USDT wallet, Grow, existing crypto cards and P2P pass synthetic fixtures and a separate authorized live readback; identity, complete pagination, revisions, reauthentication, two accounts and the deployment gate are proven. Unknown values and collisions do not post money.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
