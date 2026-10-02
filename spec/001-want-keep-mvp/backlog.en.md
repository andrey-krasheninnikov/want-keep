# Want Keep MVP backlog

Rendered from [catalog.json](catalog.json). Edit the catalog, then run `python3 spec/001-want-keep-mvp/tools/spec_tool.py render`.

The specification passed task-0.10 and is ready for development. The decision-complete order, parallelism and entry/exit gates are published in [plan.en.md](plan.en.md). Each task still awaits its own dependencies and runtime gates; SDD Ready does not mean an implemented or accepted MVP. Task cards are self-contained in RU/EN.

| Task | Outcome | Dependencies | GitHub |
| --- | --- | --- | --- |
| [task-0.1](tasks/task-0.1.md) | Verify Alfa-Bank read contract | — | [#1](https://github.com/pchkauu/want-keep/issues/1) |
| [task-0.2](tasks/task-0.2.md) | Verify Raiffeisenbank Russia read contract | — | [#2](https://github.com/pchkauu/want-keep/issues/2) |
| [task-0.3](tasks/task-0.3.md) | Verify Ozon Bank read contract | — | [#3](https://github.com/pchkauu/want-keep/issues/3) |
| [task-0.4](tasks/task-0.4.md) | Verify Bybit read contract | — | [#4](https://github.com/pchkauu/want-keep/issues/4) |
| [task-0.5](tasks/task-0.5.md) | Verify Aifory Pro read contract | — | [#5](https://github.com/pchkauu/want-keep/issues/5) |
| [task-0.6](tasks/task-0.6.md) | Verify EMCD read contract | — | [#6](https://github.com/pchkauu/want-keep/issues/6) |
| [task-0.7](tasks/task-0.7.md) | Verify free FX and quote sources | — | [#7](https://github.com/pchkauu/want-keep/issues/7) |
| [task-0.8](tasks/task-0.8.md) | Measure OpenAI quality and cost | — | [#8](https://github.com/pchkauu/want-keep/issues/8) |
| [task-0.9](tasks/task-0.9.md) | Verify infrastructure and server budget | — | [#9](https://github.com/pchkauu/want-keep/issues/9) |
| [task-0.10](tasks/task-0.10.md) | Close contracts and review SDD readiness | task-0.1, task-0.2, task-0.3, task-0.4, task-0.5, task-0.6, task-0.7, task-0.8, task-0.9 | [#10](https://github.com/pchkauu/want-keep/issues/10) |
| [task-1.1](tasks/task-1.1.md) | Create project structure and verification commands | task-0.10 | [#11](https://github.com/pchkauu/want-keep/issues/11) |
| [task-1.2](tasks/task-1.2.md) | Define money types and API contract | task-1.1 | [#12](https://github.com/pchkauu/want-keep/issues/12) |
| [task-1.3](tasks/task-1.3.md) | Create storage and transaction boundaries | task-1.2 | [#13](https://github.com/pchkauu/want-keep/issues/13) |
| [task-1.4](tasks/task-1.4.md) | Implement passkeys and access recovery | task-1.3 | [#14](https://github.com/pchkauu/want-keep/issues/14) |
| [task-1.5](tasks/task-1.5.md) | Protect secrets and private attachments | task-1.4 | [#15](https://github.com/pchkauu/want-keep/issues/15) |
| [task-1.6](tasks/task-1.6.md) | Implement household membership and resource permissions | task-1.4 | [#16](https://github.com/pchkauu/want-keep/issues/16) |
| [task-2.1](tasks/task-2.1.md) | Implement accounts and opening balances | task-1.3, task-1.6 | [#17](https://github.com/pchkauu/want-keep/issues/17) |
| [task-2.2](tasks/task-2.2.md) | Implement the transaction ledger and states | task-2.1, task-1.2 | [#18](https://github.com/pchkauu/want-keep/issues/18) |
| [task-2.3](tasks/task-2.3.md) | Add revisions, corrections and audit | task-2.2 | [#19](https://github.com/pchkauu/want-keep/issues/19) |
| [task-2.4](tasks/task-2.4.md) | Link transfers and prevent duplicates | task-2.3 | [#20](https://github.com/pchkauu/want-keep/issues/20) |
| [task-2.5](tasks/task-2.5.md) | Reconcile the ledger to source balances | task-2.3 | [#21](https://github.com/pchkauu/want-keep/issues/21) |
| [task-2.6](tasks/task-2.6.md) | Separate categories, merchants and items | task-2.3 | [#22](https://github.com/pchkauu/want-keep/issues/22) |
| [task-2.7](tasks/task-2.7.md) | Implement refunds and expense allocation | task-2.2, task-2.6, task-2.8 | [#23](https://github.com/pchkauu/want-keep/issues/23) |
| [task-2.8](tasks/task-2.8.md) | Allocate household expenses and items to members | task-2.6, task-1.6 | [#24](https://github.com/pchkauu/want-keep/issues/24) |
| [task-2.9](tasks/task-2.9.md) | Track explicit inter-member debts and reimbursements | task-2.4, task-2.8 | [#25](https://github.com/pchkauu/want-keep/issues/25) |
| [task-3.1](tasks/task-3.1.md) | Create durable background jobs | task-1.3, task-2.3 | [#26](https://github.com/pchkauu/want-keep/issues/26) |
| [task-3.2](tasks/task-3.2.md) | Define connector ingestion contracts | task-3.1, task-1.2, task-2.1, task-2.2 | [#27](https://github.com/pchkauu/want-keep/issues/27) |
| [task-3.3](tasks/task-3.3.md) | Create an isolated browser collector | task-3.2, task-1.5 | [#28](https://github.com/pchkauu/want-keep/issues/28) |
| [task-4.1](tasks/task-4.1.md) | Implement Alfa-Bank connector | task-0.1, task-3.3, task-2.4, task-2.5 | [#29](https://github.com/pchkauu/want-keep/issues/29) |
| [task-4.2](tasks/task-4.2.md) | Implement Raiffeisenbank Russia connector | task-0.2, task-3.3, task-2.4, task-2.5 | [#30](https://github.com/pchkauu/want-keep/issues/30) |
| [task-4.3](tasks/task-4.3.md) | Implement Ozon Bank connector | task-0.3, task-3.3, task-2.4, task-2.5 | [#31](https://github.com/pchkauu/want-keep/issues/31) |
| [task-4.4](tasks/task-4.4.md) | Implement Bybit connector | task-0.4, task-3.3, task-2.4, task-2.5 | [#32](https://github.com/pchkauu/want-keep/issues/32) |
| [task-4.5](tasks/task-4.5.md) | Implement Aifory Pro connector | task-0.5, task-3.3, task-2.4, task-2.5 | [#33](https://github.com/pchkauu/want-keep/issues/33) |
| [task-4.6](tasks/task-4.6.md) | Implement EMCD connector | task-0.6, task-3.3, task-2.4, task-2.5 | [#34](https://github.com/pchkauu/want-keep/issues/34) |
| [task-5.1](tasks/task-5.1.md) | Create OpenAI gateway and spend control | task-0.8, task-3.1, task-1.5 | [#35](https://github.com/pchkauu/want-keep/issues/35) |
| [task-5.2](tasks/task-5.2.md) | Review every transaction through AI commands | task-5.1, task-2.4, task-2.6, task-2.8, task-2.9 | [#36](https://github.com/pchkauu/want-keep/issues/36) |
| [task-5.3](tasks/task-5.3.md) | Process receipts and line items | task-5.2, task-1.5, task-2.7 | [#37](https://github.com/pchkauu/want-keep/issues/37) |
| [task-5.4](tasks/task-5.4.md) | Implement chat and clarification queue | task-5.3, task-5.2 | [#38](https://github.com/pchkauu/want-keep/issues/38) |
| [task-5.5](tasks/task-5.5.md) | Generate grounded AI insights | task-5.1, task-6.8 | [#39](https://github.com/pchkauu/want-keep/issues/39) |
| [task-6.1](tasks/task-6.1.md) | Implement rates and currency valuation | task-0.7, task-2.2, task-3.2 | [#40](https://github.com/pchkauu/want-keep/issues/40) |
| [task-6.2](tasks/task-6.2.md) | Account for credit cards and grace periods | task-4.1, task-4.2, task-4.3, task-2.2 | [#41](https://github.com/pchkauu/want-keep/issues/41) |
| [task-6.3](tasks/task-6.3.md) | Calculate accruals and savings forecasts | task-4.1, task-4.2, task-4.3, task-4.4, task-4.6, task-6.1 | [#42](https://github.com/pchkauu/want-keep/issues/42) |
| [task-6.4](tasks/task-6.4.md) | Compare dated cash-flow returns | task-6.3 | [#43](https://github.com/pchkauu/want-keep/issues/43) |
| [task-6.5](tasks/task-6.5.md) | Aggregate trading P&L and mining | task-6.1, task-2.4 | [#44](https://github.com/pchkauu/want-keep/issues/44) |
| [task-6.6](tasks/task-6.6.md) | Plan monthly budgets and income | task-2.7, task-2.6, task-6.1, task-6.2, task-2.8 | [#45](https://github.com/pchkauu/want-keep/issues/45) |
| [task-6.7](tasks/task-6.7.md) | Reserve money for goals | task-6.6, task-2.1 | [#46](https://github.com/pchkauu/want-keep/issues/46) |
| [task-6.8](tasks/task-6.8.md) | Calculate daily allowances and liquidity forecast | task-6.6, task-6.7, task-6.3, task-2.5 | [#47](https://github.com/pchkauu/want-keep/issues/47) |
| [task-7.1](tasks/task-7.1.md) | Create the desktop shell and RU/EN sign-in | task-1.4, task-1.2, task-7.11, task-1.6, task-2.1 | [#48](https://github.com/pchkauu/want-keep/issues/48) |
| [task-7.2](tasks/task-7.2.md) | Show accounts, transactions and corrections | task-7.1, task-2.5, task-2.6, task-2.7, task-5.2, task-7.9, task-2.9 | [#49](https://github.com/pchkauu/want-keep/issues/49) |
| [task-7.3](tasks/task-7.3.md) | Create chat with account selection and files | task-7.1, task-5.4, task-7.9 | [#50](https://github.com/pchkauu/want-keep/issues/50) |
| [task-7.4](tasks/task-7.4.md) | Create the monthly budget editor | task-7.1, task-6.6, task-7.9 | [#51](https://github.com/pchkauu/want-keep/issues/51) |
| [task-7.5](tasks/task-7.5.md) | Show goals and reservations | task-7.1, task-6.7, task-7.9 | [#52](https://github.com/pchkauu/want-keep/issues/52) |
| [task-7.6](tasks/task-7.6.md) | Build the dashboard, limits and insights | task-7.2, task-7.4, task-7.5, task-6.8, task-5.5, task-7.9 | [#53](https://github.com/pchkauu/want-keep/issues/53) |
| [task-7.7](tasks/task-7.7.md) | Show credit cards, savings and returns | task-7.2, task-6.2, task-6.3, task-6.4, task-6.5, task-7.9 | [#54](https://github.com/pchkauu/want-keep/issues/54) |
| [task-7.8](tasks/task-7.8.md) | Add notifications and web push | task-7.1, task-6.8, task-5.5, task-1.4, task-3.1, task-7.9 | [#55](https://github.com/pchkauu/want-keep/issues/55) |
| [task-7.9](tasks/task-7.9.md) | Add household context and ownership to the interface | task-7.1, task-1.6 | [#56](https://github.com/pchkauu/want-keep/issues/56) |
| [task-7.10](tasks/task-7.10.md) | Create tokens, typography and brand assets | task-1.1 | [#57](https://github.com/pchkauu/want-keep/issues/57) |
| [task-7.11](tasks/task-7.11.md) | Create the Base UI component catalog | task-7.10 | [#58](https://github.com/pchkauu/want-keep/issues/58) |
| [task-7.12](tasks/task-7.12.md) | Verify desktop UX and visual acceptance | task-7.2, task-7.3, task-7.4, task-7.5, task-7.6, task-7.7, task-7.8, task-7.9, task-7.13, task-7.14, task-7.15 | [#59](https://github.com/pchkauu/want-keep/issues/59) |
| [task-7.13](tasks/task-7.13.md) | Create connection and reauthorization screens | task-7.1, task-7.9, task-3.3, task-4.1, task-4.2, task-4.3, task-4.4, task-4.5, task-4.6 | [#60](https://github.com/pchkauu/want-keep/issues/60) |
| [task-7.14](tasks/task-7.14.md) | Create settings and accounting health screens | task-7.9, task-7.8, task-8.2, task-2.6 | [#61](https://github.com/pchkauu/want-keep/issues/61) |
| [task-7.15](tasks/task-7.15.md) | Add contextual financial-event animations | task-7.11, task-7.2, task-7.4, task-7.5, task-7.14 | [#62](https://github.com/pchkauu/want-keep/issues/62) |
| [task-8.1](tasks/task-8.1.md) | Prepare deployment and system health | task-0.9, task-3.3, task-5.1, task-7.8 | [#63](https://github.com/pchkauu/want-keep/issues/63) |
| [task-8.2](tasks/task-8.2.md) | Pull encrypted backups to the MacBook | task-8.1, task-1.5 | [#64](https://github.com/pchkauu/want-keep/issues/64) |
| [task-8.3](tasks/task-8.3.md) | Verify recovery from a local backup | task-8.2 | [#65](https://github.com/pchkauu/want-keep/issues/65) |
| [task-9.1](tasks/task-9.1.md) | Run end-to-end acceptance of the full MVP | task-4.1, task-4.2, task-4.3, task-4.4, task-4.5, task-4.6, task-7.3, task-7.6, task-7.7, task-7.8, task-8.3, task-7.12 | [#66](https://github.com/pchkauu/want-keep/issues/66) |
| [task-9.2](tasks/task-9.2.md) | Run final independent review and hand off the MVP | task-9.1 | [#67](https://github.com/pchkauu/want-keep/issues/67) |
