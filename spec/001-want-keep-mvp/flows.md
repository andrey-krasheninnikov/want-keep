# Потоки данных

[English](flows.en.md)

## Импорт и AI-проверка

```mermaid
flowchart LR
  P[Платформа] --> G[API gateway / Playwright]
  G --> R[SourceRecord и coverage]
  R --> M[Mapping и domain validation]
  M --> L[Атомарный ledger + outbox]
  L --> A[AI review версии]
  A --> V[Проверка разрешённой команды]
  V --> C[Категория / подтверждённая связь]
  V --> Q[Уточнение]
  C --> D[Проекции и дашборд]
  L --> D
  P --> S[BalanceSnapshot]
  S --> X[Сверка]
  L --> X
```

SourceRecord сохраняется и при неполном mapping. Ledger может работать, пока AI ждёт; неустановленные финансовые значения не проводятся. У дашборда отдельные признаки свежести источника, покрытия и AI-review.

## Чек и поздний импорт

```mermaid
sequenceDiagram
  actor O as Участник
  participant W as Веб-чат
  participant A as Application
  participant AI as OpenAI
  participant L as Ledger
  O->>W: Выбрать счёт и отправить чек
  W->>A: Message + attachmentId + accountId + idempotency
  A->>A: Авторизация и проверка файла
  A->>AI: Разрешённые данные без секретов
  AI-->>A: Структурированное предложение
  alt Не хватает точных данных
    A-->>W: Уточнение конкретных полей
  else Документ не относится к учёту
    A-->>W: Явный пропуск и причина
  else Данные установлены
    A->>L: Создать или связать операцию, expectedRevision
    A-->>W: Результат и ссылка на операцию
  end
  A->>L: Поздний банковский импорт
  L->>L: Точная идемпотентность + проверка экономической связи
  L-->>W: Одна операция, несколько доказательств
```

Равные сумма и время формируют кандидатов, не доказательство единственности. Если найдено несколько покупок, требуется уточнение. Изменение операции во время ответа AI делает старое предложение неприменимым.

## План и дневной лимит

```mermaid
flowchart TD
  F[Подтверждённые деньги по валюте] --> K[Available limit]
  R[Уникальные резервы целей] --> K
  O[Непогашенные обязательства] --> K
  B[Остаток гибкого бюджета] --> K
  E[Доходы по ожидаемым датам] --> Q[Дневной forecast и кассовые разрывы]
  F --> Q
  R --> Q
  O --> Q
  B --> Q
  AI[Предложение AI] --> U[Решение уполномоченного участника]
  U --> B
```

Получение дохода меняет факт; наступление плановой даты само по себе не создаёт деньги. Погашенный платёж снимает соответствующую резервируемую часть обязательства.

## Резервирование на Mac

```mermaid
sequenceDiagram
  participant M as MacBook
  participant S as VPS приложения
  participant P as Managed PostgreSQL
  participant F as Неизменяемые вложения
  participant B as Локальная копия
  M->>S: Авторизованный hourly pull при доступности
  S->>P: Logical dump по private VPC
  S->>F: Inventory по consistent cutoff
  P-->>S: Поток DB
  F-->>S: Вложения + manifest
  S-->>M: Ограниченный поток export
  M->>M: Проверить части и checksum
  M->>B: Атомарно завершить зашифрованный набор
  M->>S: Подтверждение проверенного manifest
  Note over M,P: Недоступный Mac увеличивает возраст последней полной копии
```

Последняя полная копия не заменяется незавершённой. Recovery-ключ должен быть доступен вне единственной аварийной системы. Retention: 48 почасовых, 30 дневных, 8 недельных и 12 месячных точек при cap 20 GiB; последний полный набор не удаляется. RPO до часа условен доступностью Mac и проверкой набора. Восстановление старого набора в чистый managed PostgreSQL не активирует старые банковские сессии автоматически. Подробный контракт и открытые runtime gates: [hosting evidence](evidence/hosting.md).

## Семейные права и разрезы

```mermaid
flowchart LR
  U[Отдельная сессия пользователя] --> M[Членство и scope семьи]
  M --> C[Команда и текущие права]
  C --> L[Журнал: сумма и автор]
  L --> A[Доли участников и категории]
  A --> F[Один семейный факт]
  A --> P[Индивидуальные разрезы]
  F --> K[Общий K по валюте]
  K --> D[Одна матрица лимитов]
  D --> P
```

## Task-2.4: от факта к одному эффекту

```mermaid
flowchart LR
  A[Ручной ввод / проверенная нормализация] --> B[Семейная транзакция и текущие права]
  I[CommitPage: admission / generation / lease] --> B
  B --> C[Идемпотентность D-39]
  C --> D{Доказанная связь?}
  D -->|Да| E[Проверить состав и защиту полей]
  D -->|Только вероятный кандидат| F[Сохранить waiting и evidence без второго эффекта]
  F --> G[Участник: link или separate с revisions]
  G --> E
  E --> H[Решение + journal revisions + проекции + review/outbox]
  H --> J[Результат команды / checkpoint]
  H --> K[История и составной undo]
```

Отдельные исходные статусы и даты остаются в истории. Неполный поиск и ожидание дают matching_unresolved; source observations сохраняются отдельно. Устаревший import job попадает в quarantine до этой цепочки. Банковский IO и распознавание документов подключаются в профильных задачах.

## Task-3.2: входная страница коннектора

```mermaid
sequenceDiagram
  participant J as Проверенное sync job
  participant C as API/Browser collector
  participant E as EvidenceStore
  participant G as Admission gate
  participant A as Accounts/Ledger
  participant Q as Quarantine
  participant R as Restart reconciler
  J->>G: Проверить exact gateway binding
  J->>G: BeforeRead(binding, revision, generation, lease)
  G->>C: Server-issued request без household/actor/internal IDs
  C-->>G: Exact echo + issued cursor + evidence + typed records/coverage
  G->>E: Сохранить raw evidence с server-derived household/job и disposition=staged
  alt provider failure
    G->>Q: Связать evidence с household/job
    G->>G: Атомарно сохранить waiting/retry/failed
  else binding/revision/generation/lease/cursor актуальны
    G->>A: CommitPage: resolve accounts + source revisions + observations/postings
    A-->>G: Audit/outbox/checkpoint + immutable receipt атомарно
    opt Подтверждение commit потеряно
      G->>G: Readback receipt; без доказательства оставить staged
    end
    opt account/source ambiguity
      A->>Q: Evidence + source_ambiguous/transaction_unresolved
      A-->>G: Partial coverage без неподтверждённого эффекта
    end
  else page отклонена после staging
    G->>Q: Durable retention через отдельный lifecycle context
    Q-->>G: Только успех переводит staged в rejected_result
  else результат устарел
    G->>Q: Evidence reference + safe reason
  end
  opt После рестарта disposition остался staged
    R->>E: Прочитать staged batches
    R->>G: Найти terminal receipt по household/job/evidence
    G-->>R: page/provider_outcome/rejected_result/stale_result либо неизвестно
    R->>E: Идемпотентно завершить только доказанный disposition
  end
```

Страница самостоятельна: каждый поддерживаемый счёт, используемый balance или posting, имеет account descriptor в той же странице. Следующая страница повторяет descriptor и исходный cursor; повтор не создаёт account/opening/financial effect. `nextCursor` либо отсутствует, либо непустой. Provider failure также повторяет issued cursor; sync-result boundary отклоняет запоздалый outcome прежней страницы, а общая lease identity продолжает heartbeat после продвижения checkpoint. Ошибка второй страницы не продвигает её cursor, а уже подтверждённая первая страница остаётся зафиксированной с partial coverage. Совокупный набор gaps после объединения с checkpoint ограничен 100 значениями; переполнение откатывает страницу. Неоднозначный счёт или source не превращает страницу в полный успех и не отменяет независимые поддержанные записи. Подтверждённый provider mapping `RUR → RUB` сохраняет raw code в evidence/metadata и использует RUB в финансовом домене. Повтор с новым evidence ID/locator, эквивалентными пустыми optional-полями и тем же нормализованным payload/raw digest не создаёт source revision. Evidence использует канонический base64 без CR/LF. Неизвестный результат commit подтверждается только атомарной receipt; без неё evidence остаётся staged. Restart reconciler завершает disposition только по сохранённой terminal receipt и не повторяет финансовый эффект.

## Task-3.3: browser job

```mermaid
sequenceDiagram
  participant W as Go worker
  participant V as Credentials vault
  participant C as Collector via Unix socket
  participant P as Synthetic/provider portal
  participant E as Encrypted evidence
  participant G as Admission commit fence
  W->>W: Проверить stored job binding/revision/generation/lease
  W->>V: Borrow browser_session
  V-->>W: Plaintext только в памяти job
  W->>C: Capabilities(exact binding/revision)
  W->>W: Сохранить external_started
  W->>C: Read(server-issued request, ephemeral storageState)
  C->>C: Новый BrowserContext + build-owned allowlist
  C->>P: Только exact read или statement POST
  P-->>C: Typed page либо reauth/MFA/CAPTCHA
  C-->>W: Exact SyncResult без session
  W->>E: Зашифровать raw evidence с household/job/page/item AAD
  W->>G: CommitPage или CommitFailure
  alt binding/revision/generation/lease актуальны
    G-->>W: Receipt и terminal disposition
  else результат устарел
    G-->>W: Quarantine без checkpoint и финансового эффекта
  else связь потеряна после external_started
    W-->>W: unresolved без автоматического provider replay
  end
  W->>V: Clear borrowed session
  C->>C: Закрыть BrowserContext
```

Collector не получает actor, household permission, произвольный маршрут или browser script. Вход пользователя в кабинет и provider-specific workflow добавляются task-4.x. Production egress и runtime admission подтверждает task-8.x.

## Task-6.1: курс и оценка

```mermaid
flowchart LR
  A[CBR XML] --> R[Неизменяемые наблюдения]
  B[Frankfurter с фильтром CBR] --> R
  C[CoinGecko Demo] --> R
  R --> X[Точный USD-кросс]
  X --> H[Снимок компонента на дату операции]
  X --> T[Текущий эквивалент остатка]
  H --> V[Отчёт: факт]
  T --> V
  T --> F[Отдельная справочная переоценка]
```

При отсутствии leg сохраняются исходная сумма и причина неполноты. Повторный курс создаёт новую версию наблюдения, но не переписывает исторический снимок. Исполнимая котировка платформы проходит отдельный путь и не выводится из справочного кросса.
