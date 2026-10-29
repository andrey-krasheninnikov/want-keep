<!-- want-keep-task: task-4.7 -->
# task-4.7 — Подтвердить финансовый mapping и live-допуск Alfa / Verify Alfa financial mapping and live admission

## RU

Подтвердить финансовый mapping, историю, кэшбэк и deployment binding Alfa.

**Состояние:** Открыта; needs-info. Для финансового mapping и live-допуска нужны подтверждённые данные источника, второй аккаунт и разрешённая проверка точного deployment binding. Read-коннектор #29 завершён; production admission остаётся закрытым.

**Зависимости:** `task-4.1`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Read-коннектор #29 включён в базу: счета, карточные алиасы, unknown source records и зашифрованные evidence. Денежные проводки и подтверждённые банковские остатки не формируются; второй live-аккаунт ещё требуется.

### Требуемое поведение

Завершить финансовую нормализацию и live-приёмку Alfa после реализации read-коннектора в #29. По решению владельца эти проверки выделены в отдельную задачу и не останавливают реализацию транспорта. Production admission остаётся закрытым.

Объём D-47: текущие счета, дебетовые карты, операции и кэшбэк. Накопительные счета, вклады и кредитные продукты Alfa исключены.

### Что требуется

- Подтвердить состав `amount`, `total`, `holds`, `currentBalance`, единицы и источник времени баланса. Затем реализовать известные owned/available/locked/debt без выдуманного `sourceAsOf` и двойного вычитания удержаний.
- Проверить финансовые даты, `HOLD`/`SUCCESS`/`FAILED`/`null`, комиссии и lifecycle изменений. Реализовать проверенные переходы и экономическую классификацию; внутренний перевод не становится доходом или расходом.
- Подтвердить устойчивость account/operation/card IDs после повторного входа и на втором независимом аккаунте. Проверить reconnect и отсутствие слияния аккаунтов участников. Второй аккаунт будет доступен позже.
- Проверить фильтр выбранной даты, конец и глубину истории, поздние изменения и смещение страниц при новых операциях. До этого сохранять `source_partial`.
- Установить устойчивую связь начисления кэшбэка, банковской выплаты, исправления и отмены. Выплата создаёт один доход в дату выплаты; прогноз и начисления сами по себе не увеличивают средства.
- Проверить возможность официального API для личного проекта. Если остаётся кабинет, подтвердить актуальность закрытого read workflow и минимального набора сессионных заголовков.
- Проверить route/DNS/TLS на целевом хосте; сформировать provider/host evidence для точного D-43 binding. Не отключать TLS-проверку и не разрешать изменяющие банковские запросы.

### Приёмка

Сохранить обезличенные RU/EN-доказательства и синтетические fixtures. Обновить нормализацию, контрактные и PostgreSQL-тесты. Выполнить `make test-contract PROVIDER=alfa`, `make test-integration AREA=alfa` и collector security suite. Отдельно зафиксировать разрешённое живое чтение двух аккаунтов, повторный вход, историю и кэшбэк. Синтетические тесты не заменяют live-доказательства.

До завершения этой задачи текущий read-коннектор сохраняет счета и алиасы, source records со статусом unknown и зашифрованные evidence. Банковские операции не создают денежного эффекта, баланс не объявляется подтверждённым и production admission не включается.

Связи: task-4.7; зависит от task-4.1 / #29. REQ-039/041/042/048/065/073/076/088; AC-041/042/048/079/087/090/106. Общая приёмка task-9.1 зависит от завершения этой задачи.

### Ключевые контракты

Существующий Alfa read-коннектор, финансовая нормализация, source snapshots и lifecycle, provider/host evidence для D-43 binding.

### Вне объёма

Накопительные счета, вклады и кредитные продукты Alfa исключены по D-47. Не включать production admission и не проводить неподтверждённые денежные факты до закрытия live-gates.

### Блокирующие зависимости

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): выполнена.

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

Вопросы #94 закрыты раздельными synthetic/live доказательствами; два аккаунта изолированы, денежные эффекты подтверждены, D-43 binding проверен.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно. Требуется разрешённое живое чтение; второй аккаунт пока недоступен.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Verify Alfa financial mapping, history, cashback and deployment binding.

**Status:** Open; needs-info. Financial mapping and live admission require verified source data, a second account and an authorized check of the exact deployment binding. Read connector #29 is complete; production admission remains disabled.

**Dependencies:** `task-4.1`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

The read connector from #29 is included in the base: accounts, card aliases, unknown source records and encrypted evidence. It creates no monetary postings or confirmed bank balances; a second live account is still required.

### Desired behavior

Complete Alfa financial normalization and live acceptance after the read connector in #29. The owner moved these checks into a separate task so transport implementation can proceed. Production admission remains disabled.

D-47 scope: current accounts, debit cards, transactions and cashback. Alfa savings accounts, deposits and credit products are excluded.

### Required work

- Verify the meaning of `amount`, `total`, `holds`, `currentBalance`, units and the balance timestamp. Implement known owned/available/locked/debt without inventing `sourceAsOf` or subtracting holds twice.
- Verify financial dates, `HOLD`/`SUCCESS`/`FAILED`/`null`, fees and revision lifecycle. Implement confirmed transitions and economic classification; internal transfers must not become income or expenses.
- Verify account/operation/card IDs across reauthentication and a second independent account. Test reconnect and member-account isolation. The second account will be available later.
- Verify the selected-date filter, history end and depth, late revisions and page shifts under new transactions. Retain `source_partial` until then.
- Establish stable links between cashback accrual, bank payout, revision and cancellation. Recognize one income entry on the payout date; forecasts and accruals alone do not increase funds.
- Verify official API eligibility for the personal project. If cabinet access remains necessary, verify the closed read workflow and minimum session headers.
- Verify route/DNS/TLS on the target host and produce provider/host evidence for the exact D-43 binding. Do not disable TLS validation or permit bank mutations.

### Acceptance

Retain sanitized RU/EN evidence and synthetic fixtures. Update normalization, contract and PostgreSQL tests. Run `make test-contract PROVIDER=alfa`, `make test-integration AREA=alfa` and the collector security suite. Record authorized live reads of two accounts, reauthentication, history and cashback separately. Synthetic checks do not replace live evidence.

Until this task is complete, the read connector retains accounts and aliases, unknown source records and encrypted evidence. Bank transactions have no monetary effect, balances remain unconfirmed and production admission stays disabled.

Links: task-4.7; depends on task-4.1 / #29. REQ-039/041/042/048/065/073/076/088; AC-041/042/048/079/087/090/106. Full task-9.1 acceptance depends on this task.

### Key interfaces

Existing Alfa read connector, financial normalization, source snapshots and lifecycle, provider/host evidence for the D-43 binding.

### Out of scope

Alfa savings accounts, deposits and credit products are excluded under D-47. Do not enable production admission or post unqualified monetary facts before the live gates pass.

### Blocked by

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): completed.

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

Separate synthetic/live evidence resolves #94; two accounts are isolated, monetary effects confirmed and the D-43 binding verified.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately. Authorized live reading is required; the second account is not yet available.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
