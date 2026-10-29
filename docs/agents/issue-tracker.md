# Issue tracker

Use GitHub Issues in `andrey-krasheninnikov/want-keep` through `gh`.
Always pass the repository explicitly.

Read the issue, comments, labels and dependencies before changing it.
Create, edit, comment or close only when the current request authorizes it.
When a skill says "publish to the issue tracker", create a GitHub issue.

Pull requests as a triage request surface: no.

Keep existing issue numbers, task IDs, discussions and requirement links.
Maintain RU/EN descriptions with current behavior, desired behavior,
key interfaces, acceptance criteria, verification and scope boundaries.

Use native GitHub blocking dependencies. If unavailable, retain explicit
"Blocked by" issue links. Closed prerequisites remain historical references.
A task can start only when its blocking issues are closed and its entry
conditions are satisfied.

For SDD tasks, edit catalog.json and render the task card before updating
the issue. Keep approved issue instructions and the catalog consistent.
Re-read each external change; reconcile uncertain results before retrying.
