<!-- want-keep-task: task-7.14 -->
# task-7.14 — Создать настройки и состояние учёта / Create settings and accounting health screens

## RU

Настроить учёт и понять сохранность данных.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-7.9`, `task-7.8`, `task-8.2`, `task-2.6`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Desktop-оболочка, отдельный вход RU/EN, дизайн-система и backend семейных прав включены в базу. Их наличие не доказывает готовность всех экранов и сценариев этой карточки.

### Требуемое поведение

SCR-031–SCR-035: личные язык/валюта/уведомления, семья/приглашение, собственные passkey/recovery/устройства, категории/продавцы/правила с preview, состояние AI/источников/курсов/лимитов затрат и возраст последней Mac-копии. Диагностика раскрывается отдельно. Экран не исполняет восстановление БД и не выдаёт партнёру чужие средства доступа; ссылки ведут к существующим действиям подключения/безопасности. В личных настройках отдельное отключение декоративных эффектов; системное reduced motion имеет приоритет.

### Ключевые контракты

Settings/Household/Security/Categories/Rules/SystemStatus; собственные средства доступа, персональные эффекты и возраст Mac-копии.

### Вне объёма

Только desktop web на ноутбуке с macOS; мобильные экраны не добавлять. Финансовые расчёты и проверка полномочий остаются на сервере. Не расширять согласованный продуктовый объём.

### Блокирующие зависимости

- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): открыта; блокирует выполнение.
- [#55](https://github.com/andrey-krasheninnikov/want-keep/issues/55) (`task-7.8`): открыта; блокирует выполнение.
- [#64](https://github.com/andrey-krasheninnikov/want-keep/issues/64) (`task-8.2`): открыта; блокирует выполнение.
- [#22](https://github.com/andrey-krasheninnikov/want-keep/issues/22) (`task-2.6`): выполнена.

### Границы изменений

- `web/src/features/settings/`
- `web/src/features/system/`

### Экранный контракт

### SCR-031 — Настройки

`/settings`

**Вопрос:** Как настроить удобный учёт?

**Главный ответ:** Личные предпочтения и понятные разделы настроек.

**Структура сверху вниз:** Язык/валюта → уведомления → семья → безопасность → категории → состояние системы.

**Следующее действие:** Сохранить FORM-14; перейти SCR-032/033/034/035.

**Объяснение и детализация:** Смена языка/валюты не меняет финансовый факт или права; переключателя темы нет.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

### SCR-032 — Семья

`/settings/household`

**Вопрос:** Кто в семье и что каждый может?

**Главный ответ:** Два отдельных участника с прозрачными правами.

**Структура сверху вниз:** Состав/приглашение → описание общего доступа → личные/общие ресурсы и правила.

**Следующее действие:** Если есть место, выдать/перевыпустить приглашение после собственного passkey-подтверждения до 5 минут; отозвать действующей сессией. Сначала показать метаданные/revision; секрет повторно не читается.

**Объяснение и детализация:** Нет смены ролей, выхода/замены участника и восстановления партнёром в MVP.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: FORM-02.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16.

### SCR-033 — Безопасность

`/settings/security`

**Вопрос:** Как сохранить мой доступ?

**Главный ответ:** Собственные passkey, recovery и активные устройства.

**Структура сверху вниз:** Средства входа → запасной доступ → устройства/сессии → последствия отзыва.

**Следующее действие:** Добавить passkey, обновить коды, отозвать своё устройство FORM-14.

**Объяснение и детализация:** Recovery-коды показываются только в защищённом собственном потоке, не доступны партнёру.

**Права:** Только собственные средства доступа и сессии.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

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

### SCR-035 — Состояние учёта и копий

`/system`

**Вопрос:** Учёт работает и данные сохранены?

**Главный ответ:** Что работает, насколько свежа копия и какие действия нужны.

**Структура сверху вниз:** Критичные проблемы → последняя успешная копия/возраст → AI/лимит затрат → источники/курсы → диагностика.

**Следующее действие:** Открыть проблемное SCR-028/025, выполнить инструкции подключения Mac из runbook; обновить статус.

**Объяснение и детализация:** Условный RPO равен свежести копии; offline Mac не означает новую копию. Нет кнопки восстановления БД; это отдельная операторская процедура.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14, UISTATE-15.

#### FORM-02 — Начало семьи и приглашение

**Поля:** Имя участника/семьи, язык, таймзона/валюта; закрытое приглашение, имя второго участника и passkey.

**Проверки и права:** Операторский bootstrap однократен. Приглашение одно, случайное, на 24 часа; выдача/перевыпуск требуют собственной auth до 5 минут и expectedRevision. Отзыв с CSRF не требует свежей auth. Принятие проверяет browser/purpose/revision, inviter membership и лимит атомарно с новым passkey/сессией/кодами.

**Результат:** Создано членство, показаны личные recovery-коды; далее onboarding. Секреты не в URL журналов/аналитики.

#### FORM-14 — Личные настройки и безопасность

**Поля:** Язык, валюта отображения, push; имя passkey, отзыв своего устройства, перевыпуск собственных recovery-кодов.

**Проверки и права:** Только собственная безопасность; опасные изменения требуют reauthentication по auth-контракту. Не удалять последний путь входа без замены. Запрет push не блокирует in-app.

**Результат:** Персональные предпочтения сохранены; отозванная сессия/подписка перестаёт работать, коды не попадают в чат.

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
- **UISTATE-15 — Нужен банковский вход:** Назвать подключение и владельца, дать ему безопасно войти; партнёру показать ожидание без доступа к секрету.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.
- **UISTATE-17 — Отмена:** Объяснить отсутствие нового подтверждённого результата, дать повторить явно; не выдавать отмену системного passkey за поломку.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-003:** Общую валюту отображения можно переключать между RUB, USD, USDT, USDC, BTC и ETH.
- **REQ-014:** Категория, подкатегория, продавец и позиция чека являются отдельными аналитическими признаками.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-049:** Каждый участник входит со своими passkey и одноразовыми кодами восстановления; сброс чужого входа партнёром недоступен.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-053:** Напоминания и сводки доступны внутри приложения и через разрешённый web-push.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-057:** Зашифрованная резервная копия выгружается на MacBook ежечасно при его доступности; восстановление проверяется.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-074:** Изменения плана и целей уведомляют второго участника; прочтение и push-подписки принадлежат конкретному пользователю.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-082:** Экранные состояния объясняют последствия и безопасный следующий шаг без потери ввода.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-014

- **Дано:** Чек содержит молоко, продавец — условный магазин; другая покупка — ресторан.
- **Когда:** Владелец фильтрует расходы и исправляет категорию.
- **Тогда:** Доступны независимые срезы по виду расхода, продавцу и товару; пользовательская правка сохраняется.
- **Уровень:** `end-to-end`.

#### AC-049

- **Дано:** Оба участника зарегистрировали собственные passkey и личные коды восстановления.
- **Когда:** Участник восстанавливает свой вход, повторяет код, пробует чужой origin и сброс входа партнёра.
- **Тогда:** Свой вход восстановлен после нового passkey с атомарным отзывом своих старых ключей, recovery-кодов, сессий и подписок; доступ партнёра сохранён. Повтор кода, чужой origin и сброс чужого входа отклонены.
- **Уровень:** `end-to-end`.

#### AC-053

- **Дано:** Есть обязательный платёж, дневная сводка и ошибка синхронизации.
- **Когда:** Наступает время уведомления; push разрешён, затем отозван.
- **Тогда:** Внутренние уведомления сохраняются; разрешённый push отправляется без дублей; отзыв push не отключает внутренний канал; детали денег по умолчанию не раскрываются на экране блокировки.
- **Уровень:** `end-to-end+manual`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

#### AC-057

- **Дано:** Есть база, вложения и MacBook, который временно недоступен.
- **Когда:** Создаются копии, Mac возвращается в сеть, затем проводится восстановление.
- **Тогда:** Показан возраст последней полной копии; после возвращения копирование возобновляется; восстановлены согласованные данные и вложения до четырёх часов; часовой RPO заявляется только при доступном Mac.
- **Уровень:** `integration+manual`.

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-088

- **Дано:** Оба имеют уведомления и устройства, A изменяет разрешённую цель.
- **Когда:** B читает уведомление; A восстанавливает вход и отзывает своё устройство.
- **Тогда:** Событие доставлено B один раз; read-state и отзыв устройства одного не меняют подписки другого. Уточнения личного плана направлены его владельцу, остальные доступны обоим.
- **Уровень:** `end-to-end`.

#### AC-099

- **Дано:** Есть загрузка, пустой список/поиск, устаревшие/частичные данные, offline, отказ и конкурирующие правки.
- **Когда:** Пользователь выполняет чтение или сохранение.
- **Тогда:** Неизвестное не становится нулём, подтверждение даётся после readback; неизвестный исход проверяется по ID команды до повторного создания. Конфликт сохраняет ввод и предлагает сравнение. Истечение сессии ведёт к входу, банковская reauth — к нужному владельцу, ожидание AI не блокирует обычный учёт.
- **Уровень:** `manual+e2e`.

#### AC-003

- **Дано:** Для всех необходимых пар есть актуальная оценка.
- **Когда:** Участник переключает RUB на USD, USDT, USDC, BTC и ETH.
- **Тогда:** Меняется эквивалент итогов, исходные суммы операций и счетов сохраняются.
- **Уровень:** `end-to-end`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-019

- **Дано:** AI предлагает сумму, противоречащую источнику, и связь с несколькими кандидатами.
- **Когда:** Приложение проверяет предложения.
- **Тогда:** Противоречивое изменение отклонено, неоднозначность поступает в очередь уточнений; категории и подтверждённые связи могут применяться автоматически.
- **Уровень:** `integration`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

#### AC-056

- **Дано:** Выбран конкретный тариф, регион, валюта счёта и налоги.
- **Когда:** Проверяется эксплуатационная смета для сотен операций в месяц.
- **Тогда:** Зафиксирована датированная полная смета сервера не выше $40; OpenAI учитывается отдельно до $50; обязательный платный источник остаётся блокером.
- **Уровень:** `manual`.

### Проверка результата

```sh
make e2e SCENARIO=settings-health
```

Настройки персональны, правила имеют preview и историю, Mac offline показывает фактическую свежесть/RPO, неизвестные статусы не выглядят успехом.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Configure accounting and understand data preservation.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-7.9`, `task-7.8`, `task-8.2`, `task-2.6`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

The desktop shell, separate RU/EN sign-in, design system and household-permission backend are included in the base. Their presence does not establish readiness of every screen and scenario in this task.

### Desired behavior

SCR-031–SCR-035: personal language/currency/notifications, household/invite, own passkeys/recovery/devices, categories/merchants/rules with preview, AI/source/rate/spend-limit health and last Mac backup age. Diagnostics are expandable. The screen neither restores the database nor exposes partner credentials; links lead to existing connection/security actions. Personal settings include decorative effects off; system reduced motion takes precedence.

### Key interfaces

Settings/Household/Security/Categories/Rules/SystemStatus; personal access credentials, decorative-effect preferences and Mac-backup age.

### Out of scope

Desktop web on a macOS laptop only; do not add mobile screens. Financial calculations and authority checks remain server-side. Do not expand the agreed product scope.

### Blocked by

- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): open; blocks execution.
- [#55](https://github.com/andrey-krasheninnikov/want-keep/issues/55) (`task-7.8`): open; blocks execution.
- [#64](https://github.com/andrey-krasheninnikov/want-keep/issues/64) (`task-8.2`): open; blocks execution.
- [#22](https://github.com/andrey-krasheninnikov/want-keep/issues/22) (`task-2.6`): completed.

### Change boundaries

- `web/src/features/settings/`
- `web/src/features/system/`

### Screen contract

### SCR-031 — Settings

`/settings`

**Question:** How do I configure comfortable accounting?

**Primary answer:** Personal preferences and clear settings groups.

**Top-down structure:** Language/currency → notifications → household → security → categories → system health.

**Next action:** Save FORM-14; open SCR-032/033/034/035.

**Explanation and details:** Changing language/currency changes neither financial fact nor authority; no theme toggle.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

### SCR-032 — Household

`/settings/household`

**Question:** Who belongs and what can each do?

**Primary answer:** Two distinct members with transparent permissions.

**Top-down structure:** Members/invitation → shared-access explanation → personal/shared resources and rules.

**Next action:** When capacity exists, issue/reissue after own passkey confirmation within 5 minutes; revoke with an active session. Show metadata/revision first; never reread the secret.

**Explanation and details:** No role changing, member exit/replacement or partner recovery in MVP.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-02.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16.

### SCR-033 — Security

`/settings/security`

**Question:** How do I preserve my access?

**Primary answer:** Own passkeys, recovery and active devices.

**Top-down structure:** Access methods → backup access → devices/sessions → revocation consequences.

**Next action:** Add passkey, rotate codes, revoke own device FORM-14.

**Explanation and details:** Recovery codes appear only in protected own flow, unavailable to partner.

**Permissions:** Own credentials and sessions only.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

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

### SCR-035 — Accounting and backup health

`/system`

**Question:** Is accounting working and data preserved?

**Primary answer:** What works, how fresh the backup is and what needs action.

**Top-down structure:** Critical issues → last successful backup/age → AI/spend limit → sources/rates → diagnostics.

**Next action:** Open affected SCR-028/025, follow Mac connection runbook instructions; refresh status.

**Explanation and details:** Conditional RPO follows backup age; offline Mac never implies a fresh copy. No database-restore button; separate operator procedure.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14, UISTATE-15.

#### FORM-02 — Household setup and invitation

**Fields:** Member/household name, language, timezone/currency; private invitation, joining member name and passkey.

**Validation and permissions:** Operator bootstrap is one-time. One random invitation lasts 24 hours; issue/reissue requires own authentication within 5 minutes and expectedRevision. Revocation uses CSRF without fresh auth. Acceptance checks browser/purpose/revision, inviter membership and capacity atomically with a new passkey/session/codes.

**Outcome:** Membership created, personal recovery codes shown; proceed to onboarding. Secrets excluded from URL logs/analytics.

#### FORM-14 — Personal preferences and security

**Fields:** Language, display currency, push; passkey name, revoke own device, regenerate own recovery codes.

**Validation and permissions:** Own security only; sensitive changes require auth-contract reauthentication. Do not remove the last access path without replacement. Push denial never blocks in-app.

**Outcome:** Personal preferences saved; revoked session/subscription stops working and codes never enter chat.

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
- **UISTATE-15 — Bank sign-in needed:** Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.
- **UISTATE-17 — Cancelled:** Explain that no new outcome was confirmed and offer explicit retry; cancelled system passkey prompts are not a malfunction.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-003:** The reporting currency can switch among RUB, USD, USDT, USDC, BTC and ETH.
- **REQ-014:** Category, subcategory, merchant and receipt item are separate analytical dimensions.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-049:** Each member signs in with their own passkeys and one-time recovery codes; partner-assisted reset is unavailable.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-053:** Reminders and summaries are available in-app and through authorized web push.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-057:** An encrypted backup is pulled to the MacBook hourly while reachable; recovery is tested.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-074:** Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-082:** Screen states explain consequences and a safe next step without losing input.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-014

- **Given:** A receipt contains milk from a fictional store; another purchase is from a restaurant.
- **When:** The owner filters expenses and corrects a category.
- **Then:** Expense type, merchant and item can be analyzed independently; owner corrections persist.
- **Level:** `end-to-end`.

#### AC-049

- **Given:** Both members enrolled their own passkeys and personal recovery codes.
- **When:** A member recovers their sign-in, reuses a code, tries an alien origin and attempts to reset their partner’s sign-in.
- **Then:** Own access is restored after a new passkey with atomic revocation of own old keys, recovery codes, sessions and subscriptions; the partner’s access remains. Code reuse, alien origin and resetting the partner’s sign-in fail.
- **Level:** `end-to-end`.

#### AC-053

- **Given:** There is an obligation, a daily summary and a sync error.
- **When:** Notification time arrives; push permission is enabled and later revoked.
- **Then:** In-app notifications remain; allowed push is sent without duplicates; revocation does not disable in-app delivery; financial details are hidden on the lock screen by default.
- **Level:** `end-to-end+manual`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

#### AC-057

- **Given:** The database, attachments and a temporarily unreachable MacBook exist.
- **When:** Backups are attempted, the Mac reconnects and recovery is rehearsed.
- **Then:** Last complete backup age is visible; copying resumes after reconnection; consistent data and attachments restore within four hours; hourly RPO is claimed only while the Mac is reachable.
- **Level:** `integration+manual`.

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-088

- **Given:** Both have notifications and devices and A changes an authorized goal.
- **When:** B reads the notification; A recovers sign-in and revokes their device.
- **Then:** B receives the event once; one member’s read state and device revocation do not alter the other’s subscriptions. Personal-plan clarifications target its owner; other clarifications are open to both.
- **Level:** `end-to-end`.

#### AC-099

- **Given:** Loading, empty list/search, stale/partial data, offline, failure and concurrent edits occur.
- **When:** The user reads or saves.
- **Then:** Unknown never becomes zero and success follows readback; unknown outcomes are reconciled by command ID before another creation. Conflicts retain input and offer comparison. Session expiry leads to sign-in, bank reauth to the proper owner, and AI waiting does not block ordinary accounting.
- **Level:** `manual+e2e`.

#### AC-003

- **Given:** A current valuation exists for every required pair.
- **When:** The member switches RUB to USD, USDT, USDC, BTC and ETH.
- **Then:** Equivalent totals change while original account and transaction amounts remain unchanged.
- **Level:** `end-to-end`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-019

- **Given:** AI proposes a source-conflicting amount and a match with several candidates.
- **When:** The application validates the proposals.
- **Then:** The conflicting change is rejected and ambiguity enters the clarification queue; categories and substantiated links may be applied automatically.
- **Level:** `integration`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

#### AC-056

- **Given:** A concrete plan, region, billing currency and taxes are selected.
- **When:** Operating costs are checked for hundreds of monthly transactions.
- **Then:** A dated all-in server estimate is at most $40; OpenAI has a separate $50 cap; a mandatory paid data source remains a blocker.
- **Level:** `manual`.

### Verification

```sh
make e2e SCENARIO=settings-health
```

Settings are personal, rules have preview/history, offline Mac shows actual freshness/RPO and unknown states never appear successful.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
