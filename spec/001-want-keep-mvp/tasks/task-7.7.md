<!-- want-keep-task: task-7.7 -->
# task-7.7 — Показать кредитки, накопления и доходность / Show credit cards, savings and returns

## RU

Сделать условия, прогнозы и результаты продуктов понятными.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-7.2`, `task-6.2`, `task-6.3`, `task-6.4`, `task-6.5`, `task-7.9`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Desktop-оболочка, отдельный вход RU/EN, дизайн-система и backend семейных прав включены в базу. Их наличие не доказывает готовность всех экранов и сценариев этой карточки.

### Требуемое поведение

Показать долг/minimum/due/grace с provenance, фактические/прогнозные начисления, метод доходности и gross/net/fees/funding/mining отдельно. Unknown terms не превращать в зелёный статус льготы; оценочные результаты имеют метод и валюту. Финансовые формулы не реализовывать в компонентах UI.

### Ключевые контракты

Ответы credit/savings/returns/crypto reports с native/reporting валютами, методами, actual/forecast и provenance.

### Вне объёма

Только desktop web на ноутбуке с macOS; мобильные экраны не добавлять. Финансовые расчёты и проверка полномочий остаются на сервере. Не расширять согласованный продуктовый объём.

### Блокирующие зависимости

- [#49](https://github.com/andrey-krasheninnikov/want-keep/issues/49) (`task-7.2`): открыта; блокирует выполнение.
- [#41](https://github.com/andrey-krasheninnikov/want-keep/issues/41) (`task-6.2`): открыта; блокирует выполнение.
- [#42](https://github.com/andrey-krasheninnikov/want-keep/issues/42) (`task-6.3`): открыта; блокирует выполнение.
- [#43](https://github.com/andrey-krasheninnikov/want-keep/issues/43) (`task-6.4`): открыта; блокирует выполнение.
- [#44](https://github.com/andrey-krasheninnikov/want-keep/issues/44) (`task-6.5`): открыта; блокирует выполнение.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): открыта; блокирует выполнение.

### Границы изменений

- `web/src/features/credit/`
- `web/src/features/savings/`
- `web/src/features/returns/`

### Экранный контракт

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

### SCR-021 — Доходность

`/analytics/returns`

**Вопрос:** Какие накопления приносят доход с учётом времени?

**Главный ответ:** Фактическая и прогнозная доходность с общей базой сравнения.

**Структура сверху вниз:** Период/валюта → итог дохода/XIRR и ограничение → продукты/денежные потоки → факт vs прогноз.

**Следующее действие:** Сравнить продукты, открыть SCR-008 или конкретные потоки SCR-010.

**Объяснение и детализация:** Даты, комиссии, FX и solver assumptions раскрыты; неопределённый/множественный корень XIRR не ноль.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

### SCR-022 — Криптоаналитика

`/analytics/crypto`

**Вопрос:** Из чего получился результат по криптоактивам?

**Главный ответ:** Реализованный/нереализованный результат отдельно от комиссий и майнинга.

**Структура сверху вниз:** Период/валюта/аккаунт → результаты → fees/funding/mining → позиции/сделки → движения.

**Следующее действие:** Открыть счёт SCR-008, объясняющие операции SCR-009/010.

**Объяснение и детализация:** Оценка портфеля/FX отдельно от торгового P&L; неизвестная себестоимость/история явно ограничивает результат.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

#### FORM-03 — Счёт и начальный остаток

**Поля:** Название, валюта, личный/семейный наличный счёт, дата начала в timezone семьи и точный остаток. Импортный продукт имеет отдельные owned/available/locked/debt, подтверждение и источник; карты — алиасы без баланса.

**Проверки и права:** Оба member создают счета и исправляют факты учёта. При смене владельца или personal/household принадлежности существующего личного счёта требуется его текущий владелец; для семейного счёта — любой member. Проверенный внешний владелец и история операций этим не меняются. Точные decimal, валюта обязательна. Импортируемые поля меняются через correction; начальный остаток не доход. Ручное создание — только наличные, личный счёт только для себя. Перенос даты сохраняет операции до неё в истории; новое открытие заменяет прежнее в расчёте. Дата не может быть в будущем. Неизвестный ответ восстанавливается по прежнему Idempotency-Key через /commands; partial/unknown не равны нулю.

**Результат:** Счёт в учёте создан/исправлен с audit; это не открытие банковского продукта.

#### FORM-05 — Учесть перевод или обмен

**Поля:** Откуда/куда, даты, обе суммы/валюты, комиссии и счёт комиссии, существующие движения.

**Проверки и права:** Оба участника; разные счета семьи; одна исходящая и одна входящая сторона. Principal не входит в доходы/расходы; отдельные комиссии, включая третий актив. Пустой existingTransactions создаёт новое движение. Непустой список содержит ID/revisions всех участников (до 100); суммы, комиссии и дата основной записи проверяются без создания недостающих сторон. Связь переводов/обменов/оплаты подтверждается по версиям, неоднозначность остаётся в matching; отдельная покупка снимает ожидание один раз.

**Результат:** Связано движение денег в журнале; никакой реальной отправки или покупки актива.

#### FORM-06 — Исправление, сопоставление и отмена

**Поля:** Операция, expectedRevision, основание; полный principal и fees, дата, payer, raw merchant/note; category и merchant identity через set|clear, позиции через replace|clear. Undo: decisionId и expectedRevisions; сравнение до/после и с источником.

**Проверки и права:** Оба участника исправляют семейные факты. Сервер сохраняет actor, счета/активы principal и происхождение, проверяет активные категории/продавцов и меняет полный item set атомарно. Позиции и скидки точно равны principal; top-level category с позициями запрещена. Undo сохраняет поздние независимые поля. Сопоставление и доли остаются профильным задачам.

**Результат:** Новое решение и финансовые revisions с историей, либо no_change/conflict без эффекта и потери ввода. Исключение не меняет банковский статус; undo пересчитывает текущий эффект.

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
- **UISTATE-15 — Нужен банковский вход:** Назвать подключение и владельца, дать ему безопасно войти; партнёру показать ожидание без доступа к секрету.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-034:** Доходность вложений сравнивается с учётом дат денежных потоков и валюты оценки.
- **REQ-035:** Торговая аналитика отделяет реализованный результат, нереализованный результат, комиссии и funding.
- **REQ-036:** Вознаграждения майнинга отделены от переводов между собственными кошельками.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-086:** Детализация счёта зависит от продукта и показывает доступность денег перед служебными сведениями.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-031

- **Дано:** Покупка RUB 1 000 сделана с кредитки, затем долг погашен с собственного счёта.
- **Когда:** Формируется бюджет и сводка кредитки.
- **Тогда:** Покупка учтена один раз; погашение не второй расход; проценты и комиссии — отдельные расходы; неизвестный минимальный платёж не вычисляется догадкой.
- **Уровень:** `integration`.

#### AC-032

- **Дано:** Для карты подтверждены условия, выписка, исключения и крайняя дата.
- **Когда:** Совершаются покупка, частичное погашение и операция, исключённая из льготы.
- **Тогда:** Сумма и срок согласованы с подтверждёнными условиями; при нехватке условий отображается неизвестность, а не обещание сохранения льготы.
- **Уровень:** `contract`.

#### AC-033

- **Дано:** Есть вклад или Earn с подтверждёнными условиями, пополнением и выводом.
- **Когда:** Рассчитывается доход за период и прогноз.
- **Тогда:** Факт отделён от прогноза и переоценки; смена ставки и капитализация учитываются по условиям; неизвестные условия блокируют точный прогноз.
- **Уровень:** `integration`.

#### AC-034

- **Дано:** Два вложения имеют разные даты пополнений и одинаковый конечный остаток.
- **Когда:** Строится сравнение доходности.
- **Тогда:** Показаны фактический доход и годовая денежно-взвешенная доходность с датами/методом; некорректные или неоднозначные расчёты обозначены недоступными, не нулём.
- **Уровень:** `unit`.

#### AC-035

- **Дано:** Источник передал реализованный результат USDT 10, комиссию USDT 1 и нереализованный результат USDT 5.
- **Когда:** Обновляется отчёт торгового счёта.
- **Тогда:** Показатели разделены; нереализованные USDT 5 не становятся полученным доходом; чистый результат не дублируется отдельным повторным вычетом уже включённой комиссии.
- **Уровень:** `integration`.

#### AC-036

- **Дано:** Награда BTC 0.0001 начислена на mining-счёт и затем переведена на основной.
- **Когда:** Обе записи импортируются.
- **Тогда:** Доход учитывается при подтверждённом начислении один раз; последующий перевод дохода не создаёт.
- **Уровень:** `integration`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
- **Уровень:** `integration`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

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

#### AC-103

- **Дано:** Есть текущий счёт, кредитка, вклад/Earn/Coinhold и криптопродукт с неполными данными.
- **Когда:** Пользователь открывает каждый вариант SCR-008.
- **Тогда:** Текущий счёт показывает доступно/резерв/блокировки, кредитка долг/ближайший платёж/grace, накопление факт/прогноз/срок, крипто доступные активы/позиции/комиссии. Неизвестные условия помечены, заёмные деньги не объявляются доступным семейным пулом.
- **Уровень:** `manual+e2e`.

### Проверка результата

```sh
make test-web FILTER=products && make e2e SCENARIO=financial-products
```

Кредитное погашение, неизвестный grace, XIRR unavailable и trading fees отображаются с правильным смыслом.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Make product terms, forecasts and results understandable.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-7.2`, `task-6.2`, `task-6.3`, `task-6.4`, `task-6.5`, `task-7.9`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

The desktop shell, separate RU/EN sign-in, design system and household-permission backend are included in the base. Their presence does not establish readiness of every screen and scenario in this task.

### Desired behavior

Show debt/minimum/due/grace with provenance, actual/forecast accruals, return method and separate gross/net/fees/funding/mining. Unknown terms do not become a green grace status; estimates include method and currency. UI components do not own financial formulas.

### Key interfaces

Credit/savings/returns/crypto-report responses with native/reporting currencies, methods, actual/forecast and provenance.

### Out of scope

Desktop web on a macOS laptop only; do not add mobile screens. Financial calculations and authority checks remain server-side. Do not expand the agreed product scope.

### Blocked by

- [#49](https://github.com/andrey-krasheninnikov/want-keep/issues/49) (`task-7.2`): open; blocks execution.
- [#41](https://github.com/andrey-krasheninnikov/want-keep/issues/41) (`task-6.2`): open; blocks execution.
- [#42](https://github.com/andrey-krasheninnikov/want-keep/issues/42) (`task-6.3`): open; blocks execution.
- [#43](https://github.com/andrey-krasheninnikov/want-keep/issues/43) (`task-6.4`): open; blocks execution.
- [#44](https://github.com/andrey-krasheninnikov/want-keep/issues/44) (`task-6.5`): open; blocks execution.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): open; blocks execution.

### Change boundaries

- `web/src/features/credit/`
- `web/src/features/savings/`
- `web/src/features/returns/`

### Screen contract

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

### SCR-021 — Returns

`/analytics/returns`

**Question:** Which savings earn returns accounting for time?

**Primary answer:** Actual and forecast returns on a common comparison basis.

**Top-down structure:** Period/currency → income/XIRR and limitation → products/cash flows → actual vs forecast.

**Next action:** Compare products, open SCR-008 or specific flows SCR-010.

**Explanation and details:** Dates, fees, FX and solver assumptions expand; undefined/multiple XIRR roots are not zero.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

### SCR-022 — Crypto analytics

`/analytics/crypto`

**Question:** What produced the crypto result?

**Primary answer:** Realized/unrealized results separate from fees and mining.

**Top-down structure:** Period/currency/account → results → fees/funding/mining → positions/trades → movements.

**Next action:** Open account SCR-008, explanatory transactions SCR-009/010.

**Explanation and details:** Portfolio valuation/FX separate from trading P&L; unknown cost basis/history explicitly limits result.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

#### FORM-03 — Account and opening balance

**Fields:** Name, asset, personal/household cash account, start date in household timezone and exact opening balance. Imported products have separate owned/available/locked/debt, confirmation and provenance; cards are balance-free aliases.

**Validation and permissions:** Both members create accounts and correct accounting facts. Changing owner or personal/household scope of an existing personal account requires its current owner; either member may change a household account. This never changes verified external ownership or transaction history. Exact decimals and currency required. Imported fields change through correction; opening balance is not income. Manual creation is cash only; a personal account is created for oneself. Moving the date keeps earlier operations in history; the new opening replaces the previous calculation input. The date cannot be in the future. Unknown outcomes use the original Idempotency-Key via /commands; partial/unknown are not zero.

**Outcome:** Accounting account created/corrected with audit; this does not open a bank product.

#### FORM-05 — Record transfer or exchange

**Fields:** From/to accounts, dates, both amounts/currencies, fees/fee account, existing movements.

**Validation and permissions:** Either member; distinct household accounts; one outgoing and one incoming side. Principal is excluded from income/expenses; separate fees may use a third asset. Empty existingTransactions creates new movement. A nonempty list supplies all participant IDs/revisions (up to 100); amounts, fees and primary date are checked without creating missing sides. Transfer/exchange/payment links require version checks; ambiguity remains in matching; separate-purchase confirmation releases waiting once.

**Outcome:** Ledger movements linked; no actual transfer or asset purchase.

#### FORM-06 — Correction, matching and undo

**Fields:** Transaction, expectedRevision and reason; complete principal and fees, date, payer, raw merchant/note; category and merchant identity via set|clear, items via replace|clear. Undo uses decisionId and expectedRevisions; compare before/after and source.

**Validation and permissions:** Both members correct household facts. The server retains actor, principal accounts/assets and provenance, validates active categories/merchants and replaces the complete item set atomically. Items and discounts equal principal exactly; top-level category with items is forbidden. Undo preserves later independent fields. Matching and shares remain with their owning tasks.

**Outcome:** New decision and financial revisions with history, or no_change/conflict without effect or lost input. Exclusion does not change bank state; undo recomputes the current effect.

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
- **UISTATE-15 — Bank sign-in needed:** Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-034:** Investment returns are compared using dated cash flows and valuation currency.
- **REQ-035:** Trading analytics separates realized P&L, unrealized P&L, fees and funding.
- **REQ-036:** Mining rewards are separate from transfers between owned wallets.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-086:** Account details depend on the product and show availability before technical details.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-031

- **Given:** A RUB 1,000 credit-card purchase is followed by repayment from an owned account.
- **When:** The budget and card summary are built.
- **Then:** The purchase is counted once; repayment is not another expense; interest and fees are separate expenses; an unknown minimum payment is not guessed.
- **Level:** `integration`.

#### AC-032

- **Given:** Card terms, statement, exclusions and deadline are confirmed.
- **When:** A purchase, partial repayment and grace-excluded transaction occur.
- **Then:** Amount and deadline follow confirmed terms; missing terms produce an unknown state rather than a promise of grace eligibility.
- **Level:** `contract`.

#### AC-033

- **Given:** A deposit or Earn product has confirmed terms, a top-up and a withdrawal.
- **When:** Period income and forecast are calculated.
- **Then:** Actual income is separate from forecast and revaluation; rate changes and compounding follow the terms; unknown terms prevent an exact forecast.
- **Level:** `integration`.

#### AC-034

- **Given:** Two investments have different top-up dates and the same ending balance.
- **When:** A return comparison is built.
- **Then:** Actual income and annualized money-weighted return show dates/method; invalid or ambiguous calculations are unavailable rather than zero.
- **Level:** `unit`.

#### AC-035

- **Given:** A source reports USDT 10 realized P&L, USDT 1 fee and USDT 5 unrealized P&L.
- **When:** The trading account report updates.
- **Then:** Metrics are separate; unrealized USDT 5 is not received income; net results do not suffer a second deduction for already-included fees.
- **Level:** `integration`.

#### AC-036

- **Given:** A BTC 0.0001 reward is credited to mining and transferred to the main account.
- **When:** Both records are imported.
- **Then:** Income is recognized once on confirmed credit; the subsequent transfer creates no additional income.
- **Level:** `integration`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
- **Level:** `integration`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

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

#### AC-103

- **Given:** A current account, credit card, deposit/Earn/Coinhold and crypto product have incomplete data.
- **When:** The user opens each SCR-008 variant.
- **Then:** Current account shows available/reserved/blocked, credit shows debt/next payment/grace, savings shows actual/forecast/maturity, crypto shows available assets/positions/fees. Unknown terms are labelled; borrowed funds are not presented as available household pool.
- **Level:** `manual+e2e`.

### Verification

```sh
make test-web FILTER=products && make e2e SCENARIO=financial-products
```

Credit repayment, unknown grace, unavailable XIRR and trading fees display correct semantics.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
