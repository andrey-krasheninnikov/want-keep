<!-- want-keep-task: task-2.6 -->
# task-2.6 — Разделить категории, продавцов и товары / Separate categories, merchants and items

## RU

Раздельно классифицировать семейные расходы по категории, продавцу и позициям без повторного финансового эффекта.

**Состояние:** Реализованы backend/API каталога категорий и продавцов, классификация операций и позиции чеков. UI, OCR/PDF, реальный OpenAI, отчёты и распределение по участникам остаются профильным задачам.

**Зависимости:** `task-2.3`.

**Тип:** `implementation`.

### Изменение и контракты

Семейный каталог содержит двухуровневые starter/custom категории со стабильными RU/EN-ключами, revisions и архивом, а также отдельных продавцов и подтверждённые уникальные алиасы. Ledger версионирует category, merchant identity и атомарный item set с точными gross/discount/net, защитой полей и selective undo. Общая скидка распределяется largest remainder по стабильному item ID; позиции равны principal и не удваивают расход. Review сохраняет предложение, но не применяет его и не меняет каталог. API даёт guarded-команды, историю и repeatable-read фильтры.

### Границы изменений

- `backend/internal/categories/`
- `backend/internal/ledger/`
- `backend/internal/storage/`
- `backend/internal/delivery/categories/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/012_categories_merchants_items.sql`
- `api/`
- `backend/test/integration/categories/`

### Экранный контракт

### SCR-009 — Операции

`/transactions`

**Вопрос:** На что потрачено и всё ли учтено?

**Главный ответ:** Назначение операций с понятным итогом выбранного периода.

**Структура сверху вниз:** Поиск/месяц/счёт/плательщик/назначение/category/merchant/item/status → итоги → строки дата/продавец/сумма/доли.

**Следующее действие:** Открыть SCR-010; занести FORM-04 или чек в SCR-024.

**Объяснение и детализация:** Фильтры category, merchant и item независимы; некатегоризированный расход остаётся видимым. Позиции образуют одну оплату без повторного итога. Личные суммы и unallocated являются разрезом одного семейного факта по MembershipID; payer, owner и actor показаны отдельно. Переводы/обмены помечены как движения, исключённые из доходов/расходов; фильтр не меняет расчётную семантику.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-04, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

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

### SCR-011 — Чек

`/receipts/:id`

**Вопрос:** Что распознано и к какой покупке относится?

**Главный ответ:** Связь с одной операцией либо конкретный вопрос.

**Структура сверху вниз:** Результат создано/найдено/вопрос → счёт → оригинал рядом с позициями → суммы/доли.

**Следующее действие:** Уточнить/распределить FORM-07/12; открыть SCR-010.

**Объяснение и детализация:** Gross, discount и net позиции объясняют одну оплату; общая скидка распределена детерминированно. Неполные скидки или расхождение требуют уточнения, без выдуманной позиции. Недостоверные поля выделяются, неподходящий документ объясняется, небезопасное содержимое не исполняется. OCR/PDF остаётся task-5.3.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-07, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

### SCR-034 — Категории и правила

`/settings/categories`

**Вопрос:** Как уменьшить ручные уточнения?

**Главный ответ:** Понятные категории, продавцы и правила назначения расходов.

**Структура сверху вниз:** Категории/подкатегории и архив → продавцы/подтверждённые алиасы → правила/приоритет → preview примеров.

**Следующее действие:** Создать/исправить FORM-15; проверить затронутые операции SCR-009.

**Объяснение и детализация:** Starter labels имеют стабильные RU/EN-ключи, custom name не переводится. Продавец не становится подкатегорией; архив сохраняет историю. Правила используют merchant/category AND, priority и доли активных участников; preview объясняет выбранные revisions или rule_conflict. Новое правило действует только на новые факты. AI proposal требует пользовательского подтверждения; правило personal/shared не меняет чужой личный план.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-15.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

#### FORM-04 — Доход или расход

**Поля:** Тип, счёт, дата/время, сумма/валюта, категория/подкатегория, продавец, назначение и доли, комментарий/чек.

**Проверки и права:** Оба участника заносят факт на любой счёт семьи; actor из сессии, payer отдельно. Сумма >0, актив совпадает со счётом. Расход может содержать активные category/merchant и typed allocation по текущим MembershipID; неизвестное назначение остаётся unallocated. Доход не принимает allocation. Подтверждённые счёт/сумма/дата дают posted и один семейный факт даже без классификации.

**Результат:** Одна операция, видимые назначения и AI-статус, связь чека; подтверждённый результат и ссылка.

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

#### FORM-12 — Ответ и предложение AI

**Поля:** Ответ на конкретное уточнение или явное решение по предложенным изменениям; версия объекта и вопроса.

**Проверки и права:** Оба для операций, только владелец для личного плана/цели; actor не меняется текстом. Одновременный ответ проверяет версию; preview перед финансовым изменением.

**Результат:** Команда подтверждена, отказана, устарела или ожидает; ответ/основание и автор сохранены.

#### FORM-15 — Категории и правила

**Поля:** Название категории/подкатегории, родитель и состояние; имя продавца, состояние и подтверждённые алиасы; условия правила merchant/category, priority 1–1000, состояние и точные доли по участникам; preview без сохранения.

**Проверки и права:** Оба участника управляют семейным каталогом и правилами. Условия одного правила объединяются AND; меньшее priority важнее. Одинаковые результаты равного приоритета совместимы, разные дают rule_conflict. Доли дают ровно 100% активных участников; expectedRevision, CSRF и actor из сессии обязательны. Правило применяется только к новым фактам и не меняет историю.

**Результат:** Версионированная категория, продавец или правило сохранены; preview показывает применённые revisions либо безопасную unresolved-причину. no_change/conflict не создают эффекта, исторические ссылки остаются доступны.

- **UISTATE-01 — Загрузка:** Скелетон структуры и подпись загрузки; суммы не подменяются нулями.
- **UISTATE-02 — Обновление:** Сохранить предыдущие данные и контекст, показать время последнего успеха; блокировать только конфликтующие действия.
- **UISTATE-03 — Пусто:** Объяснить полезный результат и предложить первое действие: счёт, чек, план или цель.
- **UISTATE-04 — Нет совпадений:** Сохранить фильтры, объяснить отсутствие результатов, предложить очистить условия.
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

- **REQ-014:** Категория, подкатегория, продавец и позиция чека являются отдельными аналитическими признаками.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-014

- **Дано:** Чек содержит молоко, продавец — условный магазин; другая покупка — ресторан.
- **Когда:** Владелец фильтрует расходы и исправляет категорию.
- **Тогда:** Доступны независимые срезы по виду расхода, продавцу и товару; пользовательская правка сохраняется.
- **Уровень:** `end-to-end`.

#### AC-016

- **Дано:** Оплачено RUB 900 за позиции RUB 600 и RUB 400 со скидкой RUB 100.
- **Когда:** AI извлекает и категоризирует позиции.
- **Тогда:** Сумма распределений точно RUB 900; скидка сохраняется; расхождение суммы направляется на уточнение, а не исправляется выдуманной позицией.
- **Уровень:** `integration`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

### Проверка результата

```sh
make test-go PKG=./internal/categories/... && make test-integration AREA=categories && make test-categories-race
```

Стартовый каталог, иерархия, продавцы/алиасы, шесть активов, точные позиции/скидки, correction/undo, review proposal, права, replay, миграция, фильтры и пагинация проходят без частичной записи и повторного расхода.

Зависимость task-2.3 включена в базу; команды существуют. Доказательства и границы: evidence/task-2.6-classification.md. Обязательны make check, categories/audit/ledger/accounts/storage/identity/household integration/race и privacy suite. Это не подтверждает эксплуатационную готовность.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Classify household expenses independently by category, merchant and receipt items without duplicate financial effects.

**Status:** Category and merchant catalog backend/API, transaction classification and receipt items are implemented. UI, OCR/PDF, real OpenAI, reports and member allocation remain with their owning tasks.

**Dependencies:** `task-2.3`.

**Kind:** `implementation`.

### Change and contracts

The household catalog contains two-level starter/custom categories with stable RU/EN keys, revisions and archival, plus separate merchants and unique confirmed aliases. Ledger versions category, merchant identity and an atomic item set with exact gross/discount/net, field protection and selective undo. A receipt-wide discount uses largest remainder with stable item ID; items equal principal and never duplicate expense. Review stores a proposal without applying it or mutating the catalog. The API provides guarded commands, history and repeatable-read filters.

### Change boundaries

- `backend/internal/categories/`
- `backend/internal/ledger/`
- `backend/internal/storage/`
- `backend/internal/delivery/categories/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/012_categories_merchants_items.sql`
- `api/`
- `backend/test/integration/categories/`

### Screen contract

### SCR-009 — Transactions

`/transactions`

**Question:** Where did money go and is everything recorded?

**Primary answer:** Transaction purpose and a meaningful selected-period total.

**Top-down structure:** Search/month/account/payer/allocation/category/merchant/item/status → totals → date/merchant/amount/share rows.

**Next action:** Open SCR-010; enter FORM-04 or receipt in SCR-024.

**Explanation and details:** Category, merchant and item filters are independent; uncategorized expense remains visible. Items form one payment without a duplicate total. Member amounts and unallocated are a MembershipID view of one household fact; payer, owner and actor are separate. Transfers/exchanges are marked as movements excluded from income/expense; filters never change accounting semantics.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-04, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

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

### SCR-011 — Receipt

`/receipts/:id`

**Question:** What was extracted and which purchase does it belong to?

**Primary answer:** One transaction link or a specific clarification.

**Top-down structure:** Created/found/question outcome → account → original beside items → amounts/shares.

**Next action:** Clarify/allocate FORM-07/12; open SCR-010.

**Explanation and details:** Item gross, discount and net explain one payment; a receipt-wide discount is allocated deterministically. Incomplete discounts or a mismatch require clarification without an invented item. Uncertain fields are labelled, unsuitable documents are explained and unsafe content is never executed. OCR/PDF remains task-5.3.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-07, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

### SCR-034 — Categories and rules

`/settings/categories`

**Question:** How do we reduce manual clarifications?

**Primary answer:** Clear categories, merchants and expense-allocation rules.

**Top-down structure:** Categories/subcategories and archive → merchants/confirmed aliases → rules/priority → example preview.

**Next action:** Create/correct FORM-15; inspect affected transactions SCR-009.

**Explanation and details:** Starter labels use stable RU/EN keys and a custom name is not translated. A merchant never becomes a subcategory; archival preserves history. Rules use merchant/category AND, priority and active-member shares; preview explains selected revisions or rule_conflict. A new rule affects only new facts. An AI proposal requires user confirmation; a personal/shared rule never edits a partner personal plan.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-15.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

#### FORM-04 — Income or expense

**Fields:** Type, account, date/time, amount/currency, category/subcategory, merchant, purpose/shares, note/receipt.

**Validation and permissions:** Either member records on any household account; actor comes from session and payer is independent. Amount >0 and asset matches the account. An expense may contain active category/merchant references and typed allocation by current MembershipID; unknown purpose remains unallocated. Income rejects allocation. Confirmed account/amount/date produce posted and one household fact even without classification.

**Outcome:** One transaction, visible allocation and AI status, linked receipt; confirmed outcome and link.

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

#### FORM-12 — AI response and proposal

**Fields:** Answer to a specific clarification or explicit decision on proposed changes; object/question revision.

**Validation and permissions:** Either member for transactions, owner only for personal plan/goal; text cannot change actor. Concurrent response checks revision; preview before financial change.

**Outcome:** Command confirmed, rejected, stale or pending; response/reason and author retained.

#### FORM-15 — Categories and rules

**Fields:** Category/subcategory name, parent and state; merchant name, state and confirmed aliases; merchant/category rule conditions, priority 1–1000, state and exact member shares; preview without persistence.

**Validation and permissions:** Either member manages the household catalog and rules. Conditions within one rule use AND and lower priority wins. Equal-priority identical outcomes are compatible; different outcomes yield rule_conflict. Shares total exactly 100% across active members; expectedRevision, CSRF and the session actor are mandatory. A rule applies only to new facts and never rewrites history.

**Outcome:** A versioned category, merchant or rule is stored; preview exposes applied revisions or a safe unresolved reason. no_change/conflict creates no effect and historical references remain available.

- **UISTATE-01 — Loading:** Structural skeleton and loading label; amounts are never replaced by zero.
- **UISTATE-02 — Refreshing:** Keep previous data/context and last-success time; block only conflicting actions.
- **UISTATE-03 — Empty:** Explain the useful outcome and offer a first account, receipt, plan or goal action.
- **UISTATE-04 — No matches:** Keep filters, explain no results and offer to clear conditions.
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

- **REQ-014:** Category, subcategory, merchant and receipt item are separate analytical dimensions.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-014

- **Given:** A receipt contains milk from a fictional store; another purchase is from a restaurant.
- **When:** The owner filters expenses and corrects a category.
- **Then:** Expense type, merchant and item can be analyzed independently; owner corrections persist.
- **Level:** `end-to-end`.

#### AC-016

- **Given:** RUB 900 was paid for RUB 600 and RUB 400 items with a RUB 100 discount.
- **When:** AI extracts and categorizes the items.
- **Then:** Allocations total exactly RUB 900 and retain the discount; a mismatch is clarified rather than patched with an invented item.
- **Level:** `integration`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

### Verification

```sh
make test-go PKG=./internal/categories/... && make test-integration AREA=categories && make test-categories-race
```

Starter catalog, hierarchy, merchants/aliases, six assets, exact items/discounts, correction/undo, review proposal, permissions, replay, migration, filters and pagination pass without partial writes or duplicate expense.

Dependency task-2.3 is included in the base; commands exist. Evidence and boundaries: evidence/task-2.6-classification.en.md. Require make check, categories/audit/ledger/accounts/storage/identity/household integration/race and privacy suite. This does not confirm operational readiness.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
