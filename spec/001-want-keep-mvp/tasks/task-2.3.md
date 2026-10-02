<!-- want-keep-task: task-2.3 -->
# task-2.3 — Добавить версии, исправления и аудит / Add revisions, corrections and audit

## RU

Исправлять семейные финансовые факты, сохранять пользовательские поля и объяснять каждое решение.

**Состояние:** Реализованы backend/API исправлений, отмены, исключения, истории и сохраняемый контракт review. Сопоставление, категории, доли, возвраты, OpenAI и экраны остаются профильным задачам.

**Зависимости:** `task-2.2`.

**Тип:** `implementation`.

### Изменение и контракты

Решения исправляют principal согласованным комплектом, отдельную группу комиссий, дату, плательщика, продавца и комментарий. Поля защищаются независимо от банковского lifecycle; raw source и эффективная revision разделены. Undo по decisionId и версиям всех участников сохраняет поздние независимые изменения, отвергает пересечения и A → B → A. included/excluded независимо от posted/reversed/cancelled; восстановление использует актуальный источник. Решение, проекции, audit/outbox, один review-запрос на revision и исход команды атомарны. История API содержит до/после, доказательства, исходные значения и причины невозможности отмены. Результат review проверяет актуальную версию/права/защиту без вызова OpenAI и без цикла событий.

### Границы изменений

- `backend/internal/ledger/`
- `backend/internal/accounts/`
- `backend/internal/storage/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/009_transaction_corrections_audit.sql`
- `api/`
- `backend/test/integration/audit/`

### Экранный контракт

### SCR-010 — Карточка операции

`/transactions/:id`

**Вопрос:** Правильно ли учтена эта покупка?

**Главный ответ:** Сумма, категория, продавец, позиции, назначение и плательщик одной операции.

**Структура сверху вниз:** Результат учёта → счёт/дата/статус → category/merchant → позиции gross/discount/net → доли → чек → исправить/отменить/возврат.

**Следующее действие:** Исправить FORM-06/07, вернуть FORM-08, явный долг FORM-09; чек → SCR-011.

**Объяснение и детализация:** История до/после показывает автора, decisionId, источник и защищённые поля. Распределение показывает personal/shared, точные суммы каждого участника, unallocated, применённые rule revisions и позиции. Явное значение позиции важнее покупки, затем merchant/category rule и equal для явно совместной траты. Source update не стирает пользовательский выбор; изменение amount-based распределения требует согласованной правки, share-based пересчитывается. Matching сохраняет один носитель семейного и персонального эффекта; undo проверяет revisions всех участников.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-06, FORM-07, FORM-08, FORM-09, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-05 — Учесть перевод или обмен

**Поля:** Откуда/куда, даты, обе суммы/валюты, комиссии и счёт комиссии, существующие движения.

**Проверки и права:** Оба участника; разные счета семьи; одна исходящая и одна входящая сторона. Principal не входит в доходы/расходы; отдельные комиссии, включая третий актив. Пустой existingTransactions создаёт новое движение. Непустой список содержит ID/revisions всех участников (до 100); суммы, комиссии и дата основной записи проверяются без создания недостающих сторон. Связь переводов/обменов/оплаты подтверждается по версиям, неоднозначность остаётся в matching; отдельная покупка снимает ожидание один раз.

**Результат:** Связано движение денег в журнале; никакой реальной отправки или покупки актива.

#### FORM-06 — Исправление, сопоставление и отмена

**Поля:** Операция, expectedRevision, основание; полный principal и fees, дата, payer, raw merchant/note; category и merchant identity через set|clear, позиции через replace|clear. Undo: decisionId и expectedRevisions; сравнение до/после и с источником.

**Проверки и права:** Оба участника исправляют семейные факты. Сервер сохраняет actor, счета/активы principal и происхождение, проверяет активные категории/продавцов и меняет полный item set атомарно. Позиции и скидки точно равны principal; top-level category с позициями запрещена. Undo сохраняет поздние независимые поля. Сопоставление и доли остаются профильным задачам.

**Результат:** Новое решение и финансовые revisions с историей, либо no_change/conflict без эффекта и потери ввода. Исключение не меняет банковский статус; undo пересчитывает текущий эффект.

#### FORM-07 — Чек и распределение позиций

**Поля:** Фото/PDF, обязательный счёт списания включая наличные; позиции, скидки, категории, personal/shared и доли % или суммы.

**Проверки и права:** Оба участника; лимиты файлов по контракту. Task-2.6 атомарно проверяет позиции, активы и скидки против одной оплаты, распределяет известную общую скидку детерминированно и требует уточнение при неполных данных. OCR/PDF, сопоставление и personal/shared доли выполняют task-5.3/2.4/2.8.

**Результат:** Создано/связано с существующим/ожидает уточнения/документ не подходит с причиной. Одно подтверждённое списание.

#### FORM-08 — Возврат покупки

**Поля:** Исходная покупка, возвращаемые позиции/доли/сумма, счёт поступления и фактическая дата.

**Проверки и права:** Оба member; совокупный возврат не больше покупки; исходные исторические FX и распределение по возвращённой части сохраняются.

**Результат:** Исходный месяц покупки пересчитан; деньги поступили текущей датой; FX отдельно.

#### FORM-09 — Явный долг и возмещение

**Поля:** Кредитор и должник как участники семьи, точная сумма/валюта, основание и необязательный расход. Погашение выбирает существующий перевод и задаёт его revision, transferAmount и settledAmount.

**Проверки и права:** Долг создаётся только явно. Участники различны и активны. Перевод идёт между их личными счетами; комиссия не погашает долг. Один principal не используется сверх остатка. Для разных активов обе суммы обязательны. Изменённые ссылки показывают attention_required или stale вместо скрытого списания.

**Результат:** Долг создан, исправлен, погашен или восстановлен с новой revision и историей; семейный денежный факт не дублируется.

- **UISTATE-01 — Загрузка:** Скелетон структуры и подпись загрузки; суммы не подменяются нулями.
- **UISTATE-02 — Обновление:** Сохранить предыдущие данные и контекст, показать время последнего успеха; блокировать только конфликтующие действия.
- **UISTATE-03 — Пусто:** Объяснить полезный результат и предложить первое действие: счёт, чек, план или цель.
- **UISTATE-05 — Частичные данные:** Назвать отсутствующий источник/период и последствия для суммы; доступные блоки работают; неизвестное обозначить отдельно.
- **UISTATE-06 — Устаревшие данные:** Показать дату последнего успеха и влияние на решение; дать обновить или перейти к подключению.
- **UISTATE-07 — Ошибка:** Понятная причина и следующий шаг у проблемного блока; ввод и исправные данные сохранить, диагностику раскрывать отдельно.
- **UISTATE-08 — Offline:** Показать отсутствие связи; не обещать сохранение. Чувствительные черновики только в памяти текущей вкладки, без новой offline-очереди.
- **UISTATE-09 — Сохранение:** Немедленно показать прогресс текущего действия и не допускать дублирующую отправку команды.
- **UISTATE-10 — Исход неизвестен:** Сохранить ID команды/ввод, запросить её результат; не создавать новую финансовую команду вслепую. После перезагрузки сверять серверный список недавних команд.
- **UISTATE-11 — Конфликт версии:** Показать авторов и различия, сохранить мой ввод; загрузить актуальную версию и дать повторно применить выбранные изменения после проверки.
- **UISTATE-12 — Недостаточно прав:** Финансовые данные доступны семье; запрещённое изменение объясняет владельца. Сервер отклоняет команду независимо от видимости кнопки.
- **UISTATE-13 — Сессия истекла:** Закрыть защищённое содержимое; вход для того же участника, безопасный возврат по внутреннему маршруту. Чужой вход не получает прежний черновик.
- **UISTATE-14 — Ожидание AI:** Отличать очередь, обработку, уточнение и паузу из-за лимита/API; обычный учёт доступен, результат не выдумывать.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-012:** Исправление учёта сохраняет оригинал, автора, основание, версию и возможность отмены решения.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-018:** Каждая новая или содержательно изменённая операция получает AI-проверку своей версии.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-012

- **Дано:** AI ошибочно связал две операции; исходные импортированные записи сохранены.
- **Когда:** Владелец отменяет связь и исправляет категорию.
- **Тогда:** Пересчитаны производные отчёты; видна история; повторный импорт не стирает правку владельца.
- **Уровень:** `integration`.

#### AC-018

- **Дано:** Есть импортированная, ручная и созданная чатом операции.
- **Когда:** Они создаются, а затем одна финансовая запись исправляется.
- **Тогда:** Каждая актуальная версия поставлена на проверку; повторная доставка задачи не дублирует эффект; устаревший ответ AI не меняет новую версию.
- **Уровень:** `integration`.

#### AC-019

- **Дано:** AI предлагает сумму, противоречащую источнику, и связь с несколькими кандидатами.
- **Когда:** Приложение проверяет предложения.
- **Тогда:** Противоречивое изменение отклонено, неоднозначность поступает в очередь уточнений; категории и подтверждённые связи могут применяться автоматически.
- **Уровень:** `integration`.

#### AC-021

- **Дано:** Есть утверждённый бюджет и предложение перераспределения.
- **Когда:** Приходит новый расход, затем уполномоченный участник подтверждает предложенное изменение.
- **Тогда:** До подтверждения план неизменен; подтверждение применяет показанную версию предложения один раз; устаревшее предложение пересогласуется.
- **Уровень:** `integration`.

#### AC-061

- **Дано:** Процесс падает между сохранением записи и подтверждением задания.
- **Когда:** Задание повторяется, одновременно приходит правка владельца.
- **Тогда:** Применён один эффект, правка защищена версией, незавершённое состояние восстанавливается; внешняя неоднозначность не вызывает слепой повтор.
- **Уровень:** `integration`.

#### AC-064

- **Дано:** На одном счёте две покупки RUB 300 близко по времени; владелец исправил одну.
- **Когда:** AI получает чек без уникального идентификатора и старый результат классификации.
- **Тогда:** Запрашивается выбор совпадения; покупки не сливаются по вероятности; правка человека не теряется.
- **Уровень:** `integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-go PKG=./internal/ledger/... && make test-integration AREA=audit && make test-audit-race
```

Исправления шести активов, выборочная и составная отмена, исключение, merge источника, review, ABA, concurrency, replay/rollback/restart, история, права, миграция и retention проходят без повторного эффекта.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Correct household financial facts, preserve user-selected fields and explain every decision.

**Status:** Correction, undo, exclusion and history backend/API and a persisted review contract are implemented. Matching, categories, shares, refunds, OpenAI and screens remain with their owning tasks.

**Dependencies:** `task-2.2`.

**Kind:** `implementation`.

### Change and contracts

Decisions correct the complete principal group, separate fees, purchase time, payer, merchant and note. Fields are protected independently from bank lifecycle; normalized source and effective revision are separate. Undo by decisionId and all current participant revisions retains later independent edits and rejects overlaps and A → B → A. included/excluded is independent from posted/reversed/cancelled; restoration uses current source facts. Decision, projections, audit/outbox, one review request per revision and command outcome are atomic. History API exposes before/after, evidence, source values and undo rejection reasons. Review results check current version/rights/protections without calling OpenAI or causing an event loop.

### Change boundaries

- `backend/internal/ledger/`
- `backend/internal/accounts/`
- `backend/internal/storage/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/009_transaction_corrections_audit.sql`
- `api/`
- `backend/test/integration/audit/`

### Screen contract

### SCR-010 — Transaction details

`/transactions/:id`

**Question:** Is this purchase accounted for correctly?

**Primary answer:** Amount, category, merchant, items, allocation and payer for one transaction.

**Top-down structure:** Accounting outcome → account/date/status → category/merchant → item gross/discount/net → shares → receipt → correct/undo/refund.

**Next action:** Correct FORM-06/07, refund FORM-08, explicit debt FORM-09; receipt → SCR-011.

**Explanation and details:** Before/after history exposes actor, decisionId, source and protected fields. Allocation shows personal/shared purpose, exact member amounts, unallocated, applied rule revisions and items. Explicit item value wins over purchase, then merchant/category rule and equal for an explicitly shared expense. A source update cannot erase the user choice; changing an amount-based allocation requires a consistent correction while share-based allocation recalculates. Matching retains one household and member effect carrier; undo checks every participant revision.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-06, FORM-07, FORM-08, FORM-09, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-05 — Record transfer or exchange

**Fields:** From/to accounts, dates, both amounts/currencies, fees/fee account, existing movements.

**Validation and permissions:** Either member; distinct household accounts; one outgoing and one incoming side. Principal is excluded from income/expenses; separate fees may use a third asset. Empty existingTransactions creates new movement. A nonempty list supplies all participant IDs/revisions (up to 100); amounts, fees and primary date are checked without creating missing sides. Transfer/exchange/payment links require version checks; ambiguity remains in matching; separate-purchase confirmation releases waiting once.

**Outcome:** Ledger movements linked; no actual transfer or asset purchase.

#### FORM-06 — Correction, matching and undo

**Fields:** Transaction, expectedRevision and reason; complete principal and fees, date, payer, raw merchant/note; category and merchant identity via set|clear, items via replace|clear. Undo uses decisionId and expectedRevisions; compare before/after and source.

**Validation and permissions:** Both members correct household facts. The server retains actor, principal accounts/assets and provenance, validates active categories/merchants and replaces the complete item set atomically. Items and discounts equal principal exactly; top-level category with items is forbidden. Undo preserves later independent fields. Matching and shares remain with their owning tasks.

**Outcome:** New decision and financial revisions with history, or no_change/conflict without effect or lost input. Exclusion does not change bank state; undo recomputes the current effect.

#### FORM-07 — Receipt and item allocation

**Fields:** Photo/PDF, required debit account including cash; items, discounts, categories, personal/shared and percentage or amount shares.

**Validation and permissions:** Either member; file limits follow the contract. Task-2.6 atomically validates items, assets and discounts against one payment, allocates a known receipt-wide discount deterministically and requires clarification for incomplete data. Task-5.3/2.4/2.8 own OCR/PDF, matching and personal/shared shares.

**Outcome:** Created/linked to existing/awaiting clarification/document unsuitable with reason. One confirmed debit.

#### FORM-08 — Purchase refund

**Fields:** Original purchase, returned items/shares/amount, receiving account and actual date.

**Validation and permissions:** Either member; cumulative refund cannot exceed purchase; original historical FX and refunded-part allocation are retained.

**Outcome:** Original purchase month recalculated; cash arrives on actual date; FX separate.

#### FORM-09 — Explicit debt and reimbursement

**Fields:** Creditor and debtor as household members, exact amount/currency, reason and optional expense. Settlement selects an existing transfer and supplies its revision, transferAmount and settledAmount.

**Validation and permissions:** Debt is created explicitly only. Members are distinct and active. Transfer runs between their personal accounts; fees do not settle debt. Principal cannot be reused beyond its remainder. Cross-asset settlement requires both amounts. Changed references surface attention_required or stale instead of silently reducing debt.

**Outcome:** The debt is created, corrected, settled or restored with a new revision and history; the household cash fact is not duplicated.

- **UISTATE-01 — Loading:** Structural skeleton and loading label; amounts are never replaced by zero.
- **UISTATE-02 — Refreshing:** Keep previous data/context and last-success time; block only conflicting actions.
- **UISTATE-03 — Empty:** Explain the useful outcome and offer a first account, receipt, plan or goal action.
- **UISTATE-05 — Partial data:** Name the missing source/period and its effect on the amount; available sections work and unknowns stay explicit.
- **UISTATE-06 — Stale data:** Show last-success date and impact on the decision; offer refresh or connection details.
- **UISTATE-07 — Error:** Plain cause and next step beside the affected section; preserve input/healthy data and expand diagnostics separately.
- **UISTATE-08 — Offline:** Show missing connectivity and do not promise saved data. Sensitive drafts remain only in current-tab memory, without a new offline queue.
- **UISTATE-09 — Saving:** Immediately show current-action progress and prevent duplicate command submission.
- **UISTATE-10 — Unknown outcome:** Keep command ID/input and query its result; never blindly create another financial command. After reload reconcile the server list of recent commands.
- **UISTATE-11 — Version conflict:** Show authors/differences and keep my input; load current version and allow chosen changes to be reapplied after validation.
- **UISTATE-12 — Insufficient permission:** Household can read financial data; forbidden edits explain ownership. Server rejects the command regardless of button visibility.
- **UISTATE-13 — Session expired:** Hide protected contents; require the same member to sign in and return through a safe internal route. Another identity never receives the prior draft.
- **UISTATE-14 — AI waiting:** Distinguish queued, processing, clarification and budget/API pause; ordinary accounting remains available and results are not invented.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-012:** Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-018:** Every new or materially changed transaction receives AI review of its version.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-012

- **Given:** AI incorrectly linked two transactions; original imports are retained.
- **When:** The owner unlinks them and corrects the category.
- **Then:** Derived reports are recalculated, history is visible and reimport does not overwrite the owner's correction.
- **Level:** `integration`.

#### AC-018

- **Given:** Imported, manual and chat-created transactions exist.
- **When:** They are created and one financial record is then corrected.
- **Then:** Every current version is queued for review; repeated job delivery does not duplicate effects; a stale AI response cannot alter a newer version.
- **Level:** `integration`.

#### AC-019

- **Given:** AI proposes a source-conflicting amount and a match with several candidates.
- **When:** The application validates the proposals.
- **Then:** The conflicting change is rejected and ambiguity enters the clarification queue; categories and substantiated links may be applied automatically.
- **Level:** `integration`.

#### AC-021

- **Given:** An approved budget and a reallocation proposal exist.
- **When:** A new expense arrives and an authorized member later confirms the proposal.
- **Then:** The plan stays unchanged until confirmation; confirmation applies the displayed proposal version once; a stale proposal must be reconfirmed.
- **Level:** `integration`.

#### AC-061

- **Given:** A process crashes between persisting a record and acknowledging its job.
- **When:** The job is retried while the owner submits a correction.
- **Then:** One effect is applied, the correction is version-protected and incomplete state recovers; an ambiguous external outcome is not blindly retried.
- **Level:** `integration`.

#### AC-064

- **Given:** One account has two RUB 300 purchases close in time; the owner corrected one.
- **When:** AI receives a receipt without a unique identifier and a stale classification result.
- **Then:** A match selection is requested; purchases are not merged merely by likelihood and the owner's correction survives.
- **Level:** `integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

### Verification

```sh
make test-go PKG=./internal/ledger/... && make test-integration AREA=audit && make test-audit-race
```

Six-asset corrections, selective/compound undo, exclusion, source merge, review, ABA, concurrency, replay/rollback/restart, history, permissions, migration and retention pass without duplicate effects.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
