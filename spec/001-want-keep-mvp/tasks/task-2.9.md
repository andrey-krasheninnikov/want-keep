<!-- want-keep-task: task-2.9 -->
# task-2.9 — Учитывать явные долги и возмещения внутри семьи / Track explicit inter-member debts and reimbursements

## RU

Учитывать явные долги и возмещения внутри семьи.

**Состояние:** Реализованы backend/API явных семейных долгов, частичных и межвалютных погашений, corrections, selective undo и автоматическая инвалидизация изменившихся связей. UI, наличные возмещения без ledger-перевода, банковский IO и production остаются профильным задачам.

**Зависимости:** `task-2.4`, `task-2.8`.

**Тип:** `implementation`.

### Изменение и контракты

Ledger хранит явный версионный долг между двумя разными активными MembershipID; actor берётся из сессии, а optional expense связывается с текущей posted revision без вывода суммы. Состояния open, settled, attention_required и voided не влияют на активы, остатки, доходы и расходы семьи. Погашение связывает существующий posted перевод между личными счетами debtor и creditor; комиссии не входят в principal. Один стабильный transfer или matching group может погасить несколько долгов только в пределах ещё не использованной received principal. Одинаковый актив требует равенства transferAmount и settledAmount; разные активы требуют обе точные native-суммы. Финансовое изменение, exclusion или reversal перевода переводит settlement в stale и восстанавливает долг; текстовая правка не влияет. Изменение связанного расхода переводит долг в attention_required. Corrections и selective undo используют происхождение полей, expectedRevision и A-B-A conflict detection. Результат, immutable revisions, settlement usage, audit, outbox и terminal command outcome фиксируются атомарно.

### Границы изменений

- `backend/internal/ledger/`
- `backend/internal/storage/`
- `backend/internal/delivery/reimbursements/`
- `backend/migrations/019_family_reimbursements.sql`
- `backend/test/integration/family-reimbursements/`
- `api/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-006:** Перевод между счетами семьи, включая счета разных участников, меняет остатки без дохода или расхода по основной сумме.
- **REQ-068:** Взаимный долг учитывается только по явному указанию и не увеличивает активы или расходы семьи.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-006

- **Дано:** Два собственных RUB-счёта и перевод RUB 1 000 с комиссией RUB 10.
- **Когда:** Получены обе стороны перевода в любом порядке.
- **Тогда:** Связана одна операция перевода; основная сумма исключена из доходов/расходов, комиссия RUB 10 учтена один раз.
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

### Проверка результата

```sh
make check
make test-integration AREA=all
make test-family-reimbursements-race
make test-integration AREA=privacy
git diff --check
```

Долг RUB 300 погашается 100+200 без второго расхода; шесть активов сохраняют точность. Cross-asset требует две суммы; replay, конкурентное перепогашение, чужой или неверно направленный перевод не создают эффект. Финансовая правка перевода восстанавливает долг, текстовая правка сохраняет settlement; изменение расхода требует внимания. Проверяются command visibility, CSRF, keyset, миграция, immutable history и права роли.

Task-2.4 и task-2.8 включены в базу; команды реализованы. Доказательства и границы: evidence/task-2.9-family-reimbursements.md. Доказаны backend-части AC-006/082/086; UI, реальный банк и production не подтверждаются.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Track explicit inter-member debts and reimbursements.

**Status:** The backend/API for explicit household debts, partial and cross-asset settlements, corrections, selective undo and automatic invalidation of changed links is implemented. UI, cash reimbursement without a ledger transfer, bank IO and production remain with their owning tasks.

**Dependencies:** `task-2.4`, `task-2.8`.

**Kind:** `implementation`.

### Change and contracts

Ledger stores an explicit versioned debt between two distinct active MembershipIDs; actor comes from the session and an optional expense references its current posted revision without deriving the amount. Open, settled, attention_required and voided states do not affect household assets, balances, income or expenses. Settlement links an existing posted transfer between the debtor and creditor personal accounts; fees are excluded from principal. One stable transfer or matching group may settle several debts only within its unused received principal. Same-asset transferAmount and settledAmount must be equal; cross-asset settlement requires both exact native amounts. A financial edit, exclusion or reversal of the transfer makes the settlement stale and restores the debt; a text edit does not. A linked expense change makes the debt attention_required. Corrections and selective undo use field provenance, expectedRevision and A-B-A conflict detection. Result, immutable revisions, settlement usage, audit, outbox and terminal command outcome commit atomically.

### Change boundaries

- `backend/internal/ledger/`
- `backend/internal/storage/`
- `backend/internal/delivery/reimbursements/`
- `backend/migrations/019_family_reimbursements.sql`
- `backend/test/integration/family-reimbursements/`
- `api/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-006:** Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.
- **REQ-068:** An inter-member debt is recorded only explicitly and does not increase household assets or expenses.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-006

- **Given:** Two owned RUB accounts and a RUB 1,000 transfer with a RUB 10 fee.
- **When:** Both transfer legs arrive in either order.
- **Then:** One transfer is linked; principal is excluded from income/expenses and the RUB 10 fee is counted once.
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

### Verification

```sh
make check
make test-integration AREA=all
make test-family-reimbursements-race
make test-integration AREA=privacy
git diff --check
```

A RUB 300 debt settles through 100+200 without another expense; all six assets retain precision. Cross-asset settlement requires two amounts; replay, concurrent over-settlement and foreign or misdirected transfers create no effect. A financial transfer edit restores debt, a text edit preserves settlement and an expense change requires attention. Command visibility, CSRF, keyset pagination, migration, immutable history and role grants are verified.

Task-2.4 and task-2.8 are included in the base; commands exist. Evidence and boundaries: evidence/task-2.9-family-reimbursements.en.md. Backend portions of AC-006/082/086 are proven; UI, live banking and production are not verified.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
