<!-- want-keep-task: task-0.1 -->
# task-0.1 — Проверить контракт чтения Альфа-Банк / Verify Alfa-Bank read contract

## RU

Получить проверяемую матрицу доступа к обязательным продуктам Альфа-Банк.

**Состояние:** Исследование завершено 2026-09-07. D-37 фиксирует текущий объём Alfa: дебетовая карта, текущий и накопительные счета, вклады и кэшбэк; кредитка Alfa отложена без блокировки общей ручной поддержки кредиток. Доступная авторизованная вкладка подтвердила продуктовые разделы и историю без публикации финансовых данных. Структурированный fixture, право автоматизации, второй аккаунт и reauth перенесены в entry/runtime gate task-4.1; SDD Ready не заявляет работающий коннектор.

**Зависимости:** нет.

**Тип:** `research`.

### Изменение и контракты

Проверить официальный API и доступ к личным счетам; при его отсутствии исследовать разрешённое чтение авторизованного кабинета. Для каждого продукта зафиксировать счета, остатки, операции, устойчивые ID, статусы, комиссии, пагинацию, глубину истории, условия/сроки, котировки, требования MFA и границы прав. Хранить только синтетические или обезличенные контракты; секреты подключает владелец вне репозитория. Недоступность продукта или платный обязательный доступ оформить блокером конкретного адаптера; не подменять автоматизацию ручной выпиской. Проверить независимые внешние аккаунты участников одной платформы и устойчивую идентичность при повторной авторизации; не считать connectionId идентификатором реального счёта.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/alfa.md`
- `spec/001-want-keep-mvp/evidence/alfa.en.md`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-042:** Alfa автоматически читает текущие счета, дебетовые карты, операции и кэшбэк по проверенному контракту.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-042

- **Дано:** Подключён разрешённый аккаунт Alfa с продуктами D-47.
- **Когда:** Запрошены текущие счета, остатки, дебетовые карты, операции и кэшбэк.
- **Тогда:** Данные совпадают с source evidence; накопительные счета, вклады и кредиты Alfa исключены. Unknown блокирует только deployment коннектора.
- **Уровень:** `contract+manual`.

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

### Проверка результата

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Документ на двух языках содержит источник и дату, проверенные и непроверенные поля, примеры запросов/ответов без секретов и итог по каждому продукту. Реальный read-only прогон выполняется только после безопасного предоставления доступа владельцем; его отсутствие явно записано.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Produce a verifiable access matrix for mandatory Alfa-Bank products.

**Status:** Research completed on 2026-09-07. D-37 defines the current Alfa scope: debit card, current and savings accounts, deposits and cashback; an Alfa credit card is deferred without blocking shared manual credit-card support. The available signed-in tab confirmed product sections and history without publishing financial data. A structured fixture, automation permission, second account and reauthentication are task-4.1 entry/runtime gates; SDD Ready does not claim a running connector.

**Dependencies:** none.

**Kind:** `research`.

### Change and contracts

Verify the official API and personal-account eligibility; otherwise investigate authorized reading of the signed-in portal. For each product record accounts, balances, transactions, stable IDs, statuses, fees, pagination, history depth, terms/deadlines, quotes, MFA and permission boundaries. Retain only synthetic or sanitized contracts; the owner supplies secrets outside the repository. An inaccessible product or mandatory paid access blocks its adapter; manual statements do not substitute for automation. Verify independent member accounts at the same provider and stable identity across reauthorization; do not treat connectionId as real-account identity.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/alfa.md`
- `spec/001-want-keep-mvp/evidence/alfa.en.md`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-042:** Alfa automatically reads current accounts, debit cards, transactions and cashback under a verified contract.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-042

- **Given:** An authorized Alfa account with D-47 products is connected.
- **When:** Current accounts, balances, debit cards, transactions and cashback are requested.
- **Then:** Data matches source evidence; Alfa savings accounts, deposits and credit products are excluded. Unknown blocks only connector deployment.
- **Level:** `contract+manual`.

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

### Verification

```sh
python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

The bilingual document includes dated sources, verified/unverified fields, secret-free request/response examples and per-product outcomes. A live read-only check runs only after the owner securely supplies access; absence of access is explicit.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
