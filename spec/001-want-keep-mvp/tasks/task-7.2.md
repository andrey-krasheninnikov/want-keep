<!-- want-keep-task: task-7.2 -->
# task-7.2 — Показать счета, операции и исправления / Show accounts, transactions and corrections

## RU

Дать владельцу проверяемую ленту и управление ошибками учёта.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-7.1`, `task-2.5`, `task-2.6`, `task-2.7`, `task-5.2`, `task-7.9`, `task-2.9`.

**Тип:** `implementation`.

### Изменение и контракты

Показать счета/карты без двойных балансов, native/reporting amounts, history coverage, sync/MFA states, transfer links, source evidence и audit. Реализовать ручной ввод наличных, фильтры/категоризацию, разрешение дубликатов и correction/undo через версии API. Простое чтение экрана не запускает запись или AI-задания. Реализовать SCR-012 сверки и SCR-013 взаиморасчётов с явным созданием долга и связью возмещения; запись не исполняет банковский платёж.

### Границы изменений

- `web/src/features/accounts/`
- `web/src/features/transactions/`
- `web/src/features/connections/`

### Экранный контракт

### SCR-007 — Деньги и счета

`/accounts`

**Вопрос:** Где деньги и сколько доступно?

**Главный ответ:** Собственные средства, доступно, резерв и задолженность по валютам.

**Структура сверху вниз:** Сводка → личные/семейные группы счетов → доступно/резерв/блокировка/долг → свежесть.

**Следующее действие:** Открыть SCR-008, добавить счёт FORM-03 или подключить SCR-029; операции → SCR-009.

**Объяснение и детализация:** Эквивалент не означает обеспеченность другой валютой; скрытые из фильтра счета остаются частью семейного пула.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-03.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

### SCR-008 — Карточка счёта

`/accounts/:id`

**Вопрос:** Что доступно именно на этом счёте?

**Главный ответ:** Ответ зависит от продукта, но доступность и обязательства идут первыми.

**Структура сверху вниз:** Текущий: доступно/резерв/блокировки. Кредитка: долг, обязательный платёж/дата, сумма сохранения grace, свои средства. Вклад/Earn/Coinhold: факт/прогноз, срок/условия вывода. Крипто: активы/позиции, доступно/маржа/блокировка, результат/fees/funding/mining. Затем операции.

**Следующее действие:** Уточнить остаток → SCR-012; посмотреть доходность → SCR-021/022; учесть движение FORM-05.

**Объяснение и детализация:** Условия, источник, дата оценки и история раскрываются; неизвестный grace/вывод не имитировать.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-03, FORM-05, FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

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

### SCR-012 — Сверка остатка

`/accounts/:id/reconciliation`

**Вопрос:** Почему остаток отличается?

**Главный ответ:** Owned, available, locked и debt источника и журнала на sourceAsOf, их точная разница, качество данных и доказанные объяснения.

**Структура сверху вниз:** Lifecycle/result и свежесть → четыре компонента source/ledger/difference → объяснения и связанные операции → replay → resolution.

**Следующее действие:** Запустить/дождаться повторной загрузки, повторно войти в источник, открыть связанные движения SCR-010 либо после completed/unavailable replay явно скорректировать owned/debt.

**Объяснение и детализация:** Unknown не равен нулю; balanced stale не становится fresh. Available/locked напрямую не корректируются. Явный adjustment не является доходом/расходом, не меняет снимок источника и заранее показывает рассчитанный эффект.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

### SCR-013 — Взаиморасчёты

`/reimbursements`

**Вопрос:** Есть ли невозмещённые суммы?

**Главный ответ:** Явные долги, непогашенный остаток и состояние связанных переводов.

**Структура сверху вниз:** Открытые и требующие внимания долги → кто кому/валюта/остаток → основание и исходный расход → активные или устаревшие погашения → история решений.

**Следующее действие:** Создать долг, связать проведённый перевод, исправить данные или отменить выбранное решение FORM-09; открыть исходную операцию SCR-010.

**Объяснение и детализация:** Распределение, плательщик и перевод сами долг не создают. Комиссия не уменьшает остаток. Межвалютное погашение показывает обе native-суммы. Изменившийся расход или перевод требует явного решения.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-09.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

#### FORM-03 — Счёт и начальный остаток

**Поля:** Название, валюта, личный/семейный наличный счёт, дата начала в timezone семьи и точный остаток. Импортный продукт имеет отдельные owned/available/locked/debt, подтверждение и источник; карты — алиасы без баланса.

**Проверки и права:** Оба member создают счета и исправляют факты учёта. При смене владельца или personal/household принадлежности существующего личного счёта требуется его текущий владелец; для семейного счёта — любой member. Проверенный внешний владелец и история операций этим не меняются. Точные decimal, валюта обязательна. Импортируемые поля меняются через correction; начальный остаток не доход. Ручное создание — только наличные, личный счёт только для себя. Перенос даты сохраняет операции до неё в истории; новое открытие заменяет прежнее в расчёте. Дата не может быть в будущем. Неизвестный ответ восстанавливается по прежнему Idempotency-Key через /commands; partial/unknown не равны нулю.

**Результат:** Счёт в учёте создан/исправлен с audit; это не открытие банковского продукта.

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
- **UISTATE-15 — Нужен банковский вход:** Назвать подключение и владельца, дать ему безопасно войти; партнёру показать ожидание без доступа к секрету.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-002:** Учёт поддерживает RUB, USD, USDT, USDC, BTC и ETH; наличные, банковские деньги и платформенные кошельки различаются счетами.
- **REQ-003:** Общую валюту отображения можно переключать между RUB, USD, USDT, USDC, BTC и ETH.
- **REQ-004:** Начало учёта задаётся датой; начальные остатки отделены от доходов и расходов.
- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-009:** Статусы ожидающей, проведённой, отменённой и возвращённой операции учитываются явно.
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-012:** Исправление учёта сохраняет оригинал, автора, основание, версию и возможность отмены решения.
- **REQ-013:** Расхождение журнала и баланса источника расследуется без скрытого автоматического выравнивания.
- **REQ-014:** Категория, подкатегория, продавец и позиция чека являются отдельными аналитическими признаками.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-018:** Каждая новая или содержательно изменённая операция получает AI-проверку своей версии.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-068:** Взаимный долг учитывается только по явному указанию и не увеличивает активы или расходы семьи.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-002

- **Дано:** Созданы RUB наличные 1 000, RUB банк 2 000, USD наличные 10, USDT 20, USDC 12.000000000123, BTC 0.001 и ETH 0.001234567891.
- **Когда:** Владелец открывает счета.
- **Тогда:** Показаны семь отдельных счетов с исходными активами и точными остатками; дробные остатки не обрезаются до точности UI или заказа провайдера. RUB суммируется только в соответствующем срезе.
- **Уровень:** `integration`.

#### AC-003

- **Дано:** Для всех необходимых пар есть актуальная оценка.
- **Когда:** Участник переключает RUB на USD, USDT, USDC, BTC и ETH.
- **Тогда:** Меняется эквивалент итогов, исходные суммы операций и счетов сохраняются.
- **Уровень:** `end-to-end`.

#### AC-004

- **Дано:** История запрошена с 1 августа; начальный остаток RUB 5 000 подтверждён.
- **Когда:** Импортируется расход RUB 500 от 2 августа.
- **Тогда:** Остаток равен RUB 4 500; доход августа не увеличивается на начальные RUB 5 000; неподтверждённое начало обозначается явно.
- **Уровень:** `integration`.

#### AC-005

- **Дано:** Источник сообщает собственные RUB 100, долг RUB 300 и кредитный лимит RUB 1 000.
- **Когда:** Строится сводка денег.
- **Тогда:** Кредитный лимит не увеличивает собственный капитал или доступный бюджет; отсутствующее поле отображается как неизвестное.
- **Уровень:** `integration`.

#### AC-008

- **Дано:** Расход RUB 300 создан из чата с выбранным счётом.
- **Когда:** Поступают соответствующий чек, банковская операция и повтор той же операции.
- **Тогда:** Расход остаётся RUB 300, все источники связаны; две отдельные покупки одной суммы не объединяются лишь из-за равенства суммы.
- **Уровень:** `integration`.

#### AC-009

- **Дано:** Карточная авторизация RUB 500 сначала ожидает подтверждения.
- **Когда:** Она проводится либо отменяется.
- **Тогда:** Проведение создаёт один фактический расход; отмена ожидающей операции расхода не создаёт. Заблокированная сумма и статус не скрыты.
- **Уровень:** `integration`.

#### AC-012

- **Дано:** AI ошибочно связал две операции; исходные импортированные записи сохранены.
- **Когда:** Владелец отменяет связь и исправляет категорию.
- **Тогда:** Пересчитаны производные отчёты; видна история; повторный импорт не стирает правку владельца.
- **Уровень:** `integration`.

#### AC-013

- **Дано:** Учёт показывает RUB 900, источник RUB 1 000 на сопоставимый момент.
- **Когда:** Завершается сверка.
- **Тогда:** Показаны оба остатка и разница RUB 100; повторно проверяется история; корректирующая запись требует установленной причины или решения владельца.
- **Уровень:** `integration`.

#### AC-014

- **Дано:** Чек содержит молоко, продавец — условный магазин; другая покупка — ресторан.
- **Когда:** Владелец фильтрует расходы и исправляет категорию.
- **Тогда:** Доступны независимые срезы по виду расхода, продавцу и товару; пользовательская правка сохраняется.
- **Уровень:** `end-to-end`.

#### AC-019

- **Дано:** AI предлагает сумму, противоречащую источнику, и связь с несколькими кандидатами.
- **Когда:** Приложение проверяет предложения.
- **Тогда:** Противоречивое изменение отклонено, неоднозначность поступает в очередь уточнений; категории и подтверждённые связи могут применяться автоматически.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

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

#### AC-064

- **Дано:** На одном счёте две покупки RUB 300 близко по времени; владелец исправил одну.
- **Когда:** AI получает чек без уникального идентификатора и старый результат классификации.
- **Тогда:** Запрашивается выбор совпадения; покупки не сливаются по вероятности; правка человека не теряется.
- **Уровень:** `integration`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-082

- **Дано:** A оплачивает общий расход RUB 1000 и явно отмечает возмещение RUB 300 от B.
- **Когда:** B переводит 100, затем 200; обе стороны переводов импортируются повторно.
- **Тогда:** Долг уменьшается 300→200→0 один раз; основная сумма переводов не доход/расход. Обычная покупка без указания долг не создаёт; валютное погашение требует явного соответствия сумм.
- **Уровень:** `integration`.

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

#### AC-006

- **Дано:** Два собственных RUB-счёта и перевод RUB 1 000 с комиссией RUB 10.
- **Когда:** Получены обе стороны перевода в любом порядке.
- **Тогда:** Связана одна операция перевода; основная сумма исключена из доходов/расходов, комиссия RUB 10 учтена один раз.
- **Уровень:** `integration`.

#### AC-010

- **Дано:** В августе оплачен расход RUB 1 000.
- **Когда:** В сентябре получен связанный частичный возврат RUB 400.
- **Тогда:** Расход августа становится RUB 600; движение RUB +400 остаётся в сентябре; пересчёт и связь доступны в истории.
- **Уровень:** `integration`.

#### AC-105

- **Дано:** Есть личный счёт A, семейный счёт и запись покупки; B видит их и может исправлять учёт.
- **Когда:** B исправляет покупку и пытается через FORM-03/API изменить владельца или scope счёта A; затем A меняет свой счёт и B меняет семейный.
- **Тогда:** Исправление покупки разрешено, изменение принадлежности личного счёта B отклоняется сервером. Действия текущего владельца и изменение семейного счёта разрешены с revision/audit. Внешний владелец и история операций не изменены, фильтр участника не даёт дополнительных прав.
- **Уровень:** `integration+e2e`.

### Проверка результата

```sh
make test-web FILTER=transactions && make e2e SCENARIO=accounting
```

Ввод/исправление, late import, сверка и неизвестные поля работают без ложных итогов.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Give the owner a verifiable activity feed and accounting corrections.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-7.1`, `task-2.5`, `task-2.6`, `task-2.7`, `task-5.2`, `task-7.9`, `task-2.9`.

**Kind:** `implementation`.

### Change and contracts

Show accounts/cards without duplicate balances, native/reporting amounts, history coverage, sync/MFA states, transfer links, source evidence and audit. Implement manual cash entry, filters/classification, duplicate resolution and correction/undo through versioned APIs. Reading a screen does not trigger writes or AI jobs. Implement SCR-012 reconciliation and SCR-013 reimbursements with explicit debt creation and settlement links; recording never executes a bank payment.

### Change boundaries

- `web/src/features/accounts/`
- `web/src/features/transactions/`
- `web/src/features/connections/`

### Screen contract

### SCR-007 — Money and accounts

`/accounts`

**Question:** Where is the money and how much is available?

**Primary answer:** Own funds, available, reserved and debt by currency.

**Top-down structure:** Summary → personal/household account groups → available/reserved/blocked/debt → freshness.

**Next action:** Open SCR-008, add account FORM-03 or connect SCR-029; transactions → SCR-009.

**Explanation and details:** Equivalent does not fund another currency; accounts outside a view filter remain in the household pool.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-03.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

### SCR-008 — Account details

`/accounts/:id`

**Question:** What is available in this account?

**Primary answer:** The answer depends on product; availability and obligations come first.

**Top-down structure:** Current: available/reserved/blocked. Credit: debt, required payment/date, grace-preserving amount, own funds. Deposit/Earn/Coinhold: actual/forecast, maturity/withdrawal terms. Crypto: assets/positions, available/margin/blocked, P&L/fees/funding/mining. Then transactions.

**Next action:** Resolve balance → SCR-012; returns → SCR-021/022; record movement FORM-05.

**Explanation and details:** Terms, source, valuation date and history expand; never invent unknown grace/withdrawal terms.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-03, FORM-05, FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

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

### SCR-012 — Balance reconciliation

`/accounts/:id/reconciliation`

**Question:** Why does the balance differ?

**Primary answer:** Source and ledger owned, available, locked and debt at sourceAsOf, their exact differences, data quality and evidence-backed explanations.

**Top-down structure:** Lifecycle/result and freshness → four source/ledger/difference components → explanations and related transactions → replay → resolution.

**Next action:** Start/wait for bounded replay, reauthenticate the source, open related movements in SCR-010 or, after completed/unavailable replay, explicitly adjust owned/debt.

**Explanation and details:** Unknown is not zero; balanced stale does not become fresh. Available/locked cannot be adjusted directly. An explicit adjustment is not income/expense, never changes the source observation and previews the server-derived effect.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

### SCR-013 — Reimbursements

`/reimbursements`

**Question:** Are any reimbursements outstanding?

**Primary answer:** Explicit debts, outstanding amounts and linked-transfer state.

**Top-down structure:** Open and attention-required debts → debtor/creditor/currency/outstanding → reason and source expense → active or stale settlements → decision history.

**Next action:** Create debt, link a posted transfer, correct data or undo a selected decision in FORM-09; open the source transaction in SCR-010.

**Explanation and details:** Allocation, payer and transfer never create debt by themselves. Fees do not reduce outstanding. Cross-asset settlement shows both native amounts. A changed expense or transfer requires an explicit decision.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-09.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

#### FORM-03 — Account and opening balance

**Fields:** Name, asset, personal/household cash account, start date in household timezone and exact opening balance. Imported products have separate owned/available/locked/debt, confirmation and provenance; cards are balance-free aliases.

**Validation and permissions:** Both members create accounts and correct accounting facts. Changing owner or personal/household scope of an existing personal account requires its current owner; either member may change a household account. This never changes verified external ownership or transaction history. Exact decimals and currency required. Imported fields change through correction; opening balance is not income. Manual creation is cash only; a personal account is created for oneself. Moving the date keeps earlier operations in history; the new opening replaces the previous calculation input. The date cannot be in the future. Unknown outcomes use the original Idempotency-Key via /commands; partial/unknown are not zero.

**Outcome:** Accounting account created/corrected with audit; this does not open a bank product.

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
- **UISTATE-15 — Bank sign-in needed:** Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-002:** Accounting supports RUB, USD, USDT, USDC, BTC and ETH; cash, bank money and platform wallets are separate accounts.
- **REQ-003:** The reporting currency can switch among RUB, USD, USDT, USDC, BTC and ETH.
- **REQ-004:** Accounting starts on a selected date; opening balances are separate from income and expenses.
- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-009:** Pending, posted, cancelled and refunded transaction states are explicit.
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-012:** Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.
- **REQ-013:** Ledger/source balance discrepancies are investigated without hidden automatic balancing.
- **REQ-014:** Category, subcategory, merchant and receipt item are separate analytical dimensions.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-018:** Every new or materially changed transaction receives AI review of its version.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-068:** An inter-member debt is recorded only explicitly and does not increase household assets or expenses.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-002

- **Given:** Accounts contain RUB cash 1,000, RUB bank 2,000, USD cash 10, USDT 20, USDC 12.000000000123, BTC 0.001 and ETH 0.001234567891.
- **When:** The owner opens accounts.
- **Then:** Seven distinct accounts show original assets and exact balances; residuals are not truncated to UI or provider order precision. RUB is combined only in the relevant aggregate.
- **Level:** `integration`.

#### AC-003

- **Given:** A current valuation exists for every required pair.
- **When:** The member switches RUB to USD, USDT, USDC, BTC and ETH.
- **Then:** Equivalent totals change while original account and transaction amounts remain unchanged.
- **Level:** `end-to-end`.

#### AC-004

- **Given:** History is requested from August 1; an opening RUB 5,000 balance is confirmed.
- **When:** A RUB 500 expense dated August 2 is imported.
- **Then:** Balance is RUB 4,500; August income excludes the opening RUB 5,000; an unverified opening is explicit.
- **Level:** `integration`.

#### AC-005

- **Given:** The source reports RUB 100 owned, RUB 300 debt and a RUB 1,000 credit limit.
- **When:** A money summary is built.
- **Then:** The credit limit does not increase net worth or the spendable budget; missing fields are shown as unknown.
- **Level:** `integration`.

#### AC-008

- **Given:** A RUB 300 expense was created from chat for a selected account.
- **When:** The matching receipt, bank transaction and duplicate bank delivery arrive.
- **Then:** Expense remains RUB 300 and all evidence is linked; separate equal-amount purchases are not merged merely by amount.
- **Level:** `integration`.

#### AC-009

- **Given:** A RUB 500 card authorization is initially pending.
- **When:** It posts or is cancelled.
- **Then:** Posting creates one actual expense; cancelling a pending authorization creates none. The hold and status remain visible.
- **Level:** `integration`.

#### AC-012

- **Given:** AI incorrectly linked two transactions; original imports are retained.
- **When:** The owner unlinks them and corrects the category.
- **Then:** Derived reports are recalculated, history is visible and reimport does not overwrite the owner's correction.
- **Level:** `integration`.

#### AC-013

- **Given:** The ledger shows RUB 900 and the source RUB 1,000 at a comparable instant.
- **When:** Reconciliation completes.
- **Then:** Both balances and the RUB 100 discrepancy are shown; history is rechecked; an adjustment requires an established cause or the owner's decision.
- **Level:** `integration`.

#### AC-014

- **Given:** A receipt contains milk from a fictional store; another purchase is from a restaurant.
- **When:** The owner filters expenses and corrects a category.
- **Then:** Expense type, merchant and item can be analyzed independently; owner corrections persist.
- **Level:** `end-to-end`.

#### AC-019

- **Given:** AI proposes a source-conflicting amount and a match with several candidates.
- **When:** The application validates the proposals.
- **Then:** The conflicting change is rejected and ambiguity enters the clarification queue; categories and substantiated links may be applied automatically.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

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

#### AC-064

- **Given:** One account has two RUB 300 purchases close in time; the owner corrected one.
- **When:** AI receives a receipt without a unique identifier and a stale classification result.
- **Then:** A match selection is requested; purchases are not merged merely by likelihood and the owner's correction survives.
- **Level:** `integration`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-082

- **Given:** A pays a RUB 1,000 joint expense and explicitly records RUB 300 reimbursement due from B.
- **When:** B transfers 100 and then 200; both legs of the transfers are imported again.
- **Then:** Debt falls 300→200→0 once; transfer principal is not income/expense. An ordinary purchase creates no debt without instruction; cross-currency settlement requires explicit amount mapping.
- **Level:** `integration`.

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

#### AC-006

- **Given:** Two owned RUB accounts and a RUB 1,000 transfer with a RUB 10 fee.
- **When:** Both transfer legs arrive in either order.
- **Then:** One transfer is linked; principal is excluded from income/expenses and the RUB 10 fee is counted once.
- **Level:** `integration`.

#### AC-010

- **Given:** A RUB 1,000 expense was paid in August.
- **When:** A linked RUB 400 partial refund is received in September.
- **Then:** August expense becomes RUB 600; the RUB +400 cash movement stays in September; the recalculation and link are auditable.
- **Level:** `integration`.

#### AC-105

- **Given:** There is A’s personal account, a household account and a purchase; B can read them and correct accounting.
- **When:** B corrects the purchase and uses FORM-03/API to change A’s account owner/scope; A then changes their own account and B changes the household account.
- **Then:** Purchase correction succeeds; B’s personal-account ownership change is rejected server-side. Current-owner actions and household-account changes succeed with revision/audit. External ownership and transaction history remain unchanged; member filter grants no extra authority.
- **Level:** `integration+e2e`.

### Verification

```sh
make test-web FILTER=transactions && make e2e SCENARIO=accounting
```

Entry/correction, late import, reconciliation and unknown fields work without misleading totals.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
