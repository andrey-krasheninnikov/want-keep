<!-- want-keep-task: task-5.3 -->
# task-5.3 — Обрабатывать чеки и позиции / Process receipts and line items

## RU

Извлекать подтверждаемые данные чека и связывать их с одной оплатой.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-5.2`, `task-1.5`, `task-2.7`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Приватные вложения, журнал операций, проверка версий и gateway с контролем расходов включены в базу. Обработка чеков, общий чат и продуктовые инсайты имеют отдельные критерии приёмки.

### Требуемое поведение

Обработать фото и PDF через ограниченный pipeline с проверкой формата/размера/страниц до AI. Извлекать позиции, суммы, валюту, дату, продавца и скидки с provenance. Проверять итог арифметически, привязывать выбранный счёт, выявлять duplicate attachment и match candidates. Нерелевантное пропускать с причиной; неполное уточнять; не исполнять активный контент или URL документа. Сохранить лимиты 10 MiB/10 страниц, обрабатывать страницы с detail=high и ограниченным input/output; проверить максимум, мелкие/нечитаемые суммы, mixed receipts и injection отдельно от чистых синтетических assets/openai-eval. Inline PDF/изображения не создают долгоживущие Files objects; никакого перехода по ссылкам в документе. Исследование task-0.8 завершено: использовать gpt-5.6-terra xhigh и финальную strict-схему из evidence/openai.prompts.json; Luna/Sol/MiniMax/DeepSeek автоматически не подключать. Финальный xhigh eval 206/206 не заменяет runtime/locale проверки. reasoning.effort=xhigh; никаких автоматических downgrade при лимите $50. Статус записи формирует приложение, не объяснение модели.

### Ключевые контракты

Приватные attachments, выбранный account, receipt extraction schema, item set и correction/matching-контракты журнала; очередь уточнений и budget-controlled gateway.

### Вне объёма

Не менять согласованную модель и лимит расходов, не предоставлять модели секреты или платёжные инструменты. Пользовательские экраны и соседние сценарии реализуются своими задачами.

### Блокирующие зависимости

- [#36](https://github.com/andrey-krasheninnikov/want-keep/issues/36) (`task-5.2`): выполнена.
- [#15](https://github.com/andrey-krasheninnikov/want-keep/issues/15) (`task-1.5`): выполнена.
- [#23](https://github.com/andrey-krasheninnikov/want-keep/issues/23) (`task-2.7`): выполнена.

### Границы изменений

- `backend/internal/receipts/`
- `backend/internal/attachments/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-015:** Для отправки чека требуется счёт списания; фото/PDF остаётся связанным с результатом обработки.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-022:** Полные операции и чеки могут передаваться OpenAI, секреты доступа и лишние закрытые данные исключаются.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-015

- **Дано:** Загружено фото или PDF чека.
- **Когда:** Владелец отправляет его без счёта, затем выбирает наличный счёт.
- **Тогда:** Без выбора запись не проводится; после выбора обработка использует этот счёт. Оба аутентифицированных участника семьи читают вложение, включая чек партнёра; посторонний и участник другой семьи получают отказ.
- **Уровень:** `end-to-end`.

#### AC-016

- **Дано:** Оплачено RUB 900 за позиции RUB 600 и RUB 400 со скидкой RUB 100.
- **Когда:** AI извлекает и категоризирует позиции.
- **Тогда:** Сумма распределений точно RUB 900; скидка сохраняется; расхождение суммы направляется на уточнение, а не исправляется выдуманной позицией.
- **Уровень:** `integration`.

#### AC-017

- **Дано:** Присланы полный расход, расход без суммы и очевидно нерелевантное фото.
- **Когда:** AI обрабатывает сообщения.
- **Тогда:** Первый расход записан или связан с существующим; второй ждёт вопроса; третье сообщение получает явную причину пропуска; вымышленных сумм нет.
- **Уровень:** `end-to-end`.

#### AC-022

- **Дано:** Операция и чек доступны владельцу; рядом в системе хранятся ключи источника.
- **Когда:** Формируется запрос AI и диагностическая запись.
- **Тогда:** В запросе только разрешённые данные операции/документа; ключи и сессии отсутствуют в запросе и логах; политика хранения OpenAI раскрыта.
- **Уровень:** `contract`.

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

#### AC-068

- **Дано:** Загружаются повреждённый PDF, неверно обозначенный тип, чрезмерный файл и чек с вредоносным текстом.
- **Когда:** Срабатывают проверка файла и обработка.
- **Тогда:** Файл с ошибкой не проводится; нет выполнения вложенного кода, произвольного скачивания URL или публичного доступа; понятная причина/уточнение видна в чате.
- **Уровень:** `integration`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-091

- **Дано:** Январский чек: общие товары 600 (50/50), личные A 100, B 300; в феврале доли статьи стали 60/40.
- **Когда:** В феврале возвращаются общие товары на 200.
- **Тогда:** Январь уменьшается семье на 200, каждому на 100 с исходной исторической оценкой; февральская пропорция не меняет январь, cash date возврата остаётся февральской.
- **Уровень:** `integration`.

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-integration AREA=receipts && make eval-ai SUITE=receipts
```

Валидные, частичные, повторные и вредоносные документы проходят требуемые состояния; итог распределения равен оплате.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Extract verifiable receipt data and link it to one payment.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-5.2`, `task-1.5`, `task-2.7`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Private attachments, the transaction ledger, revision checks and the budget-controlled gateway are included in the base. Receipt processing, shared chat and product insights have separate acceptance criteria.

### Desired behavior

Process photos and PDFs through a bounded pipeline validating format/size/pages before AI. Extract items, amounts, currency, date, merchant and discounts with provenance. Validate totals arithmetically, bind the selected account and detect duplicate attachments/match candidates. Skip irrelevant content with a reason, clarify incomplete data and never execute document content or URLs. Preserve the 10 MiB/10-page limits; process pages with detail=high and bounded input/output. Test boundaries, small/unreadable amounts, mixed receipts and injection beyond the clean assets/openai-eval fixtures. Inline PDF/images create no persistent Files objects; never follow document links. task-0.8 research is complete: use gpt-5.6-terra xhigh and the final strict schema in evidence/openai.prompts.json; do not automatically enable Luna/Sol/MiniMax/DeepSeek. The final xhigh evaluation 206/206 does not replace runtime/locale checks. reasoning.effort=xhigh; no automatic downgrade at the USD 50 cap. The application supplies persistence status, not the model explanation.

### Key interfaces

Private attachments, selected account, receipt extraction schema, ledger item-set and correction/matching contracts; clarification queue and budget-controlled gateway.

### Out of scope

Do not change the agreed model or spending cap, or expose secrets or payment tools to the model. Product screens and adjacent workflows remain with their owning tasks.

### Blocked by

- [#36](https://github.com/andrey-krasheninnikov/want-keep/issues/36) (`task-5.2`): completed.
- [#15](https://github.com/andrey-krasheninnikov/want-keep/issues/15) (`task-1.5`): completed.
- [#23](https://github.com/andrey-krasheninnikov/want-keep/issues/23) (`task-2.7`): completed.

### Change boundaries

- `backend/internal/receipts/`
- `backend/internal/attachments/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-015:** Receipt submission requires a debit account; the photo/PDF stays linked to the processing result.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-022:** Full transactions and receipts may be sent to OpenAI; access secrets and unrelated private data are excluded.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-015

- **Given:** A receipt photo or PDF is uploaded.
- **When:** The owner submits it without an account, then selects cash.
- **Then:** No entry is posted without selection; processing then uses the selected account. Both authenticated household members can read the attachment, including a partner’s receipt; unauthenticated and foreign-household access is denied.
- **Level:** `end-to-end`.

#### AC-016

- **Given:** RUB 900 was paid for RUB 600 and RUB 400 items with a RUB 100 discount.
- **When:** AI extracts and categorizes the items.
- **Then:** Allocations total exactly RUB 900 and retain the discount; a mismatch is clarified rather than patched with an invented item.
- **Level:** `integration`.

#### AC-017

- **Given:** A complete expense, an expense missing its amount and an obviously irrelevant image are submitted.
- **When:** AI processes the messages.
- **Then:** The first expense is recorded or matched; the second awaits clarification; the third receives an explicit skip reason; no amounts are invented.
- **Level:** `end-to-end`.

#### AC-022

- **Given:** A transaction and receipt are available to the owner; source credentials are stored elsewhere.
- **When:** An AI request and diagnostic record are produced.
- **Then:** The request contains only permitted transaction/document data; keys and sessions appear in neither request nor logs; OpenAI retention policy is disclosed.
- **Level:** `contract`.

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

#### AC-068

- **Given:** A corrupt PDF, mislabeled type, oversized file and prompt-injected receipt are uploaded.
- **When:** File validation and processing run.
- **Then:** Invalid files do not post; embedded code, arbitrary URL fetching and public access are unavailable; chat shows an understandable reason or clarification.
- **Level:** `integration`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-091

- **Given:** January receipt: joint items 600 (50/50), personal A 100, B 300; February plan shares became 60/40.
- **When:** Joint items worth 200 are returned in February.
- **Then:** January falls by 200 for the household and 100 for each member using original historical valuation; February shares do not rewrite January and refund cash date stays in February.
- **Level:** `integration`.

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

### Verification

```sh
make test-integration AREA=receipts && make eval-ai SUITE=receipts
```

Valid, partial, repeated and malicious documents follow required states; allocation total equals payment.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
