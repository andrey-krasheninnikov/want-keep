<!-- want-keep-task: task-2.8 -->
# task-2.8 — Распределять семейные расходы и позиции по участникам / Allocate household expenses and items to members

## RU

Распределять семейные расходы и позиции по участникам.

**Состояние:** Реализованы backend/API семейного распределения операций и позиций, версионные правила и точная персональная аналитика без повторного финансового эффекта. UI, возвраты, бюджеты и реальный OpenAI остаются профильным задачам.

**Зависимости:** `task-2.6`, `task-1.6`.

**Тип:** `implementation`.

### Изменение и контракты

Ledger хранит неизменяемые снимки распределения по MembershipID отдельно от payer, owner и actor; allocation/domain выбирает версионные merchant/category rules. Поддержаны точные суммы без потери исходной decimal-точности, доли и equal с largest-remainder и MembershipID tie-break, personal/shared, item override, сохраняемое основание transaction fallback и точный basis rule-derived позиции для перерасчёта. Нулевая net-позиция становится not_applicable, сохраняя basis для возможного последующего перерасчёта. Неизвестное назначение остаётся unallocated; равные конфликтующие правила требуют решения. Первая ledger revision фиксирует монотонную границу семейной последовательности правил; поздняя классификация не видит rules/revisions после неё даже при одинаковом или регрессировавшем времени. Исправление классификации повторно разрешает незащищённые части, сохраняя явные purchase/item inputs; заменяет прежнее правило либо очищает устаревшие доли. Семантически равные decimal-значения с разным scale не создают revision. Source import применяет merchant rule только через подтверждённый alias, source update не стирает пользовательское распределение, а matching сохраняет один семейный и один персональный эффект с учётом carrier/contribution; waiting хранит и обновляет basis без member-effect, separate восстанавливает его. Правила для fallback и всех уникальных условий читаются одним пакетом; allocation snapshots всей страницы операций восстанавливаются фиксированным числом set-based SQL-запросов.

### Границы изменений

- `backend/internal/allocation/`
- `backend/internal/ledger/`
- `backend/internal/storage/`
- `backend/internal/delivery/allocation/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/017_family_allocations.sql`
- `api/`
- `backend/test/integration/family-allocation/`

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

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-065

- **Дано:** Покупка USD 10 распределена по двум категориям и оценена в RUB 900.
- **Когда:** Позже возвращено USD 4 за известную позицию; текущий курс иной.
- **Тогда:** Историческая категория уменьшается на исходную стоимость возвращённой части RUB 360; реальные поступления и валютная разница сохраняются отдельно; превышение суммы возвратов блокируется.
- **Уровень:** `unit+integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-080

- **Дано:** Зарплата поступила на счёт A, общая аренда оплачена B, у A нет доступного остатка.
- **Когда:** Строятся семейный бюджет, персональные расходы и обеспеченность по валютам.
- **Тогда:** Доход общий с сохранением получателя; аренда учтена в семье один раз и в личных видах по долям. Доступность семьи включает средства обоих без автоматического обмена валют и без кредитного лимита.
- **Уровень:** `integration`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
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
make check
make test-integration AREA=all
make test-family-allocation-race
make test-integration AREA=privacy
git diff --check
```

Смешанный чек 1000 даёт семье 1000, A 400 и B 600; шесть активов и произвольная точность сохраняются, включая явное 0.5/0.5 при сумме 1. Проверяются перерасчёт share/equal, неизменность amount-based, нулевая net-позиция, excluded-расход при migration-over-015, безопасная выдача classification proposal без allocation snapshot, carrier/contribution matching с обновлением и восстановлением basis после waiting → separate, замена или очистка прежнего rule allocation при исправлении классификации, пакетная SQL-загрузка, полный hash review, подтверждённые merchant aliases, монотонная rule boundary при поздней классификации операции и позиции, одинаковом и регрессировавшем времени, сохранение явной позиции при разрешении другой, semantic decimal no-change, preview, replay, concurrent revision, миграция и права без изменения проводок.

Task-1.6 и task-2.6 включены в базу; команды реализованы. Доказательства и границы: evidence/task-2.8-family-allocation.md. Доказаны backend-части AC-078/079/080/081/086/093; возвратные части AC-065/091 остаются task-2.7. UI, бюджеты, реальный OpenAI, банки и production не подтверждаются.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Allocate household expenses and items to members.

**Status:** Household transaction and item allocation backend/API, versioned rules and exact member analytics without a duplicate financial effect are implemented. UI, refunds, budgets and real OpenAI remain with their owning tasks.

**Dependencies:** `task-2.6`, `task-1.6`.

**Kind:** `implementation`.

### Change and contracts

Ledger stores immutable allocation snapshots by MembershipID independently from payer, owner and actor; allocation/domain selects versioned merchant/category rules. Exact amounts retain their source decimal precision, while shares and equal mode use largest remainder with MembershipID tie-break; personal/shared, item override, a retained transaction-fallback basis and the exact basis of a rule-derived item support recalculation. A zero-net item becomes not_applicable while retaining its basis for a possible later recalculation. Unknown purpose remains unallocated and conflicting equal-priority rules require resolution. The first ledger revision captures a monotonic household rule-sequence boundary; late classification cannot see rules/revisions after it even when recorded time is equal or regresses. Classification correction re-resolves unprotected parts while preserving explicit purchase/item inputs, replacing the prior rule or clearing stale shares. Semantically equal decimals with different scales create no revision. Source import applies a merchant rule only through a confirmed alias, source updates cannot erase a user allocation, and matching retains one household and one member effect according to carrier/contribution; waiting retains and refreshes the basis without a member effect and separate restores it. Rules for fallback and all unique conditions are read once in a batch; allocation snapshots for the whole transaction page use a fixed number of set-based SQL queries.

### Change boundaries

- `backend/internal/allocation/`
- `backend/internal/ledger/`
- `backend/internal/storage/`
- `backend/internal/delivery/allocation/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/017_family_allocations.sql`
- `api/`
- `backend/test/integration/family-allocation/`

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

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-065

- **Given:** A USD 10 purchase is split across two categories and valued at RUB 900.
- **When:** USD 4 is later refunded for a known item at a different current rate.
- **Then:** The historical category decreases by the refunded original value RUB 360; actual cash receipts and FX difference stay separate; excess cumulative refunds are rejected.
- **Level:** `unit+integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-080

- **Given:** Salary arrived in A’s account, B paid joint rent and A has no available balance.
- **When:** The household budget, individual expenses and currency funding are calculated.
- **Then:** Income is pooled with recipient retained; rent appears once for the household and by shares in individual views. Household availability includes both members’ funds without automatic currency exchange or credit limits.
- **Level:** `integration`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
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
make check
make test-integration AREA=all
make test-family-allocation-race
make test-integration AREA=privacy
git diff --check
```

A mixed 1,000 receipt yields household 1,000, A 400 and B 600; all six assets and arbitrary precision are retained, including an explicit 0.5/0.5 split of amount 1. Share/equal recalculation, amount-based immutability, a zero-net item, an excluded expense during migration over 015, safe classification-proposal output without an allocation snapshot, matching carrier/contribution with basis refresh and restoration after waiting → separate, replacement or clearing of prior rule allocation after classification correction, set-based SQL hydration, the complete review hash, confirmed merchant aliases, the monotonic rule boundary for late transaction and item classification with equal or regressed time, explicit-item preservation while another item resolves, semantic decimal no-change, preview, replay, concurrent revision, migration and permissions are checked without changing postings.

Task-1.6 and task-2.6 are included in the base; commands exist. Evidence and boundaries: evidence/task-2.8-family-allocation.en.md. Backend portions of AC-078/079/080/081/086/093 are proven; refund portions of AC-065/091 remain with task-2.7. UI, budgets, real OpenAI, banks and production are not verified.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
