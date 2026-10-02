<!-- want-keep-task: task-7.6 -->
# task-7.6 — Собрать дашборд, лимиты и инсайты / Build the dashboard, limits and insights

## RU

Дать сводку месяца с проверяемыми деталями и валютами.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-7.2`, `task-7.4`, `task-7.5`, `task-6.8`, `task-5.5`, `task-7.9`.

**Тип:** `implementation`.

### Изменение и контракты

Показать план/факт, net worth, income/expense categories/merchants, daily available/forecast limits, goals и insights. Для каждого агрегата обеспечить drill-down и source/time/coverage; future income и revaluation отдельны. Общий эквивалент лимитов информационный, не обещает кросс-валютную ликвидность; incomplete не скрывается.

### Границы изменений

- `web/src/features/dashboard/`

### Экранный контракт

### SCR-006 — Обзор

`/overview`

**Вопрос:** Сколько можно потратить и хватит ли на обязательства?

**Главный ответ:** Доступный дневной лимит по валютам и ближайший риск нехватки.

**Структура сверху вниз:** Лимит/ближайшие платежи → план/факт → важные действия → цели → средства. Семья/участник, месяц, валюта видимы.

**Следующее действие:** Объяснить сумму → SCR-017; обязательства → SCR-016; уточнение → SCR-025.

**Объяснение и детализация:** Формула/источники/период/история → операции; прогноз отдельно от доступного, текущая переоценка отдельно.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14, UISTATE-15, UISTATE-16.

### SCR-017 — Дневные лимиты

`/plan/limits`

**Вопрос:** Сколько можно сегодня и почему?

**Главный ответ:** Доступный и прогнозный варианты по валюте/категории/участнику.

**Структура сверху вниз:** Доступно сегодня → прогноз отдельно → оставшиеся дни → обязательства/резервы/факт → личное распределение.

**Следующее действие:** Раскрыть формулу, изменить свой план SCR-015 или резерв SCR-019.

**Объяснение и детализация:** Сумма личных лимитов не больше семейного; дефицит показан явно, недостающие входы не превращаются в ноль.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14.

### SCR-020 — Доходы и расходы

`/analytics`

**Вопрос:** Что изменилось в расходах и почему?

**Главный ответ:** Главные отклонения и сравнение сопоставимых периодов.

**Структура сверху вниз:** Вывод/период/валюта/семья → план vs факт и изменение → категории/продавцы/позиции → график и таблица.

**Следующее действие:** Выбрать столбец/категорию → SCR-009 с теми же фильтрами.

**Объяснение и детализация:** Возвраты пересчитывают покупку; переводы principal исключены; периоды неполной истории помечены.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

### SCR-023 — Курсы и оценка

`/analytics/rates`

**Вопрос:** По какому курсу посчитаны деньги?

**Главный ответ:** Источник/дата/направление курса и применимость.

**Структура сверху вниз:** Пара/дата → справочный курс → доступные buy/sell котировки сервиса → известные комиссии/ограничения.

**Следующее действие:** Посмотреть историческую оценку или связанные счета/операции; обновить справочные данные.

**Объяснение и детализация:** Нет обещания исполнимого обмена; отсутствие котировки не паритет USDT/USD; реальный курс операции отдельно.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

### SCR-026 — Инсайты

`/insights`

**Вопрос:** Что стоит изменить и на чём основан совет?

**Главный ответ:** Краткое наблюдение с величиной эффекта и проверяемыми данными.

**Структура сверху вниз:** Важные выводы → причина/период/сравнение → предложение → доказательства.

**Следующее действие:** Проверить операции SCR-009; принять разрешённое изменение FORM-12 или отклонить.

**Объяснение и детализация:** Неполные данные и ограничения обозначены; прогноз не обещание, AI не меняет утверждённый план молча.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-12 — Ответ и предложение AI

**Поля:** expectedRevision вопроса/предложения, subjectExpectedRevision операции и ровно choiceId либо свободный answer; для предложения явное apply/reject.

**Проверки и права:** Оба для операций, только владелец для личного плана/цели; actor не меняется текстом. Одновременный ответ проверяет версию; preview перед финансовым изменением.

**Результат:** Команда подтверждена, отказана, устарела или ожидает; ответ/основание и автор сохранены.

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
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-014:** Категория, подкатегория, продавец и позиция чека являются отдельными аналитическими признаками.
- **REQ-020:** AI-инсайты по доходам и расходам ссылаются на проверяемые данные и отделяют прогноз от факта.
- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-030:** Дневные лимиты показывают семейный и индивидуальный доступный/прогнозный остаток, по категориям и с отдельным обеспечением каждой валютой.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-038:** Курсы обмена учитывают направление, сервис, время, сумму применимости и известные комиссии.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-052:** Дашборд объединяет счета, план/факт, доходы, расходы, цели и дневные лимиты с детализацией.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-069:** Резервы личных и совместных целей задаются явно; совместные цели отображаются отдельным общим блоком без персональных долей.
- **REQ-070:** Сумма индивидуальных дневных лимитов не превышает семейный предел одной валюты; счёт плательщика не меняет долю расходов.
- **REQ-080:** Каждый экран отвечает на вопрос пользователя и ведёт к следующему полезному действию.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-003

- **Дано:** Для всех необходимых пар есть актуальная оценка.
- **Когда:** Участник переключает RUB на USD, USDT, USDC, BTC и ETH.
- **Тогда:** Меняется эквивалент итогов, исходные суммы операций и счетов сохраняются.
- **Уровень:** `end-to-end`.

#### AC-010

- **Дано:** В августе оплачен расход RUB 1 000.
- **Когда:** В сентябре получен связанный частичный возврат RUB 400.
- **Тогда:** Расход августа становится RUB 600; движение RUB +400 остаётся в сентябре; пересчёт и связь доступны в истории.
- **Уровень:** `integration`.

#### AC-014

- **Дано:** Чек содержит молоко, продавец — условный магазин; другая покупка — ресторан.
- **Когда:** Владелец фильтрует расходы и исправляет категорию.
- **Тогда:** Доступны независимые срезы по виду расхода, продавцу и товару; пользовательская правка сохраняется.
- **Уровень:** `end-to-end`.

#### AC-020

- **Дано:** В двух месяцах известны расходы по категориям; один источник устарел.
- **Когда:** AI формирует месячный инсайт.
- **Тогда:** Числа воспроизводятся отчётом; указаны период, связанные операции/срезы и неполнота; прогноз не представлен как полученный доход.
- **Уровень:** `integration`.

#### AC-030

- **Дано:** Есть RUB-бюджет, будущая зарплата, обязательный платёж, резерв цели и USDT на другом счёте.
- **Когда:** Рассчитываются лимиты на оставшиеся дни месяца.
- **Тогда:** Доступный RUB-лимит исключает будущую зарплату, USDT, долг и резервы; прогноз учитывает даты поступлений и показывает кассовые разрывы; общий предел не размножается по категориям.
- **Уровень:** `integration`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-052

- **Дано:** Подготовлен месяц с несколькими валютами, переводом, расходами, целью и неполным источником.
- **Когда:** Владелец открывает дашборд и раскрывает показатели.
- **Тогда:** Итоги согласованы с учётом; видны состав, фильтры, валюты, свежесть и неполнота; скрытого двойного учёта нет.
- **Уровень:** `end-to-end`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

#### AC-055

- **Дано:** Владелец проверяет день с расходами, чеком, уточнением, бюджетом и целью.
- **Когда:** Участник проходит сценарий в реальных Chrome и Arc при 1280×720 и 1440×900 CSS px, затем увеличивает масштаб до 200%.
- **Тогда:** Основные действия доступны без потери данных и горизонтального прокручивания форм; измерено фактическое время сценария относительно личного ориентира до 45 минут в день.
- **Уровень:** `manual`.

#### AC-074

- **Дано:** RUB-сумма известна, отсутствует исторический BTC-кросс.
- **Когда:** Владелец выбирает отчёт в BTC.
- **Тогда:** Нативный факт сохранён и доступен; эквивалент/общий итог обозначен неполным; текущий курс не подставлен вместо исторического.
- **Уровень:** `unit+end-to-end`.

#### AC-075

- **Дано:** Один сценарий ввода чека и исправления категории доступен на двух языках.
- **Когда:** Сценарий выполняется с клавиатурой в Chrome и Arc на macOS в обоих контрольных размерах и при увеличении масштаба.
- **Тогда:** Все обязательные поля и ошибки доступны; переключение языка не сбрасывает ввод; суммы локализуются только при отображении.
- **Уровень:** `end-to-end+manual`.

#### AC-080

- **Дано:** Зарплата поступила на счёт A, общая аренда оплачена B, у A нет доступного остатка.
- **Когда:** Строятся семейный бюджет, персональные расходы и обеспеченность по валютам.
- **Тогда:** Доход общий с сохранением получателя; аренда учтена в семье один раз и в личных видах по долям. Доступность семьи включает средства обоих без автоматического обмена валют и без кредитного лимита.
- **Уровень:** `integration`.

#### AC-083

- **Дано:** Есть личные цели A и B, общая цель RUB 600000 и виртуальный резерв RUB 10000.
- **Когда:** Оба открывают личные и семейный виды, увеличивают разрешённый резерв и связывают выделенный счёт.
- **Тогда:** Общая цель показана целиком в общем блоке; персональные половины не создаются. Резерв уменьшает семейную доступность один раз; нет автоматического распределения свободных средств на цели.
- **Уровень:** `end-to-end`.

#### AC-084

- **Дано:** Осталось 10 дней, K=RUB 1000, положительные персональные остатки A=3000 и B=1000.
- **Когда:** Рассчитаны доступные лимиты; затем меняется плательщик общей покупки или дата ожидаемого дохода.
- **Тогда:** Семейный лимит 100/день, A 75, B 25; суммы не дублируют K. Плательщик не меняет доли; перенос дохода меняет прогноз, не доступный остаток. Неизвестное назначение не скрывает факт расхода.
- **Уровень:** `integration`.

#### AC-097

- **Дано:** Доступны все SCR-001–SCR-035.
- **Когда:** Проверяется порядок ответа, действия, объяснения и детализации.
- **Тогда:** Главные суммы подписаны по смыслу; обзор начинает с доступного лимита и обязательств. Прогноз/факт/резерв различаются. Источник, ID и аудит раскрываются по запросу; график ведёт к объясняющим операциям.
- **Уровень:** `manual+e2e`.

#### AC-038

- **Дано:** Два сервиса дают разные bid/ask и один не раскрывает комиссию.
- **Когда:** Владелец сравнивает RUB → USDT.
- **Тогда:** Показаны сопоставимые направления и свежесть; неизвестная комиссия не считается нулевой; недоступная котировка не заменяется обещанием рыночного курса.
- **Уровень:** `integration`.

#### AC-021

- **Дано:** Есть утверждённый бюджет и предложение перераспределения.
- **Когда:** Приходит новый расход, затем уполномоченный участник подтверждает предложенное изменение.
- **Тогда:** До подтверждения план неизменен; подтверждение применяет показанную версию предложения один раз; устаревшее предложение пересогласуется.
- **Уровень:** `integration`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-web FILTER=dashboard && make e2e SCENARIO=monthly-dashboard
```

Dashboard totals воспроизводят доменные отчёты в RU/EN и всех валютах; состояния stale/partial видны.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Provide a monthly summary with verifiable details and currencies.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-7.2`, `task-7.4`, `task-7.5`, `task-6.8`, `task-5.5`, `task-7.9`.

**Kind:** `implementation`.

### Change and contracts

Show plan/actuals, net worth, income/expense categories/merchants, daily available/forecast limits, goals and insights. Every aggregate has drill-down and source/time/coverage; future income and revaluation are separate. Equivalent total allowances are informational, not cross-currency liquidity promises; incompleteness stays visible.

### Change boundaries

- `web/src/features/dashboard/`

### Screen contract

### SCR-006 — Overview

`/overview`

**Question:** How much can I spend and cover obligations?

**Primary answer:** Available daily allowance by currency and next shortfall risk.

**Top-down structure:** Allowance/next payments → plan/actual → attention → goals → funds. Household/member, month and currency stay visible.

**Next action:** Explain amount → SCR-017; obligations → SCR-016; clarification → SCR-025.

**Explanation and details:** Formula/sources/period/history → transactions; forecast separate from available, current revaluation separate.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14, UISTATE-15, UISTATE-16.

### SCR-017 — Daily allowances

`/plan/limits`

**Question:** How much today and why?

**Primary answer:** Available and forecast variants by currency/category/member.

**Top-down structure:** Available today → separate forecast → remaining days → obligations/reserves/actual → personal allocation.

**Next action:** Expand formula, change own plan SCR-015 or reserve SCR-019.

**Explanation and details:** Personal allowances sum to no more than household cap; shortfall explicit and missing inputs never become zero.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14.

### SCR-020 — Income and expenses

`/analytics`

**Question:** What changed in spending and why?

**Primary answer:** Main deviations and comparable-period comparison.

**Top-down structure:** Conclusion/period/currency/household → plan vs actual and change → categories/merchants/items → chart and table.

**Next action:** Select bar/category → SCR-009 with the same filters.

**Explanation and details:** Refunds restate purchase; transfer principal excluded; incomplete-history periods labelled.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

### SCR-023 — Rates and valuation

`/analytics/rates`

**Question:** Which rate valued the money?

**Primary answer:** Rate source/date/direction and applicability.

**Top-down structure:** Pair/date → reference rate → available provider buy/sell quotes → known fees/limitations.

**Next action:** Inspect historical valuation or linked accounts/transactions; refresh reference data.

**Explanation and details:** No executable-exchange promise; missing quote is not USDT/USD parity; actual transaction rate separate.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

### SCR-026 — Insights

`/insights`

**Question:** What should change and what supports the suggestion?

**Primary answer:** Brief observation with impact magnitude and verifiable data.

**Top-down structure:** Important findings → reason/period/comparison → proposal → evidence.

**Next action:** Inspect transactions SCR-009; accept authorized change FORM-12 or dismiss.

**Explanation and details:** Incomplete data/limitations labelled; forecast is no promise and AI never silently edits an approved plan.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-12 — AI response and proposal

**Fields:** Question/proposal expectedRevision, transaction subjectExpectedRevision and exactly choiceId or free-text answer; explicit apply/reject for a proposal.

**Validation and permissions:** Either member for transactions, owner only for personal plan/goal; text cannot change actor. Concurrent response checks revision; preview before financial change.

**Outcome:** Command confirmed, rejected, stale or pending; response/reason and author retained.

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
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-014:** Category, subcategory, merchant and receipt item are separate analytical dimensions.
- **REQ-020:** AI income/expense insights reference verifiable data and separate forecasts from facts.
- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-030:** Daily limits show household and individual available/forecast allowances, by category and with separate funding in each currency.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-038:** Exchange quotes include direction, provider, timestamp, applicable amount and known fees.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-052:** The dashboard combines accounts, plan/actuals, income, expenses, goals and daily limits with drill-down.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-069:** Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.
- **REQ-070:** Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.
- **REQ-080:** Each screen answers a user question and leads to a useful next action.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-003

- **Given:** A current valuation exists for every required pair.
- **When:** The member switches RUB to USD, USDT, USDC, BTC and ETH.
- **Then:** Equivalent totals change while original account and transaction amounts remain unchanged.
- **Level:** `end-to-end`.

#### AC-010

- **Given:** A RUB 1,000 expense was paid in August.
- **When:** A linked RUB 400 partial refund is received in September.
- **Then:** August expense becomes RUB 600; the RUB +400 cash movement stays in September; the recalculation and link are auditable.
- **Level:** `integration`.

#### AC-014

- **Given:** A receipt contains milk from a fictional store; another purchase is from a restaurant.
- **When:** The owner filters expenses and corrects a category.
- **Then:** Expense type, merchant and item can be analyzed independently; owner corrections persist.
- **Level:** `end-to-end`.

#### AC-020

- **Given:** Category expenses are known for two months and one source is stale.
- **When:** AI generates a monthly insight.
- **Then:** Figures are reproducible from reports; period, linked transactions/aggregates and incompleteness are shown; forecast income is not presented as received.
- **Level:** `integration`.

#### AC-030

- **Given:** There is a RUB budget, future salary, an obligation, a goal reservation and USDT in another account.
- **When:** Limits are calculated for the remaining days of the month.
- **Then:** Available RUB allowance excludes future salary, USDT, debt and reservations; the forecast uses receipt dates and shows cash shortfalls; the overall ceiling is not duplicated across categories.
- **Level:** `integration`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-052

- **Given:** A month includes multiple currencies, a transfer, expenses, a goal and an incomplete source.
- **When:** The owner opens the dashboard and drills into metrics.
- **Then:** Totals reconcile to accounting; composition, filters, currencies, freshness and incompleteness are visible; no hidden double counting occurs.
- **Level:** `end-to-end`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

#### AC-055

- **Given:** The owner reviews a day containing expenses, a receipt, clarification, budget and goal.
- **When:** A member completes the flow in actual Chrome and Arc at 1280×720 and 1440×900 CSS px, then zooms to 200%.
- **Then:** Core actions work without data loss or horizontally scrolling forms; observed flow time is recorded against the owner's up-to-45-minutes/day target.
- **Level:** `manual`.

#### AC-074

- **Given:** A RUB amount is known but its historical BTC cross-rate is missing.
- **When:** The owner selects BTC reporting.
- **Then:** Native actuals remain available; equivalent/total is marked incomplete; a current rate is not substituted for the historical rate.
- **Level:** `unit+end-to-end`.

#### AC-075

- **Given:** The same receipt-entry/category-correction flow exists in both languages.
- **When:** The flow runs with a keyboard in Chrome and Arc on macOS at both reference sizes and with zoom.
- **Then:** Required fields and errors remain accessible; language switching preserves input; amounts are localized only for display.
- **Level:** `end-to-end+manual`.

#### AC-080

- **Given:** Salary arrived in A’s account, B paid joint rent and A has no available balance.
- **When:** The household budget, individual expenses and currency funding are calculated.
- **Then:** Income is pooled with recipient retained; rent appears once for the household and by shares in individual views. Household availability includes both members’ funds without automatic currency exchange or credit limits.
- **Level:** `integration`.

#### AC-083

- **Given:** There are personal goals of A and B, a joint RUB 600,000 goal and a RUB 10,000 virtual reserve.
- **When:** Both open individual and household views, increase an authorized reserve and link a dedicated account.
- **Then:** The joint goal appears whole in the shared block; personal halves are not created. The reserve reduces household availability once; free funds are not automatically allocated to goals.
- **Level:** `end-to-end`.

#### AC-084

- **Given:** 10 days remain, K=RUB 1,000 and positive individual remainders are A=3,000 and B=1,000.
- **When:** Available allowances are calculated; then a joint purchase payer or expected-income date changes.
- **Then:** Household allowance is 100/day, A 75, B 25; totals do not duplicate K. Payer does not change shares; rescheduling income changes forecast, not available funds. Unknown attribution does not hide actual expense.
- **Level:** `integration`.

#### AC-097

- **Given:** All SCR-001–SCR-035 are available.
- **When:** Answer, action, explanation and detail order is inspected.
- **Then:** Primary amounts have meaningful labels; overview starts with available allowance and obligations. Forecast/actual/reserve are distinct. Source, IDs and audit are expandable; charts lead to explanatory transactions.
- **Level:** `manual+e2e`.

#### AC-038

- **Given:** Two providers have different bid/ask quotes and one omits its fee.
- **When:** The owner compares RUB → USDT.
- **Then:** Directions and freshness are comparable; an unknown fee is not treated as zero; an unavailable quote is not replaced by a promise of market execution.
- **Level:** `integration`.

#### AC-021

- **Given:** An approved budget and a reallocation proposal exist.
- **When:** A new expense arrives and an authorized member later confirms the proposal.
- **Then:** The plan stays unchanged until confirmation; confirmation applies the displayed proposal version once; a stale proposal must be reconfirmed.
- **Level:** `integration`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

### Verification

```sh
make test-web FILTER=dashboard && make e2e SCENARIO=monthly-dashboard
```

Dashboard totals reproduce domain reports in RU/EN and all currencies; stale/partial states are visible.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
