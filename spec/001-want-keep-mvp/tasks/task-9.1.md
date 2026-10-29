<!-- want-keep-task: task-9.1 -->
# task-9.1 — Провести сквозную приёмку полного MVP / Run end-to-end acceptance of the full MVP

## RU

Собрать доказательства всех требований на полном кандидате.

**Состояние:** Открыта; ready-for-human. Приёмка требует участия пользователей и отдельных live/manual-доказательств. Начало зависит от незакрытых задач и условий ниже; SDD Ready не является блокером.

**Зависимости:** `task-4.1`, `task-4.2`, `task-4.3`, `task-4.4`, `task-4.5`, `task-4.6`, `task-7.3`, `task-7.6`, `task-7.7`, `task-7.8`, `task-8.3`, `task-7.12`, `task-4.7`.

**Тип:** `verification`.

### Изменение и контракты

### Текущее поведение

SDD имеет статус Ready for development; завершённые задачи содержат отдельные доказательства. Это не подтверждает сквозную приёмку полного MVP или эксплуатационную готовность.

### Требуемое поведение

Пройти весь каталог AC: unit/integration/contract/E2E, AI eval и manual evidence шести платформ; проверить browser push, hourly sync, restore и стоимость. Для каждого AC записать статус/evidence; mock не является live proof. Исправления принадлежат владельцам модулей.

### Ключевые контракты

REQ/AC/task/evidence, полный кандидат, все шесть интеграций и task-4.7, desktop UX, push, restore и бюджет эксплуатации.

### Вне объёма

Не расширять MVP и не заменять live/manual-доказательства моками. Исправления выполняются у владельцев поведения; закрытие тикета не заменяет подтверждённый результат приёмки.

### Блокирующие зависимости

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): выполнена.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): выполнена.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): открыта; блокирует выполнение.
- [#32](https://github.com/andrey-krasheninnikov/want-keep/issues/32) (`task-4.4`): открыта; блокирует выполнение.
- [#33](https://github.com/andrey-krasheninnikov/want-keep/issues/33) (`task-4.5`): открыта; блокирует выполнение.
- [#34](https://github.com/andrey-krasheninnikov/want-keep/issues/34) (`task-4.6`): открыта; блокирует выполнение.
- [#50](https://github.com/andrey-krasheninnikov/want-keep/issues/50) (`task-7.3`): открыта; блокирует выполнение.
- [#53](https://github.com/andrey-krasheninnikov/want-keep/issues/53) (`task-7.6`): открыта; блокирует выполнение.
- [#54](https://github.com/andrey-krasheninnikov/want-keep/issues/54) (`task-7.7`): открыта; блокирует выполнение.
- [#55](https://github.com/andrey-krasheninnikov/want-keep/issues/55) (`task-7.8`): открыта; блокирует выполнение.
- [#65](https://github.com/andrey-krasheninnikov/want-keep/issues/65) (`task-8.3`): открыта; блокирует выполнение.
- [#59](https://github.com/andrey-krasheninnikov/want-keep/issues/59) (`task-7.12`): открыта; блокирует выполнение.
- [#94](https://github.com/andrey-krasheninnikov/want-keep/issues/94) (`task-4.7`): открыта; блокирует выполнение.

### Границы изменений

- `spec/001-want-keep-mvp/evidence/acceptance.md`
- `spec/001-want-keep-mvp/verification.md`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

Полные формулировки проверяемого поведения приведены в AC ниже. REQ: `REQ-006`, `REQ-007`, `REQ-008`, `REQ-010`, `REQ-016`, `REQ-018`, `REQ-019`, `REQ-020`, `REQ-024`, `REQ-026`, `REQ-029`, `REQ-030`, `REQ-031`, `REQ-035`, `REQ-036`, `REQ-037`, `REQ-042`, `REQ-043`, `REQ-044`, `REQ-045`, `REQ-046`, `REQ-047`, `REQ-051`, `REQ-052`, `REQ-054`, `REQ-055`, `REQ-056`, `REQ-057`, `REQ-058`, `REQ-060`, `REQ-061`, `REQ-063`, `REQ-064`, `REQ-065`, `REQ-066`, `REQ-067`, `REQ-068`, `REQ-069`, `REQ-070`, `REQ-071`, `REQ-072`, `REQ-073`, `REQ-074`, `REQ-075`, `REQ-076`, `REQ-077`, `REQ-078`, `REQ-079`, `REQ-080`, `REQ-081`, `REQ-082`, `REQ-083`, `REQ-084`, `REQ-085`, `REQ-086`, `REQ-087`, `REQ-088`.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-042

- **Дано:** Подключён разрешённый аккаунт Alfa с продуктами D-47.
- **Когда:** Запрошены текущие счета, остатки, дебетовые карты, операции и кэшбэк.
- **Тогда:** Данные совпадают с source evidence; накопительные счета, вклады и кредиты Alfa исключены. Unknown блокирует только deployment коннектора.
- **Уровень:** `contract+manual`.

#### AC-043

- **Дано:** Подключён разрешённый расчётный счёт ИП в RBO API.
- **Когда:** Запрошены остатки и движения, повторный импорт и intraday no-statements.
- **Тогда:** Данные совпадают с источником; дублей нет, комиссии учтены отдельно. Неизвестный текущий остаток не равен нулю: видны последний подтверждённый остаток, его дата и пробел покрытия.
- **Уровень:** `contract+manual`.

#### AC-044

- **Дано:** Подключён разрешённый личный аккаунт Ozon Банк с дебетовой картой и связанным основным счётом.
- **Когда:** Запрошены остатки, операции и доступные сведения дебетового продукта; та же карта и счёт встречаются в нескольких представлениях.
- **Тогда:** Данные сопоставимы с источником, свидетельство чтения сохранено, карта не удваивает остаток счёта. Недоступность обязательных полей дебетового продукта отмечена явно. Отсутствие кредитки, накоплений или вкладов Ozon не блокирует MVP: эти продукты вне текущего контракта и не показаны как реализованные.
- **Уровень:** `contract+manual`.

#### AC-045

- **Дано:** Безопасно подключён read-only аккаунт Bybit с подтверждёнными контрактами Funding, используемого Easy Earn и P2P.
- **Когда:** Читаются остатки, история с выбранной даты, повторные страницы, Convert, начисление/выплата Earn и P2P; моделируется отказ прав выбранного продукта.
- **Тогда:** Точные native amounts, ID, статусы, комиссии и coverage сопоставимы с источником; Funding и детали не удваивают обмен, комиссию или доход. P2P связывает crypto/fiat с банком либо требует уточнения. Отказ доступа/неполная история явно блокируют соответствующее покрытие; отсутствие Spot/UTA trading, futures, карты, On-Chain/Advanced Earn и иных неиспользуемых продуктов не блокирует.
- **Уровень:** `contract+manual`.

#### AC-046

- **Дано:** Подключён разрешённый личный аккаунт Aifory с RUB-счетами, USDT/TRC-20, ETH/Ethereum и используемой картой USD (D-33); другие продукты не подключены.
- **Когда:** Повторно прочитаны остатки и история, обмен RUB/USDT, ETH-вывод с комиссией, пополнение карты USDT/USD и похожие pending/confirmed оплаты с отдельной fee.
- **Тогда:** Каждый включённый продукт имеет структурированное доказательство чтения; RUB-группа не дублирует дочерние счета, ETH точен. Движения и комиссии учтены один раз по подтверждённым ID/связям/статусам; знак UI и паритет USDT/USD не предполагаются. Неизвестная связь требует уточнения. Пробелы включённых продуктов блокируют адаптер; остальные продукты не требуются, но их движения по выбранным кошелькам не пропускаются.
- **Уровень:** `contract+manual`.

#### AC-047

- **Дано:** Подключён разрешённый личный аккаунт EMCD с кошельком USDT, действующими Grow, существующими картами, включая заблокированную, и историей P2P. Майнинг не использовался ни сейчас, ни ранее.
- **Когда:** Запрошены счета, остатки, операции и необходимые условия продуктов.
- **Тогда:** По каждому включённому продукту подтверждены сопоставимые с источником данные и автоматическое чтение: сводки не дублируют дочерние остатки, начисление/капитализация/выплата не утраивают доход, отказ карты не расход по основной сумме, P2P связан с денежными сторонами без дубля. Неизвестные поля/история отмечены явно; отсутствие контракта выбранных продуктов блокирует адаптер. Майнинг и другие неиспользуемые продукты не требуются.
- **Уровень:** `contract+manual`.

#### AC-052

- **Дано:** Подготовлен месяц с несколькими валютами, переводом, расходами, целью и неполным источником.
- **Когда:** Владелец открывает дашборд и раскрывает показатели.
- **Тогда:** Итоги согласованы с учётом; видны состав, фильтры, валюты, свежесть и неполнота; скрытого двойного учёта нет.
- **Уровень:** `end-to-end`.

#### AC-055

- **Дано:** Владелец проверяет день с расходами, чеком, уточнением, бюджетом и целью.
- **Когда:** Участник проходит сценарий в реальных Chrome и Arc при 1280×720 и 1440×900 CSS px, затем увеличивает масштаб до 200%.
- **Тогда:** Основные действия доступны без потери данных и горизонтального прокручивания форм; измерено фактическое время сценария относительно личного ориентира до 45 минут в день.
- **Уровень:** `manual`.

#### AC-056

- **Дано:** Выбран конкретный тариф, регион, валюта счёта и налоги.
- **Когда:** Проверяется эксплуатационная смета для сотен операций в месяц.
- **Тогда:** Зафиксирована датированная полная смета сервера не выше $40; OpenAI учитывается отдельно до $50; обязательный платный источник остаётся блокером.
- **Уровень:** `manual`.

#### AC-063

- **Дано:** Приходящая сторона USDT 100 уже импортирована; исходящая RUB 9 000 и комиссия ещё отсутствуют.
- **Когда:** Приходят поздняя сторона, исправление комиссии и повтор старой страницы.
- **Тогда:** Состояние ожидания связи сменяется проверенным обменом; доход/расход основной суммы не удваивается, устаревшая комиссия не восстанавливается.
- **Уровень:** `integration`.

#### AC-065

- **Дано:** Покупка USD 10 распределена по двум категориям и оценена в RUB 900.
- **Когда:** Позже возвращено USD 4 за известную позицию; текущий курс иной.
- **Тогда:** Историческая категория уменьшается на исходную стоимость возвращённой части RUB 360; реальные поступления и валютная разница сохраняются отдельно; превышение суммы возвратов блокируется.
- **Уровень:** `unit+integration`.

#### AC-066

- **Дано:** До аренды нет достаточных RUB; будущая зарплата запланирована после даты аренды; есть кредитный лимит.
- **Когда:** Рассчитывается прогноз и доход переносится на более позднюю дату.
- **Тогда:** Показан кассовый разрыв на дату аренды; кредитный лимит не выдаётся за доступные собственные деньги; основной план без решения владельца не меняется.
- **Уровень:** `unit+end-to-end`.

#### AC-069

- **Дано:** Три ответа AI: отказ, JSON вне схемы и корректный ответ на уже изменённую версию.
- **Когда:** Одновременно исчерпывается резерв месячного бюджета.
- **Тогда:** Ни один ответ не даёт неверного финансового эффекта; статусы различимы, платные повторы ограничены; данные и подтверждённые операции не теряются.
- **Уровень:** `integration`.

#### AC-071

- **Дано:** Источник различает gross P&L, net P&L, fee, funding и reward/transfer.
- **Когда:** Одна экономическая операция встречается в нескольких журналах.
- **Тогда:** Происхождение показателей сохранено; комиссия и доход не удваиваются; выбор net/gross подтверждён контрактом.
- **Уровень:** `contract+integration`.

#### AC-073

- **Дано:** MacBook недоступен 10 часов; последующая загрузка вложений обрывается.
- **Когда:** Проверяется статус копии и выполняется восстановление последнего полного набора.
- **Тогда:** Не заявляется часовой RPO; неполный набор не помечен успешным; последний полный набор восстановим и имеет проверяемый manifest.
- **Уровень:** `integration+manual`.

#### AC-075

- **Дано:** Один сценарий ввода чека и исправления категории доступен на двух языках.
- **Когда:** Сценарий выполняется с клавиатурой в Chrome и Arc на macOS в обоих контрольных размерах и при увеличении масштаба.
- **Тогда:** Все обязательные поля и ошибки доступны; переключение языка не сбрасывает ввод; суммы локализуются только при отображении.
- **Уровень:** `end-to-end+manual`.

#### AC-076

- **Дано:** Подготовлен синтетический набор сотен операций со сложными переводами, чеками и эталонными ответами.
- **Когда:** Выполняются AI-eval, нагрузочная проверка и ручной дневной сценарий.
- **Тогда:** Отчёт показывает ошибки, уточнения, латентность, токены/стоимость и время пользователя; финансовые инварианты проходят, бюджет оценивается по измерению; непроверенное качество не объявлено доказанным.
- **Уровень:** `manual+integration`.

#### AC-077

- **Дано:** Два пользователя состоят в одной семье.
- **Когда:** Проверяются схема, авторизация и ограничение членства.
- **Тогда:** Нет полей partner1/partner2 и ветвлений по конкретным пользователям; роли и принадлежность отделены от личности. Выход, замена и новые роли не реализованы.
- **Уровень:** `integration`.

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

#### AC-082

- **Дано:** A оплачивает общий расход RUB 1000 и явно отмечает возмещение RUB 300 от B.
- **Когда:** B переводит 100, затем 200; обе стороны переводов импортируются повторно.
- **Тогда:** Долг уменьшается 300→200→0 один раз; основная сумма переводов не доход/расход. Обычная покупка без указания долг не создаёт; валютное погашение требует явного соответствия сумм.
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

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-088

- **Дано:** Оба имеют уведомления и устройства, A изменяет разрешённую цель.
- **Когда:** B читает уведомление; A восстанавливает вход и отзывает своё устройство.
- **Тогда:** Событие доставлено B один раз; read-state и отзыв устройства одного не меняют подписки другого. Уточнения личного плана направлены его владельцу, остальные доступны обоим.
- **Уровень:** `end-to-end`.

#### AC-089

- **Дано:** Копия содержит двух участников, личные/общие цели и операции с разными авторами.
- **Когда:** Полный набор восстанавливается на изолированном сервере.
- **Тогда:** Суммы, связи и права обоих сохранены; входы не объединены, банковские сессии автоматически не оживают. Восстановление укладывается в измеренную цель RTO.
- **Уровень:** `integration+manual`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-091

- **Дано:** Январский чек: общие товары 600 (50/50), личные A 100, B 300; в феврале доли статьи стали 60/40.
- **Когда:** В феврале возвращаются общие товары на 200.
- **Тогда:** Январь уменьшается семье на 200, каждому на 100 с исходной исторической оценкой; февральская пропорция не меняет январь, cash date возврата остаётся февральской.
- **Уровень:** `integration`.

#### AC-092

- **Дано:** Свободно RUB 1000; оба пытаются зарезервировать по 800 для разрешённых целей.
- **Когда:** Команды исполняются одновременно.
- **Тогда:** Проверка общего доступного остатка и резерв атомарны: проходит максимум одна команда; отказ не уменьшает другой резерв, оба видят актуальный остаток.
- **Уровень:** `integration`.

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

#### AC-094

- **Дано:** Обзор, вход, форма и таблица используют одну дизайн-систему.
- **Когда:** Проверяются фон, поверхности, акценты и состояния.
- **Тогда:** Фон #141414, поверхности #1C1C1C/#252525 и акцент #5962F0 берутся из закреплённого foundation. Статусы, радиусы, клавиатурный фокус и motion следуют общей теме. data-theme=dark и dark установлены до первого кадра. Название и логотип Want Keep сохраняются; светлой темы нет.
- **Уровень:** `manual+e2e`.

#### AC-095

- **Дано:** Есть RU/EN строки, RUB/USD/USDT/USDC/BTC/ETH и длинные точные суммы.
- **Когда:** Проверяются загруженные и недоступные шрифты.
- **Тогда:** Шрифты foundation обслуживаются локально с лицензиями. Кириллица, валютные символы и fallback остаются читаемыми. Таблицы и сопоставляемые суммы используют Manrope с табличными цифрами. Pixelify Sans и прежние локальные font-face удалены; суммы остаются точными строками.
- **Уровень:** `manual+e2e`.

#### AC-096

- **Дано:** Каталог содержит кнопки, поля, диалоги, панели, таблицы и состояния данных.
- **Когда:** Проверяются состояния и реализация примитивов.
- **Тогда:** Foundation/react закреплены одной версией 0.2.1 с lockfile. Используются публичные exports и собранные CSS/шрифты. Base UI/shadcn и дубли примитивов удалены. Сохранены клавиатура, подписи, ref, controlled state, порталы, черновики, ошибки и восстановление фокуса. Финансовые правила и API не входят в общую библиотеку.
- **Уровень:** `manual+e2e`.

#### AC-097

- **Дано:** Доступны все SCR-001–SCR-035.
- **Когда:** Проверяется порядок ответа, действия, объяснения и детализации.
- **Тогда:** Главные суммы подписаны по смыслу; обзор начинает с доступного лимита и обязательств. Прогноз/факт/резерв различаются. Источник, ID и аудит раскрываются по запросу; график ведёт к объясняющим операциям.
- **Уровень:** `manual+e2e`.

#### AC-098

- **Дано:** Пользователь отфильтровал месяц, валюту, участника и список.
- **Когда:** Он открывает детализацию, возвращается, меняет язык и открывает прямую ссылку.
- **Тогда:** Обзор открывается после входа; левое меню содержит Обзор, Деньги, План, Аналитика, Чат, внизу Подключения/Настройки, сверху уведомления. Контекст сохраняется; сервер проверяет текущего автора независимо от фильтра.
- **Уровень:** `manual+e2e`.

#### AC-099

- **Дано:** Есть загрузка, пустой список/поиск, устаревшие/частичные данные, offline, отказ и конкурирующие правки.
- **Когда:** Пользователь выполняет чтение или сохранение.
- **Тогда:** Неизвестное не становится нулём, подтверждение даётся после readback; неизвестный исход проверяется по ID команды до повторного создания. Конфликт сохраняет ввод и предлагает сравнение. Истечение сессии ведёт к входу, банковская reauth — к нужному владельцу, ожидание AI не блокирует обычный учёт.
- **Уровень:** `manual+e2e`.

#### AC-100

- **Дано:** Участник открывает /login в обоих языках.
- **Когда:** Вход ожидает passkey, отменён или завершился ошибкой.
- **Тогда:** Компактные логотип/Want Keep по центру, свободное пространство, одна основная кнопка «Войти с passkey» / «Log in with Passkeys», ненавязчивые язык и помощь. Статус не ломает композицию; помощь ведёт к личному восстановлению. SVG не искажается.
- **Уровень:** `manual+e2e`.

#### AC-101

- **Дано:** Экраны и формы доступны в RU/EN на macOS.
- **Когда:** Проверяются 1280×720 и 1440×900 CSS px, масштаб 100%/200%, клавиатура и уменьшение движения.
- **Тогда:** Нет скрытых действий, обрезанных сумм и горизонтальной прокрутки форм; таблицы при необходимости имеют обозначенную область прокрутки. Контраст обычного текста ≥4.5:1, крупного ≥3:1, значимых границ/фокуса ≥3:1. Фокус видим и возвращается, статусы доступны без цвета. Записаны реальные версии Chrome/Arc/macOS; Chromium CI отдельно.
- **Уровень:** `manual+e2e`.

#### AC-102

- **Дано:** Оба участника получают синтетические данные и задания без подсказок по интерфейсу.
- **Когда:** Они определяют лимит/нехватку, объясняют сумму обзора, вносят чек, меняют доли, резерв цели, устраняют reauth/ожидание AI, входят и восстанавливают доступ.
- **Тогда:** Каждый завершает все семь сценариев, правильно объясняет результат и отсутствие двойного учёта. Фиксируются время, ошибки, затруднения и версии браузеров; ошибочное финансовое понимание или помощь разработчика требуют исправления и повторения затронутого сценария.
- **Уровень:** `manual+e2e`.

#### AC-103

- **Дано:** Есть текущий счёт, кредитка, вклад/Earn/Coinhold и криптопродукт с неполными данными.
- **Когда:** Пользователь открывает каждый вариант SCR-008.
- **Тогда:** Текущий счёт показывает доступно/резерв/блокировки, кредитка долг/ближайший платёж/grace, накопление факт/прогноз/срок, крипто доступные активы/позиции/комиссии. Неизвестные условия помечены, заёмные деньги не объявляются доступным семейным пулом.
- **Уровень:** `manual+e2e`.

#### AC-104

- **Дано:** Включены эффекты: добавление счёта в учёт, подтверждённое пополнение накопления, достижение цели и превышение лимита; есть повторы синхронизации и reduced motion.
- **Когда:** Пользователь получает подтверждённое событие, открывает страницу повторно, отключает эффекты или включает reduced motion.
- **Тогда:** Ракета/огоньки/конфетти/мягкое диско применяются по design.md и ID события, не повторяются от refresh/retry/backfill. Превышение лимита даёт спокойное предупреждение с действием, не награду. Нет вспышек/стробоскопа, блокировки формы или скрытого текста; отключение и reduced motion заменяют эффект статичным подтверждением. Финансовые цифры не анимируются через ложные промежуточные значения.
- **Уровень:** `manual+e2e`.

#### AC-105

- **Дано:** Есть личный счёт A, семейный счёт и запись покупки; B видит их и может исправлять учёт.
- **Когда:** B исправляет покупку и пытается через FORM-03/API изменить владельца или scope счёта A; затем A меняет свой счёт и B меняет семейный.
- **Тогда:** Исправление покупки разрешено, изменение принадлежности личного счёта B отклоняется сервером. Действия текущего владельца и изменение семейного счёта разрешены с revision/audit. Внешний владелец и история операций не изменены, фильтр участника не даёт дополнительных прав.
- **Уровень:** `integration+e2e`.

#### AC-106

- **Дано:** Подключение авторизовано, но provider/host gate неполон либо прошлый admission относится к другой версии binding.
- **Когда:** Участник или scheduler запрашивает sync, либо меняются build, contract, allowlist, config, permission или environment.
- **Тогда:** Если binding уже неполон или устарел, сервер возвращает `provider_not_admitted` без job, collector IO и проводки. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding. Job/result несёт неизменяемые binding и `admissionRevision`; смена binding во время read отменяет работу best effort, а обязательная commit-time revalidation сохраняет stale result в quarantine без source record или проводки.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make check && make test-integration AREA=all && make e2e SCENARIO=all && make eval-ai SUITE=all
```

Каждый AC имеет свежий pass либо конкретный blocker; все функции/платформы нужны для полного MVP, время и стоимость измерены.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Collect evidence for all requirements on the complete candidate.

**Status:** Open; ready-for-human. Acceptance requires user participation and separate live/manual evidence. Open prerequisites and the entry conditions below govern execution; SDD Ready is not a blocker.

**Dependencies:** `task-4.1`, `task-4.2`, `task-4.3`, `task-4.4`, `task-4.5`, `task-4.6`, `task-7.3`, `task-7.6`, `task-7.7`, `task-7.8`, `task-8.3`, `task-7.12`, `task-4.7`.

**Kind:** `verification`.

### Change and contracts

### Current behavior

The SDD is Ready for development and completed tasks carry separate evidence. This does not establish full-MVP end-to-end acceptance or production readiness.

### Desired behavior

Run the full AC catalog: unit/integration/contract/E2E, AI evaluation and manual evidence for six platforms; verify browser push, hourly sync, recovery and cost. Record status/evidence per AC; a mock is not live proof. Fixes belong to owning modules.

### Key interfaces

REQ/AC/task/evidence, the full candidate, all six integrations and task-4.7, desktop UX, push, restore and operating budget.

### Out of scope

Do not expand the MVP or substitute mocks for live/manual evidence. Fixes belong to their behavior owners; closing an issue does not replace a confirmed acceptance outcome.

### Blocked by

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): completed.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): completed.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): open; blocks execution.
- [#32](https://github.com/andrey-krasheninnikov/want-keep/issues/32) (`task-4.4`): open; blocks execution.
- [#33](https://github.com/andrey-krasheninnikov/want-keep/issues/33) (`task-4.5`): open; blocks execution.
- [#34](https://github.com/andrey-krasheninnikov/want-keep/issues/34) (`task-4.6`): open; blocks execution.
- [#50](https://github.com/andrey-krasheninnikov/want-keep/issues/50) (`task-7.3`): open; blocks execution.
- [#53](https://github.com/andrey-krasheninnikov/want-keep/issues/53) (`task-7.6`): open; blocks execution.
- [#54](https://github.com/andrey-krasheninnikov/want-keep/issues/54) (`task-7.7`): open; blocks execution.
- [#55](https://github.com/andrey-krasheninnikov/want-keep/issues/55) (`task-7.8`): open; blocks execution.
- [#65](https://github.com/andrey-krasheninnikov/want-keep/issues/65) (`task-8.3`): open; blocks execution.
- [#59](https://github.com/andrey-krasheninnikov/want-keep/issues/59) (`task-7.12`): open; blocks execution.
- [#94](https://github.com/andrey-krasheninnikov/want-keep/issues/94) (`task-4.7`): open; blocks execution.

### Change boundaries

- `spec/001-want-keep-mvp/evidence/acceptance.md`
- `spec/001-want-keep-mvp/verification.md`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

The AC scenarios below contain the complete verifiable behavior. REQ: `REQ-006`, `REQ-007`, `REQ-008`, `REQ-010`, `REQ-016`, `REQ-018`, `REQ-019`, `REQ-020`, `REQ-024`, `REQ-026`, `REQ-029`, `REQ-030`, `REQ-031`, `REQ-035`, `REQ-036`, `REQ-037`, `REQ-042`, `REQ-043`, `REQ-044`, `REQ-045`, `REQ-046`, `REQ-047`, `REQ-051`, `REQ-052`, `REQ-054`, `REQ-055`, `REQ-056`, `REQ-057`, `REQ-058`, `REQ-060`, `REQ-061`, `REQ-063`, `REQ-064`, `REQ-065`, `REQ-066`, `REQ-067`, `REQ-068`, `REQ-069`, `REQ-070`, `REQ-071`, `REQ-072`, `REQ-073`, `REQ-074`, `REQ-075`, `REQ-076`, `REQ-077`, `REQ-078`, `REQ-079`, `REQ-080`, `REQ-081`, `REQ-082`, `REQ-083`, `REQ-084`, `REQ-085`, `REQ-086`, `REQ-087`, `REQ-088`.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-042

- **Given:** An authorized Alfa account with D-47 products is connected.
- **When:** Current accounts, balances, debit cards, transactions and cashback are requested.
- **Then:** Data matches source evidence; Alfa savings accounts, deposits and credit products are excluded. Unknown blocks only connector deployment.
- **Level:** `contract+manual`.

#### AC-043

- **Given:** An authorized entrepreneur current account is connected through RBO API.
- **When:** Request balances, movements, repeated import and intraday no-statements.
- **Then:** Data matches the source; no duplicates, fees recorded separately. Unknown current balance is not zero: show the last verified balance, its date and the coverage gap.
- **Level:** `contract+manual`.

#### AC-044

- **Given:** An authorized personal Ozon Bank account with a debit card and linked main account is connected.
- **When:** Debit-product balances, transactions and available details are requested; the same card and account appear in several views.
- **Then:** Data matches the source, read evidence is retained and the card does not duplicate its account balance. Unavailable mandatory debit-product fields are explicit. Missing Ozon credit cards, savings or deposits do not block the MVP: these products are outside the current contract and are not presented as implemented.
- **Level:** `contract+manual`.

#### AC-045

- **Given:** A read-only Bybit account is securely connected with verified Funding, used Easy Earn and P2P contracts.
- **When:** Balances, history from the selected date, replayed pages, Convert, Earn accrual/payout and P2P are read; a selected-product permission failure is simulated.
- **Then:** Exact native amounts, IDs, statuses, fees and coverage match the source; Funding and details do not duplicate an exchange, fee or income. P2P links crypto/fiat with the bank or requires clarification. Access failure/incomplete history explicitly blocks the corresponding coverage; absent Spot/UTA trading, futures, card, On-Chain/Advanced Earn and other unused products do not block.
- **Level:** `contract+manual`.

#### AC-046

- **Given:** An authorized personal Aifory account provides RUB accounts, USDT/TRC-20, ETH/Ethereum and the existing USD card (D-33); other products are not connected.
- **When:** Balances/history are read again, including RUB/USDT exchange, ETH withdrawal with a fee, USDT/USD card funding and similar pending/confirmed payments with a separate fee.
- **Then:** Each included product has structured read evidence; RUB groups do not duplicate child accounts and ETH remains exact. Movements/fees count once using verified IDs/links/statuses; neither UI signs nor USDT/USD parity are assumed. Unknown linkage requires clarification. Gaps in included products block the adapter; other products are not required, but their movements through selected wallets are retained.
- **Level:** `contract+manual`.

#### AC-047

- **Given:** An authorized personal EMCD account has a USDT wallet, existing Grow deposits, existing cards including a blocked card, and P2P history. Mining has never been used.
- **When:** Accounts, balances, transactions and required product terms are requested.
- **Then:** Each included product has source-matching data and verified automatic reading: aggregates do not duplicate child balances; accrual/capitalization/payout do not triple income; declined card principal is not an expense; P2P links to monetary legs without duplicates. Unknown fields/history are explicit; missing selected-product contracts block the adapter. Mining and other unused products are not required.
- **Level:** `contract+manual`.

#### AC-052

- **Given:** A month includes multiple currencies, a transfer, expenses, a goal and an incomplete source.
- **When:** The owner opens the dashboard and drills into metrics.
- **Then:** Totals reconcile to accounting; composition, filters, currencies, freshness and incompleteness are visible; no hidden double counting occurs.
- **Level:** `end-to-end`.

#### AC-055

- **Given:** The owner reviews a day containing expenses, a receipt, clarification, budget and goal.
- **When:** A member completes the flow in actual Chrome and Arc at 1280×720 and 1440×900 CSS px, then zooms to 200%.
- **Then:** Core actions work without data loss or horizontally scrolling forms; observed flow time is recorded against the owner's up-to-45-minutes/day target.
- **Level:** `manual`.

#### AC-056

- **Given:** A concrete plan, region, billing currency and taxes are selected.
- **When:** Operating costs are checked for hundreds of monthly transactions.
- **Then:** A dated all-in server estimate is at most $40; OpenAI has a separate $50 cap; a mandatory paid data source remains a blocker.
- **Level:** `manual`.

#### AC-063

- **Given:** The incoming USDT 100 leg is imported; outgoing RUB 9,000 and fee are missing.
- **When:** The late leg, fee correction and replayed old page arrive.
- **Then:** Pending matching becomes a verified exchange; principal is not double-counted and the stale fee is not restored.
- **Level:** `integration`.

#### AC-065

- **Given:** A USD 10 purchase is split across two categories and valued at RUB 900.
- **When:** USD 4 is later refunded for a known item at a different current rate.
- **Then:** The historical category decreases by the refunded original value RUB 360; actual cash receipts and FX difference stay separate; excess cumulative refunds are rejected.
- **Level:** `unit+integration`.

#### AC-066

- **Given:** Available RUB cannot cover rent; salary is scheduled after rent; a credit limit exists.
- **When:** The forecast is calculated and the income date is moved later.
- **Then:** A rent-date cash shortfall is shown; credit is not presented as owned cash and the plan is not changed without an owner decision.
- **Level:** `unit+end-to-end`.

#### AC-069

- **Given:** Three AI outputs are a refusal, off-schema JSON and a valid response to an obsolete version.
- **When:** The monthly spend reservation is exhausted concurrently.
- **Then:** No response produces an incorrect financial effect; states are distinguishable, paid retries are bounded and data/confirmed transactions are retained.
- **Level:** `integration`.

#### AC-071

- **Given:** A source distinguishes gross P&L, net P&L, fee, funding and reward/transfer.
- **When:** One economic event appears in several logs.
- **Then:** Metric provenance is preserved; fees and income are not doubled; net/gross semantics are contract-verified.
- **Level:** `contract+integration`.

#### AC-073

- **Given:** The MacBook is unreachable for 10 hours and the subsequent attachment download is interrupted.
- **When:** Backup status is inspected and the last complete set is restored.
- **Then:** Hourly RPO is not claimed; the partial set is not marked successful; the last complete set is recoverable with a verifiable manifest.
- **Level:** `integration+manual`.

#### AC-075

- **Given:** The same receipt-entry/category-correction flow exists in both languages.
- **When:** The flow runs with a keyboard in Chrome and Arc on macOS at both reference sizes and with zoom.
- **Then:** Required fields and errors remain accessible; language switching preserves input; amounts are localized only for display.
- **Level:** `end-to-end+manual`.

#### AC-076

- **Given:** A synthetic hundreds-of-transactions set includes difficult transfers, receipts and reference answers.
- **When:** AI evaluation, load checks and a manual daily flow run.
- **Then:** The report shows errors, clarifications, latency, tokens/cost and user time; financial invariants pass and cost uses measurements; untested quality is not claimed as proven.
- **Level:** `manual+integration`.

#### AC-077

- **Given:** Two users belong to one household.
- **When:** Schema, authorization and the membership limit are inspected.
- **Then:** There are no partner1/partner2 fields or specific-user branches; roles and ownership are separate from identity. Exit, replacement and new roles are not implemented.
- **Level:** `integration`.

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

#### AC-082

- **Given:** A pays a RUB 1,000 joint expense and explicitly records RUB 300 reimbursement due from B.
- **When:** B transfers 100 and then 200; both legs of the transfers are imported again.
- **Then:** Debt falls 300→200→0 once; transfer principal is not income/expense. An ordinary purchase creates no debt without instruction; cross-currency settlement requires explicit amount mapping.
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

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-088

- **Given:** Both have notifications and devices and A changes an authorized goal.
- **When:** B reads the notification; A recovers sign-in and revokes their device.
- **Then:** B receives the event once; one member’s read state and device revocation do not alter the other’s subscriptions. Personal-plan clarifications target its owner; other clarifications are open to both.
- **Level:** `end-to-end`.

#### AC-089

- **Given:** A backup contains two members, personal/joint goals and transactions by different authors.
- **When:** The complete set is restored onto an isolated server.
- **Then:** Amounts, relationships and both users’ permissions are preserved; sign-ins are not merged and bank sessions do not revive automatically. Recovery meets the measured RTO target.
- **Level:** `integration+manual`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-091

- **Given:** January receipt: joint items 600 (50/50), personal A 100, B 300; February plan shares became 60/40.
- **When:** Joint items worth 200 are returned in February.
- **Then:** January falls by 200 for the household and 100 for each member using original historical valuation; February shares do not rewrite January and refund cash date stays in February.
- **Level:** `integration`.

#### AC-092

- **Given:** RUB 1,000 is free; both attempt to reserve 800 for authorized goals.
- **When:** Commands execute concurrently.
- **Then:** Checking household availability and reserving are atomic: at most one command succeeds; rejection does not reduce another reserve and both see current availability.
- **Level:** `integration`.

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

#### AC-094

- **Given:** Overview, sign-in, form and table use one design system.
- **When:** Backgrounds, surfaces, accents and states are inspected.
- **Then:** Background #141414, surfaces #1C1C1C/#252525 and primary #5962F0 come from the pinned foundation. Status roles, radii, keyboard focus and motion follow the shared theme. Set data-theme=dark and dark before first paint. Keep the Want Keep name and logo; no light theme.
- **Level:** `manual+e2e`.

#### AC-095

- **Given:** RU/EN strings, RUB/USD/USDT/USDC/BTC/ETH and long precise amounts exist.
- **When:** Loaded and unavailable fonts are tested.
- **Then:** Foundation fonts are served locally with their licenses. Cyrillic, currency glyphs and fallback remain legible. Tables and comparable amounts use Manrope tabular figures. Remove Pixelify Sans and the previous local font-face definitions; monetary values remain exact strings.
- **Level:** `manual+e2e`.

#### AC-096

- **Given:** A catalog contains buttons, fields, dialogs, panels, tables and data states.
- **When:** States and primitive implementations are inspected.
- **Then:** Pin foundation/react to matching 0.2.1 with a lockfile. Use public exports and compiled CSS/fonts. Remove Base UI/shadcn and duplicate primitives. Preserve keyboard access, labels, refs, controlled state, portals, drafts, errors and focus restoration. Shared packages contain no financial rules or product APIs.
- **Level:** `manual+e2e`.

#### AC-097

- **Given:** All SCR-001–SCR-035 are available.
- **When:** Answer, action, explanation and detail order is inspected.
- **Then:** Primary amounts have meaningful labels; overview starts with available allowance and obligations. Forecast/actual/reserve are distinct. Source, IDs and audit are expandable; charts lead to explanatory transactions.
- **Level:** `manual+e2e`.

#### AC-098

- **Given:** The user filtered month, currency, member and list.
- **When:** They open details, return, change language and open a direct link.
- **Then:** Overview opens after sign-in; left navigation contains Overview, Money, Plan, Analytics, Chat, lower Connections/Settings and top notifications. Context survives; the server checks the current actor independently of filters.
- **Level:** `manual+e2e`.

#### AC-099

- **Given:** Loading, empty list/search, stale/partial data, offline, failure and concurrent edits occur.
- **When:** The user reads or saves.
- **Then:** Unknown never becomes zero and success follows readback; unknown outcomes are reconciled by command ID before another creation. Conflicts retain input and offer comparison. Session expiry leads to sign-in, bank reauth to the proper owner, and AI waiting does not block ordinary accounting.
- **Level:** `manual+e2e`.

#### AC-100

- **Given:** A member opens /login in either language.
- **When:** Passkey sign-in is waiting, cancelled or failed.
- **Then:** Compact centered logo/Want Keep, whitespace, one primary “Войти с passkey” / “Log in with Passkeys” button, subtle language/help. Status preserves composition; help leads to personal recovery. SVG proportions remain intact.
- **Level:** `manual+e2e`.

#### AC-101

- **Given:** Screens and forms are available in RU/EN on macOS.
- **When:** 1280×720 and 1440×900 CSS px, 100%/200% zoom, keyboard and reduced motion are tested.
- **Then:** No hidden actions, clipped amounts or horizontally scrolling forms; tables have a labelled scroll region when needed. Normal text contrast ≥4.5:1, large text ≥3:1, meaningful boundaries/focus ≥3:1. Focus is visible and restored; states work without color. Actual Chrome/Arc/macOS versions are recorded separately from Chromium CI.
- **Level:** `manual+e2e`.

#### AC-102

- **Given:** Both members receive synthetic data and tasks without interface hints.
- **When:** They identify allowance/shortfall, explain an overview amount, enter a receipt, change shares, reserve for a goal, handle reauth/AI waiting, and sign in/recover.
- **Then:** Each completes all seven flows and correctly explains outcomes and no double counting. Time, mistakes, difficulties and browser versions are recorded; financial misunderstanding or developer assistance requires a fix and repetition of the affected flow.
- **Level:** `manual+e2e`.

#### AC-103

- **Given:** A current account, credit card, deposit/Earn/Coinhold and crypto product have incomplete data.
- **When:** The user opens each SCR-008 variant.
- **Then:** Current account shows available/reserved/blocked, credit shows debt/next payment/grace, savings shows actual/forecast/maturity, crypto shows available assets/positions/fees. Unknown terms are labelled; borrowed funds are not presented as available household pool.
- **Level:** `manual+e2e`.

#### AC-104

- **Given:** Effects are enabled for adding an accounting account, confirmed savings top-up, goal achievement and limit breach; repeated sync and reduced motion occur.
- **When:** The user receives a confirmed event, revisits the page, disables effects or enables reduced motion.
- **Then:** Rocket/sparkles/confetti/soft disco follow design.en.md and event IDs, never replay from refresh/retry/backfill. Limit breach gets a calm actionable warning, not a reward. No flashes/strobe, blocked forms or hidden text; disabled/reduced motion uses a static acknowledgement. Financial figures never animate through false intermediate values.
- **Level:** `manual+e2e`.

#### AC-105

- **Given:** There is A’s personal account, a household account and a purchase; B can read them and correct accounting.
- **When:** B corrects the purchase and uses FORM-03/API to change A’s account owner/scope; A then changes their own account and B changes the household account.
- **Then:** Purchase correction succeeds; B’s personal-account ownership change is rejected server-side. Current-owner actions and household-account changes succeed with revision/audit. External ownership and transaction history remain unchanged; member filter grants no extra authority.
- **Level:** `integration+e2e`.

#### AC-106

- **Given:** A connection is authenticated, but the provider/host gate is incomplete or the prior admission belongs to a different binding revision.
- **When:** A member or scheduler requests sync, or the build, contract, allowlist, configuration, permission or environment changes.
- **Then:** If the binding is already incomplete or stale, the server returns `provider_not_admitted` with no job, collector IO or posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding. Each job/result carries immutable binding and `admissionRevision`; a binding change during a read cancels work best effort, while mandatory commit-time revalidation retains a stale result in quarantine without a source record or posting.
- **Level:** `integration+security`.

### Verification

```sh
make check && make test-integration AREA=all && make e2e SCENARIO=all && make eval-ai SUITE=all
```

Every AC has a fresh pass or precise blocker; all features/platforms are required for the full MVP and time/cost are measured.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
