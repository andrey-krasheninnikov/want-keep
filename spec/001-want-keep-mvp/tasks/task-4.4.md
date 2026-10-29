<!-- want-keep-task: task-4.4 -->
# task-4.4 — Реализовать коннектор Bybit / Implement Bybit connector

## RU

Автоматически получать Funding USDT/USDC/ETH/BTC, используемый Easy Earn и P2P по D-36 без повторного финансового эффекта.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-0.4`, `task-3.3`, `task-2.4`, `task-2.5`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Исследовательские контракты и общая инфраструктура импорта, collector и admission включены в базу. Они не подтверждают финансовый mapping или live-допуск конкретного провайдера.

### Требуемое поведение

Реализовать официальный read-only API для Funding USDT/USDC/ETH/BTC, используемого Flexible Easy Earn и P2P по D-36; финансовые create/pay/release/ads/transfer/stake/redeem запрещены allowlist. Route-specific provider IDs живут в отдельных product/log namespaces D-39. Связи Funding/Earn/P2P по amount/time — только кандидаты. Если hourly record ID отсутствует, fallback `(coin, productId, hourlyDate)` допустим только в hourly namespace; различающийся payload создаёт `source_ambiguous`, сохраняет обе evidence revisions и не проводит деньги. Сохранять 18-значные decimals, cursor даже на короткой странице, coverage, seconds/ms semantics, revisions, native P2P fiat/quantity/quote и unknown fee. Principal, yield и funding legs учитываются один раз. До provider deployment проверить RSA readOnly, lifetime/history gaps, precision reconciliation, два аккаунта, rotation/revocation и stale jobs. Collector допускается только при новом доказанном API-пробеле; Fixed/прочие продукты не требуются. Provider evidence публикуется admission service для точного D-43 binding; conformance до admission идёт в quarantine без source record/проводки, а смена binding снова закрывает sync.

### Ключевые контракты

Официальный read-only API Bybit, Funding/Easy Earn/P2P namespaces, точные native-суммы, cursor и admission.

### Вне объёма

Не добавлять изменяющие банковские или торговые действия, несогласованные продукты и обход admission. Не заменять автоматический импорт ручной выпиской или UI-сводкой.

### Блокирующие зависимости

- [#4](https://github.com/andrey-krasheninnikov/want-keep/issues/4) (`task-0.4`): выполнена.
- [#28](https://github.com/andrey-krasheninnikov/want-keep/issues/28) (`task-3.3`): выполнена.
- [#20](https://github.com/andrey-krasheninnikov/want-keep/issues/20) (`task-2.4`): выполнена.
- [#21](https://github.com/andrey-krasheninnikov/want-keep/issues/21) (`task-2.5`): выполнена.

### Границы изменений

- `backend/internal/integrations/bybit/`
- `collector/src/providers/bybit/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-045:** Bybit автоматически читает Funding USDT/USDC/ETH/BTC, используемый Easy Earn и P2P; официальный API приоритетен. Остальные продукты отложены без блокировки по D-36.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-045

- **Дано:** Безопасно подключён read-only аккаунт Bybit с подтверждёнными контрактами Funding, используемого Easy Earn и P2P.
- **Когда:** Читаются остатки, история с выбранной даты, повторные страницы, Convert, начисление/выплата Earn и P2P; моделируется отказ прав выбранного продукта.
- **Тогда:** Точные native amounts, ID, статусы, комиссии и coverage сопоставимы с источником; Funding и детали не удваивают обмен, комиссию или доход. P2P связывает crypto/fiat с банком либо требует уточнения. Отказ доступа/неполная история явно блокируют соответствующее покрытие; отсутствие Spot/UTA trading, futures, карты, On-Chain/Advanced Earn и иных неиспользуемых продуктов не блокирует.
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

#### AC-033

- **Дано:** Есть вклад или Earn с подтверждёнными условиями, пополнением и выводом.
- **Когда:** Рассчитывается доход за период и прогноз.
- **Тогда:** Факт отделён от прогноза и переоценки; смена ставки и капитализация учитываются по условиям; неизвестные условия блокируют точный прогноз.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
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
make test-contract PROVIDER=bybit && make test-integration AREA=bybit
```

12 сценариев evidence/bybit.samples.json превращены в исполняемые contract fixtures с учётом BYBIT-E11–E18. Проверены replay/18-decimal/precision mismatch, два Convert движения и fee/yield один раз, нулевой yield, P2P rounding/bank linkage, missing hourly ID, cursor на короткой странице, partial/expired/forbidden, два владельца и stale jobs после отключения. Read-only live readback выбранных продуктов отделён от локальных тестов; исследовательский probe не заменяет работающий коннектор.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Automatically read Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P under D-36 without duplicate financial effects.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-0.4`, `task-3.3`, `task-2.4`, `task-2.5`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Research contracts and shared ingestion, collector and admission infrastructure are included in the base. They do not establish the provider-specific financial mapping or live admission.

### Desired behavior

Implement official read-only APIs for Funding USDT/USDC/ETH/BTC, used Flexible Easy Earn and P2P under D-36; the allowlist forbids financial create/pay/release/ads/transfer/stake/redeem operations. Route-specific provider IDs live in separate D-39 product/log namespaces. Amount/time matches across Funding/Earn/P2P are candidates only. When an hourly record ID is absent, fallback `(coin, productId, hourlyDate)` is allowed only in the hourly namespace; a different payload yields `source_ambiguous`, retains both evidence revisions and posts no money. Preserve 18-place decimals, cursors even on short pages, coverage, seconds/ms semantics, revisions, native P2P fiat/quantity/quote and unknown fees. Account for principal, yield and funding legs once. Before provider deployment verify RSA read-only access, lifetime/history gaps, precision reconciliation, two accounts, rotation/revocation and stale jobs. A collector is allowed only for a newly proven API gap; Fixed/other products are not required. Provider evidence is supplied to the admission service for the exact D-43 binding; pre-admission conformance runs in quarantine without source records/postings, and any binding change closes sync again.

### Key interfaces

Official Bybit read-only API, Funding/Easy Earn/P2P namespaces, exact native amounts, cursors and admission.

### Out of scope

Do not add bank or trading mutations, unapproved products or admission bypasses. Do not replace automated ingestion with manual statements or UI summaries.

### Blocked by

- [#4](https://github.com/andrey-krasheninnikov/want-keep/issues/4) (`task-0.4`): completed.
- [#28](https://github.com/andrey-krasheninnikov/want-keep/issues/28) (`task-3.3`): completed.
- [#20](https://github.com/andrey-krasheninnikov/want-keep/issues/20) (`task-2.4`): completed.
- [#21](https://github.com/andrey-krasheninnikov/want-keep/issues/21) (`task-2.5`): completed.

### Change boundaries

- `backend/internal/integrations/bybit/`
- `collector/src/providers/bybit/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-045:** Bybit automatically reads Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P; the official API is preferred. Other products are deferred without blocking under D-36.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-045

- **Given:** A read-only Bybit account is securely connected with verified Funding, used Easy Earn and P2P contracts.
- **When:** Balances, history from the selected date, replayed pages, Convert, Earn accrual/payout and P2P are read; a selected-product permission failure is simulated.
- **Then:** Exact native amounts, IDs, statuses, fees and coverage match the source; Funding and details do not duplicate an exchange, fee or income. P2P links crypto/fiat with the bank or requires clarification. Access failure/incomplete history explicitly blocks the corresponding coverage; absent Spot/UTA trading, futures, card, On-Chain/Advanced Earn and other unused products do not block.
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

#### AC-033

- **Given:** A deposit or Earn product has confirmed terms, a top-up and a withdrawal.
- **When:** Period income and forecast are calculated.
- **Then:** Actual income is separate from forecast and revaluation; rate changes and compounding follow the terms; unknown terms prevent an exact forecast.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
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
make test-contract PROVIDER=bybit && make test-integration AREA=bybit
```

Turn the 12 evidence/bybit.samples.json scenarios into executable contract fixtures using BYBIT-E11–E18. Check replay/18-decimal/precision mismatch, both Convert legs and fee/yield once, zero yield, P2P rounding/bank linkage, missing hourly ID, short-page cursor, partial/expired/forbidden, two owners and stale jobs after disconnect. Separate selected-product read-only live readback from local tests; the research probe is not a running connector.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
