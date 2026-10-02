<!-- want-keep-task: task-7.3 -->
# task-7.3 — Создать чат с выбором счёта и файлами / Create chat with account selection and files

## RU

Внести расход из текста или чека через один объяснимый диалог.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-7.1`, `task-5.4`, `task-7.9`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать текст, attachment preview, обязательный dropdown счёта для чеков, состояния загрузки/AI/уточнения/пропуска и ссылку на результат операции. Для текстовой команды при неизвестном счёте требовать уточнение; обычный аналитический вопрос счёта не требует. Повтор отправки защищён idempotency, ввод не теряется при смене языка, бюджеты применяются отдельным подтверждением.

### Границы изменений

- `web/src/features/chat/`

### Экранный контракт

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

### SCR-024 — Общий чат

`/chat`

**Вопрос:** Помоги занести и разобраться

**Главный ответ:** Общая переписка и ясный результат каждого ввода.

**Структура сверху вниз:** Переписка с авторами → контекст/уточнения → ввод текста/файла и счёт → результат с ссылкой.

**Следующее действие:** Отправить текст/чек FORM-07; ответить FORM-12; открыть SCR-010/011/025.

**Объяснение и детализация:** Создано/найдено/ожидает/пропущено различимы; секреты сюда не вводятся; ждём AI без ложного success.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-07, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-17.

### SCR-025 — Уточнения

`/chat/clarifications`

**Вопрос:** Что нужно уточнить для правильного учёта?

**Главный ответ:** Конкретный вопрос с безопасными вариантами и контекстом.

**Структура сверху вниз:** Ожидают ответа → причина/покупка → варианты и свободный ответ → preview → статус.

**Следующее действие:** Ответить FORM-12, открыть оригинал SCR-010/011; уже отвеченное показать с автором.

**Объяснение и детализация:** Ответ привязан к revision вопроса и операции. Вариант применяется транзакционно; свободный текст сохраняется с автором и отдельной answer revision и ожидает новую проверку. Конфликт сохраняет ввод; личные бюджет/цели изменяет только владелец.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

#### FORM-07 — Чек и распределение позиций

**Поля:** Фото/PDF, обязательный счёт списания включая наличные; позиции, скидки, категории, personal/shared и доли % или суммы.

**Проверки и права:** Оба участника; лимиты файлов по контракту. Task-2.6 атомарно проверяет позиции, активы и скидки против одной оплаты, распределяет известную общую скидку детерминированно и требует уточнение при неполных данных. OCR/PDF, сопоставление и personal/shared доли выполняют task-5.3/2.4/2.8.

**Результат:** Создано/связано с существующим/ожидает уточнения/документ не подходит с причиной. Одно подтверждённое списание.

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
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.
- **UISTATE-17 — Отмена:** Объяснить отсутствие нового подтверждённого результата, дать повторить явно; не выдавать отмену системного passkey за поломку.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-015:** Для отправки чека требуется счёт списания; фото/PDF остаётся связанным с результатом обработки.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-008

- **Дано:** Расход RUB 300 создан из чата с выбранным счётом.
- **Когда:** Поступают соответствующий чек, банковская операция и повтор той же операции.
- **Тогда:** Расход остаётся RUB 300, все источники связаны; две отдельные покупки одной суммы не объединяются лишь из-за равенства суммы.
- **Уровень:** `integration`.

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

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

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

#### AC-068

- **Дано:** Загружаются повреждённый PDF, неверно обозначенный тип, чрезмерный файл и чек с вредоносным текстом.
- **Когда:** Срабатывают проверка файла и обработка.
- **Тогда:** Файл с ошибкой не проводится; нет выполнения вложенного кода, произвольного скачивания URL или публичного доступа; понятная причина/уточнение видна в чате.
- **Уровень:** `integration`.

#### AC-075

- **Дано:** Один сценарий ввода чека и исправления категории доступен на двух языках.
- **Когда:** Сценарий выполняется с клавиатурой в Chrome и Arc на macOS в обоих контрольных размерах и при увеличении масштаба.
- **Тогда:** Все обязательные поля и ошибки доступны; переключение языка не сбрасывает ввод; суммы локализуются только при отображении.
- **Уровень:** `end-to-end+manual`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
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

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

#### AC-060

- **Дано:** В PDF или описании операции есть инструкция раскрыть ключ либо сделать перевод.
- **Когда:** Документ обрабатывается AI.
- **Тогда:** Инструкция считается данными; секреты и платёжные инструменты недоступны; недопустимая команда отклонена и не меняет учёт.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-web FILTER=chat && make e2e SCENARIO=receipt-chat
```

Сценарии наличных, валидного/невалидного чека, позднего банковского совпадения и AI outage работают на RU/EN.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Record a text/receipt expense through one explainable conversation.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-7.1`, `task-5.4`, `task-7.9`.

**Kind:** `implementation`.

### Change and contracts

Implement text, attachment previews, mandatory receipt account dropdown, upload/AI/clarification/skip states and a result transaction link. Text commands with unknown accounts require clarification; ordinary analytical questions do not. Protect resubmission with idempotency, preserve input across language switching and confirm budget application separately.

### Change boundaries

- `web/src/features/chat/`

### Screen contract

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

### SCR-024 — Shared chat

`/chat`

**Question:** Help me record and understand

**Primary answer:** Shared conversation and a clear outcome for each input.

**Top-down structure:** Messages with authors → context/clarifications → text/file input and account → linked outcome.

**Next action:** Send text/receipt FORM-07; answer FORM-12; open SCR-010/011/025.

**Explanation and details:** Created/found/waiting/skipped distinct; secrets do not belong here; AI waiting never implies success.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-07, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-17.

### SCR-025 — Clarifications

`/chat/clarifications`

**Question:** What needs clarification for correct accounting?

**Primary answer:** A specific question with safe options and context.

**Top-down structure:** Awaiting answer → reason/purchase → choices and free text → preview → status.

**Next action:** Answer FORM-12, open original SCR-010/011; show answered items with author.

**Explanation and details:** Answers bind to question and transaction revisions. A choice applies transactionally; free text preserves its author and distinct answer revision and awaits a new check. Conflicts preserve input; only the owner changes personal budgets/goals.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

#### FORM-07 — Receipt and item allocation

**Fields:** Photo/PDF, required debit account including cash; items, discounts, categories, personal/shared and percentage or amount shares.

**Validation and permissions:** Either member; file limits follow the contract. Task-2.6 atomically validates items, assets and discounts against one payment, allocates a known receipt-wide discount deterministically and requires clarification for incomplete data. Task-5.3/2.4/2.8 own OCR/PDF, matching and personal/shared shares.

**Outcome:** Created/linked to existing/awaiting clarification/document unsuitable with reason. One confirmed debit.

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
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.
- **UISTATE-17 — Cancelled:** Explain that no new outcome was confirmed and offer explicit retry; cancelled system passkey prompts are not a malfunction.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-015:** Receipt submission requires a debit account; the photo/PDF stays linked to the processing result.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-008

- **Given:** A RUB 300 expense was created from chat for a selected account.
- **When:** The matching receipt, bank transaction and duplicate bank delivery arrive.
- **Then:** Expense remains RUB 300 and all evidence is linked; separate equal-amount purchases are not merged merely by amount.
- **Level:** `integration`.

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

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

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

#### AC-068

- **Given:** A corrupt PDF, mislabeled type, oversized file and prompt-injected receipt are uploaded.
- **When:** File validation and processing run.
- **Then:** Invalid files do not post; embedded code, arbitrary URL fetching and public access are unavailable; chat shows an understandable reason or clarification.
- **Level:** `integration`.

#### AC-075

- **Given:** The same receipt-entry/category-correction flow exists in both languages.
- **When:** The flow runs with a keyboard in Chrome and Arc on macOS at both reference sizes and with zoom.
- **Then:** Required fields and errors remain accessible; language switching preserves input; amounts are localized only for display.
- **Level:** `end-to-end+manual`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
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

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

#### AC-060

- **Given:** A PDF or transaction description instructs the agent to reveal a key or transfer funds.
- **When:** AI processes the document.
- **Then:** The instruction is treated as data; secrets and payment tools are unavailable; an invalid command is rejected without changing accounting.
- **Level:** `integration`.

### Verification

```sh
make test-web FILTER=chat && make e2e SCENARIO=receipt-chat
```

Cash, valid/invalid receipt, late bank match and AI outage flows work in RU/EN.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
