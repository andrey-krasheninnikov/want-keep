#!/usr/bin/env python3
"""Render and validate the bilingual Want Keep specification catalog."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import sys
from urllib.parse import unquote, urlsplit


class SpecCatalog:
  def __init__(self, root):
    self.root = root
    self.data = json.loads((root / "catalog.json").read_text())
    self.requirements = {item["id"]: item for item in self.data["requirements"]}
    self.criteria = {}
    for requirement in self.data["requirements"]:
      criterion = dict(requirement["acceptance"])
      criterion["requirements"] = [requirement["id"]]
      criterion["title"] = requirement["title"]
      self.criteria[criterion["id"]] = criterion
    self.criteria.update({item["id"]: item for item in self.data["extra_acceptance"]})
    self.tasks = {item["id"]: item for item in self.data["tasks"]}
    self.screens = {item["id"]: item for item in self.data["screens"]}
    self.forms = {item["id"]: item for item in self.data["forms"]}
    self.ui_states = {item["id"]: item for item in self.data["ui_states"]}
    self.readiness = self.data.get("readiness", {})
    self._ancestry = {}

  def is_ready_for_development(self):
    return self.readiness.get("status") == "ready_for_development"

  def task_requirements(self, task):
    return sorted({req for ac in task["acceptance"] for req in self.criteria[ac]["requirements"]})

  def task_status(self, task, lang):
    if "status" in task:
      return task["status"][lang]
    if task["kind"] == "research":
      return {"ru": "Исследование — не начато; live-доступ и платные прогоны требуют безопасно предоставленного доступа владельца.", "en": "Research — not started; live access and paid runs require securely supplied owner access."}[lang]
    if self.is_ready_for_development():
      return {"ru": "Не начато; задача ожидает собственные зависимости и entry gates.", "en": "Not started; the task awaits its own dependencies and entry gates."}[lang]
    return {"ru": "Заблокировано зависимостями и проверкой SDD Ready; реализация не начата.", "en": "Blocked by dependencies and the SDD Ready gate; implementation has not started."}[lang]

  def task_verification_note(self, task, lang):
    if "verification_note" in task:
      return task["verification_note"][lang]
    return {"ru": "Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.", "en": "The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers."}[lang]

  def criterion_text(self, criterion, lang):
    labels = {"ru": ("Дано", "Когда", "Тогда", "Уровень"), "en": ("Given", "When", "Then", "Level")}[lang]
    return "\n".join([f"- **{labels[0]}:** {criterion['given'][lang]}", f"- **{labels[1]}:** {criterion['when'][lang]}", f"- **{labels[2]}:** {criterion['then'][lang]}", f"- **{labels[3]}:** `{criterion['level']}`."])

  def task_body(self, task):
    lines = [f"<!-- want-keep-task: {task['id']} -->", f"# {task['id']} — {task['title']['ru']} / {task['title']['en']}", ""]
    reqs = self.task_requirements(task)
    for lang in ("ru", "en"):
      ru = lang == "ru"
      lines += ["## " + ("RU" if ru else "EN"), "", task["goal"][lang], "", "**" + ("Состояние" if ru else "Status") + ":** " + self.task_status(task, lang), "", "**" + ("Зависимости" if ru else "Dependencies") + ":** " + (", ".join(f"`{dep}`" for dep in task["depends_on"]) or ("нет" if ru else "none")) + ".", "", "**" + ("Тип" if ru else "Kind") + f":** `{task['kind']}`.", "", "### " + ("Изменение и контракты" if ru else "Change and contracts"), "", task["behavior"][lang], "", "### " + ("Границы изменений" if ru else "Change boundaries"), ""]
      lines += [f"- `{target}`" for target in task["targets"]]
      screens = [screen for screen in self.screens.values() if task["id"] in screen["tasks"]]
      if screens:
        lines += ["", "### " + ("Экранный контракт" if ru else "Screen contract"), ""]
        for screen in screens:
          lines += self.screen_text(screen, lang, include_links=False)
        for form_id in sorted({item for screen in screens for item in screen["forms"]}):
          lines += self.form_text(self.forms[form_id], lang)
        for state_id in sorted({item for screen in screens for item in screen["states"]}):
          state = self.ui_states[state_id]
          lines += [f"- **{state_id} — {state['title'][lang]}:** {state['behavior'][lang]}"]
        lines.append("")
      lines += ["", ("Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу." if ru else "Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work."), "", "### " + ("Связанные требования" if ru else "Linked requirements"), ""]
      if task.get("requirements_display") == "ids":
        lines += [("Полные формулировки проверяемого поведения приведены в AC ниже. REQ: " if ru else "The AC scenarios below contain the complete verifiable behavior. REQ: ") + ", ".join(f"`{req}`" for req in reqs) + "."]
      else:
        lines += [f"- **{req}:** {self.requirements[req]['title'][lang]}" for req in reqs]
      lines += ["", "### " + ("Критерии приёмки" if ru else "Acceptance criteria"), "", ("Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже." if ru else "A link establishes coverage but does not prove the whole criterion; verification below records the exact result."), ""]
      for ac in task["acceptance"]:
        criterion = self.criteria[ac]
        lines += [f"#### {ac}", "", self.criterion_text(criterion, lang), ""]
      lines += ["### " + ("Проверка результата" if ru else "Verification"), "", "```sh", task["verification"]["command"], "```", "", task["verification"]["expected"][lang], "", self.task_verification_note(task, lang), "", "### " + ("Передача следующему агенту" if ru else "Handoff to the next agent"), "", ("Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата." if ru else "Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence."), "", "**Commit boundary:** " + ("commit/push/deploy требуют действующей авторизации пользователя." if ru else "commit/push/deploy require current user authorization."), ""]
    return "\n".join(lines).rstrip() + "\n"

  def generated(self):
    output = {}
    for lang in ("ru", "en"):
      ru = lang == "ru"
      suffix = "" if ru else ".en"
      notice = ("Собрано из [catalog.json](catalog.json). Редактировать каталог, затем выполнить `python3 spec/001-want-keep-mvp/tools/spec_tool.py render`." if ru else "Rendered from [catalog.json](catalog.json). Edit the catalog, then run `python3 spec/001-want-keep-mvp/tools/spec_tool.py render`.")
      lines = ["# " + ("Требования Want Keep MVP" if ru else "Want Keep MVP requirements"), "", notice, "", ("Контекст, цели, исключения и решения — в [proposal.md](proposal.md); контракты/ошибки — в [contracts.md](contracts.md); блокеры — в [verification.md](verification.md). Все требования обязательны для полного MVP." if ru else "Context, goals, exclusions and decisions are in [proposal.en.md](proposal.en.md); contracts/errors in [contracts.en.md](contracts.en.md); blockers in [verification.en.md](verification.en.md). Every requirement is mandatory for the complete MVP."), ""]
      for req in self.requirements.values():
        lines += [f"## {req['id']}", "", req["title"][lang], "", f"{'Основание' if ru else 'Source'}: `{req['decision']}`. {'Приёмка' if ru else 'Acceptance'}: [{req['acceptance']['id']}](acceptance_criteria{suffix}.md#{req['acceptance']['id'].lower()}).", ""]
      output[f"requirements{suffix}.md"] = "\n".join(lines).rstrip() + "\n"
      lines = ["# " + ("Критерии приёмки Want Keep MVP" if ru else "Want Keep MVP acceptance criteria"), "", notice, "", ("Сценарии являются будущими критериями продукта, а не отчётом о пройденных тестах. `contract+manual` требует и синтетического контракта, и отдельного подтверждения реального чтения. Подробное покрытие — в [traceability.md](traceability.md)." if ru else "Scenarios are future product criteria, not a report of passing tests. `contract+manual` requires both a synthetic contract and separate live-read evidence. Detailed coverage is in [traceability.en.md](traceability.en.md)."), "", "| REQ | AC |", "| --- | --- |"]
      for req in self.requirements:
        linked = [f"[{ac['id']}](#{ac['id'].lower()})" for ac in self.criteria.values() if req in ac["requirements"]]
        lines.append(f"| [{req}](requirements{suffix}.md#{req.lower()}) | {', '.join(linked)} |")
      for ac in self.criteria.values():
        lines += ["", f"## {ac['id']}", "", ac["title"][lang], "", "REQ: " + ", ".join(f"`{req}`" for req in ac["requirements"]) + ".", "", self.criterion_text(ac, lang)]
      output[f"acceptance_criteria{suffix}.md"] = "\n".join(lines).rstrip() + "\n"
      backlog_intro = {
        "ready": {
          "ru": "Спецификация прошла task-0.10 и готова к разработке. Решение-полный порядок, параллелизм и entry/exit gates опубликованы в [plan.md](plan.md). Каждая задача по-прежнему ждёт собственные зависимости и runtime gates; Ready SDD не означает реализованный или принятый MVP. Карточки самодостаточны и содержат RU/EN.",
          "en": "The specification passed task-0.10 and is ready for development. The decision-complete order, parallelism and entry/exit gates are published in [plan.en.md](plan.en.md). Each task still awaits its own dependencies and runtime gates; SDD Ready does not mean an implemented or accepted MVP. Task cards are self-contained in RU/EN.",
        },
        "not_ready": {
          "ru": "Полный backlog не является Ready-планом реализации. Сначала task-0.1–task-0.9 собирают доказательства, затем task-0.10 закрывает блокеры и проверяет SDD Ready. Все последующие задачи ждут этого барьера и собственных зависимостей. `plan.md` намеренно отсутствует до Ready. Карточки самодостаточны и содержат RU/EN.",
          "en": "The full backlog is not a Ready implementation plan. First task-0.1–task-0.9 collect evidence; task-0.10 then resolves blockers and reviews SDD readiness. All later tasks await that gate and their own dependencies. `plan.md` intentionally does not exist before Ready. Task cards are self-contained in RU/EN.",
        },
      }
      readiness_key = "ready" if self.is_ready_for_development() else "not_ready"
      lines = ["# " + ("Backlog Want Keep MVP" if ru else "Want Keep MVP backlog"), "", notice, "", backlog_intro[readiness_key][lang], "", "| Task | " + ("Результат" if ru else "Outcome") + " | " + ("Зависимости" if ru else "Dependencies") + " | GitHub |", "| --- | --- | --- | --- |"]
      for task in self.tasks.values():
        issue = f"[#{task['github_url'].rsplit('/', 1)[-1]}]({task['github_url']})" if task["github_url"] else ("не опубликована" if ru else "not published")
        lines.append(f"| [{task['id']}](tasks/{task['id']}.md) | {task['title'][lang]} | {', '.join(task['depends_on']) or '—'} | {issue} |")
      output[f"backlog{suffix}.md"] = "\n".join(lines).rstrip() + "\n"
      lines = ["# " + ("Трассировка требований" if ru else "Requirements traceability"), "", notice, "", "| REQ | AC | Task |", "| --- | --- | --- |"]
      for req in self.requirements:
        acs = [ac["id"] for ac in self.criteria.values() if req in ac["requirements"]]
        covered = [task["id"] for task in self.tasks.values() if req in self.task_requirements(task)]
        lines.append(f"| [{req}](requirements{suffix}.md#{req.lower()}) | {', '.join(f'[{ac}](acceptance_criteria{suffix}.md#{ac.lower()})' for ac in acs)} | {', '.join(f'[{task}](tasks/{task}.md)' for task in covered)} |")
      output[f"traceability{suffix}.md"] = "\n".join(lines).rstrip() + "\n"
      output[f"screens{suffix}.md"] = self.screens_document(lang, notice)
    output.update({f"tasks/{task['id']}.md": self.task_body(task) for task in self.tasks.values()})
    return output

  def screen_text(self, screen, lang, include_links=True):
    labels = {"ru": ("Вопрос", "Главный ответ", "Структура сверху вниз", "Следующее действие", "Объяснение и детализация", "Права"), "en": ("Question", "Primary answer", "Top-down structure", "Next action", "Explanation and details", "Permissions")}[lang]
    lines = [f"### {screen['id']} — {screen['title'][lang]}", "", f"`{screen['route']}`", ""]
    for key, label in zip(("question", "answer", "structure", "action", "detail", "rights"), labels):
      lines += [f"**{label}:** {screen[key][lang]}", ""]
    lines += ["Forms: " + (", ".join(screen["forms"]) or "—") + ".", "", "States: " + ", ".join(screen["states"]) + ".", ""]
    if include_links:
      lines += ["REQ: " + ", ".join(screen["requirements"]) + ". AC: " + ", ".join(screen["acceptance"]) + ".", "", "Task: " + ", ".join(f"[{task}](tasks/{task}.md)" for task in screen["tasks"]) + ".", ""]
    return lines

  def form_text(self, form, lang):
    labels = {"ru": ("Поля", "Проверки и права", "Результат"), "en": ("Fields", "Validation and permissions", "Outcome")}[lang]
    lines = [f"#### {form['id']} — {form['title'][lang]}", ""]
    for key, label in zip(("fields", "rules", "result"), labels):
      lines += [f"**{label}:** {form[key][lang]}", ""]
    return lines

  def screens_document(self, lang, notice):
    ru = lang == "ru"
    suffix = "" if ru else ".en"
    lines = ["# " + ("Экраны, формы и состояния Want Keep" if ru else "Want Keep screens, forms and states"), "", notice, "", f"[{'Дизайн' if ru else 'Design'}](design{suffix}.md) · [{'Навигация' if ru else 'Navigation'}](navigation{suffix}.md)", "", ("Маршруты — контракты будущего UI. Все перечисленные состояния проверяются при соответствующем отказе/действии; не перечисленное состояние неприменимо к этому экрану. Общие состояния применяются к отдельному блоку, не скрывая работающие части. Формы наследуют saving/unknown/conflict/permission/session/error/offline/success по типу команды. Изменение окна и масштаба сохраняет desktop-навигацию. Точные API payload определяются contracts и Ready; UI не изобретает поля источников." if ru else "Routes are future UI contracts. Listed states are tested under the relevant failure/action; omitted states are inapplicable to that screen. Shared states apply per section, preserving working parts. Forms inherit saving/unknown/conflict/permission/session/error/offline/success according to command type. Resizing/zoom preserves desktop navigation. Exact API payloads follow contracts and Ready; UI never invents provider fields."), "", "## " + ("Каталог состояний" if ru else "State catalog"), ""]
    for state in self.ui_states.values():
      lines += [f"### {state['id']} — {state['title'][lang]}", "", state["behavior"][lang], ""]
    lines += ["## " + ("Каталог экранов" if ru else "Screen catalog"), ""]
    for screen in self.screens.values():
      lines += self.screen_text(screen, lang)
    lines += ["## " + ("Каталог форм" if ru else "Form catalog"), ""]
    for form in self.forms.values():
      lines += self.form_text(form, lang)
    return "\n".join(lines).rstrip() + "\n"

  def validate_catalog(self):
    errors = []
    if set(self.readiness) != {"status", "date", "gate_task"}:
      errors.append("Invalid readiness metadata")
    elif self.readiness["status"] not in ("not_ready", "ready_for_development"):
      errors.append("Invalid readiness status")
    elif not re.fullmatch(r"\d{4}-\d{2}-\d{2}", self.readiness["date"]):
      errors.append("Invalid readiness date")
    elif self.readiness["gate_task"] != "task-0.10":
      errors.append("Invalid readiness gate task")
    if len(self.requirements) != len(self.data["requirements"]):
      errors.append("Duplicate REQ ID")
    if len(self.criteria) != len(self.data["requirements"]) + len(self.data["extra_acceptance"]):
      errors.append("Duplicate AC ID")
    if len(self.tasks) != len(self.data["tasks"]):
      errors.append("Duplicate task ID")
    for key, collection, pattern in (("screens", self.screens, r"SCR-\d{3}"), ("forms", self.forms, r"FORM-\d{2}"), ("ui_states", self.ui_states, r"UISTATE-\d{2}")):
      if len(collection) != len(self.data[key]):
        errors.append(f"Duplicate {key} ID")
      if any(not re.fullmatch(pattern, identifier) for identifier in collection):
        errors.append(f"Invalid {key} ID")
    if set(self.screens) != {f"SCR-{number:03}" for number in range(1, 36)}:
      errors.append("Screen inventory must be SCR-001–SCR-035")
    for screen in self.screens.values():
      if not screen["route"].startswith("/"):
        errors.append(f"Invalid route: {screen['id']}")
      for key, collection in (("requirements", self.requirements), ("acceptance", self.criteria), ("tasks", self.tasks), ("forms", self.forms), ("states", self.ui_states)):
        if (key != "forms" and not screen[key]) or any(ref not in collection for ref in screen[key]):
          errors.append(f"Unresolved {key}: {screen['id']}")
    for collection, pattern in ((self.requirements, r"REQ-\d{3}"), (self.criteria, r"AC-\d{3}"), (self.tasks, r"task-\d+\.\d+")):
      for identifier in collection:
        if not re.fullmatch(pattern, identifier):
          errors.append(f"Invalid ID: {identifier}")
    for ac in self.criteria.values():
      if not ac["requirements"] or any(req not in self.requirements for req in ac["requirements"]):
        errors.append(f"Unresolved REQ in {ac['id']}")
    for task in self.tasks.values():
      if not task["acceptance"] or any(ac not in self.criteria for ac in task["acceptance"]):
        errors.append(f"Unresolved AC in {task['id']}")
      if any(dep not in self.tasks or dep == task["id"] for dep in task["depends_on"]):
        errors.append(f"Unresolved dependency in {task['id']}")
      if task["kind"] not in ("research", "specification", "implementation", "verification"):
        errors.append(f"Invalid task kind: {task['id']}")
      if task.get("requirements_display") not in (None, "ids"):
        errors.append(f"Invalid requirements display: {task['id']}")
      if "status" in task and (not isinstance(task["status"], dict) or set(task["status"]) != {"ru", "en"}):
        errors.append(f"Invalid task status translation: {task['id']}")
      for target in task["targets"]:
        if Path(target).is_absolute() or ".." in Path(target).parts or ".git" in Path(target).parts:
          errors.append(f"Unsafe target: {target}")
      if task["github_url"] and not re.fullmatch(r"https://github.com/(?:pchkauu|andrey-krasheninnikov)/want-keep/issues/[1-9]\d*", task["github_url"]):
        errors.append(f"Invalid GitHub URL: {task['id']}")
    if errors:
      return errors
    for screen in self.screens.values():
      covered = {ac for task in screen["tasks"] for ac in self.tasks[task]["acceptance"]}
      if set(screen["acceptance"]) - covered:
        errors.append(f"Screen acceptance lacks task coverage: {screen['id']}")
      linked = {req for ac in screen["acceptance"] for req in self.criteria[ac]["requirements"]}
      if set(screen["requirements"]) - linked:
        errors.append(f"Screen requirement lacks acceptance coverage: {screen['id']}")
    for req in self.requirements:
      if not any(req in ac["requirements"] for ac in self.criteria.values()):
        errors.append(f"Uncovered requirement: {req}")
      if not any(req in self.task_requirements(task) and task["kind"] in ("implementation", "verification") for task in self.tasks.values()):
        errors.append(f"No implementation coverage: {req}")
    for ac in self.criteria:
      if not any(ac in task["acceptance"] for task in self.tasks.values()):
        errors.append(f"No task covers {ac}")
    try:
      for task in self.tasks:
        self.ancestors(task)
    except ValueError as error:
      errors.append(str(error))
      return errors
    for task in self.tasks.values():
      if task["kind"] not in ("research", "specification") and "task-0.10" not in self.ancestors(task["id"]):
        errors.append(f"Task bypasses readiness gate: {task['id']}")
    self.validate_translations(self.data, "catalog", errors)
    return errors

  def ancestors(self, identifier, stack=()):
    if identifier in stack:
      raise ValueError("Dependency cycle: " + " -> ".join(stack + (identifier,)))
    if identifier in self._ancestry:
      return self._ancestry[identifier]
    result = set()
    for dep in self.tasks[identifier]["depends_on"]:
      result.add(dep)
      result.update(self.ancestors(dep, stack + (identifier,)))
    self._ancestry[identifier] = result
    return result

  def validate_translations(self, value, path, errors):
    if isinstance(value, dict):
      if "ru" in value or "en" in value:
        if set(value) != {"ru", "en"} or any(not isinstance(value.get(lang), str) or not value[lang].strip() for lang in ("ru", "en")):
          errors.append(f"Missing translation: {path}")
      for key, child in value.items():
        self.validate_translations(child, f"{path}.{key}", errors)
    elif isinstance(value, list):
      for index, child in enumerate(value):
        self.validate_translations(child, f"{path}[{index}]", errors)

  def check_files(self, generated):
    errors = []
    for relative, content in generated.items():
      path = self.root / relative
      if not path.exists() or path.read_text() != content:
        errors.append(f"Stale or missing generated file: {relative}")
      if relative.startswith("tasks/") and len(content) >= 60000:
        errors.append(f"Issue body too long: {relative}")
    for name in ("README", "proposal", "constraints", "contracts", "glossary", "flows", "integrations", "operations", "verification", "design", "navigation"):
      for suffix in ("", ".en"):
        if not (self.root / f"{name}{suffix}.md").exists():
          errors.append(f"Missing hand-authored document: {name}{suffix}.md")
    for suffix in ("", ".en"):
      plan = self.root / f"plan{suffix}.md"
      if self.is_ready_for_development() and not plan.exists():
        errors.append(f"Missing Ready plan: {plan.name}")
      if not self.is_ready_for_development() and plan.exists():
        errors.append(f"Plan exists before Ready: {plan.name}")
    proposal = (self.root / "proposal.md").read_text() if (self.root / "proposal.md").exists() else ""
    for requirement in self.requirements.values():
      if requirement["decision"] not in proposal:
        errors.append(f"Unresolved decision source: {requirement['id']}")
    for path in self.root.rglob("*.md"):
      text = path.read_text()
      if re.search(r"/Users/|/private/tmp/|gh[pousr]_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9]{24,}", text):
        errors.append(f"Private path or secret-shaped string: {path.relative_to(self.root)}")
      for match in re.finditer(r"\[[^\]]*\]\(([^\s)]+)\)", text):
        url = match.group(1)
        if urlsplit(url).scheme or url.startswith("#"):
          continue
        file_part, _, anchor = url.partition("#")
        target = (path.parent / unquote(file_part)).resolve()
        if not target.exists():
          errors.append(f"Broken link: {path.relative_to(self.root)} -> {url}")
        elif anchor and re.fullmatch(r"(?:req|ac)-\d{3}", anchor):
          if f"## {anchor.upper()}\n" not in target.read_text():
            errors.append(f"Broken anchor: {url}")
    urls = [task["github_url"] for task in self.tasks.values() if task["github_url"]]
    if len(urls) != len(set(urls)):
      errors.append("Duplicate GitHub issue mapping")
    for relative in self.root.glob("tasks/*.md"):
      if relative.relative_to(self.root).as_posix() not in generated:
        errors.append(f"Unindexed task file: {relative.name}")
    return errors

  def run(self, action):
    errors = self.validate_catalog()
    if errors:
      print("\n".join(errors), file=sys.stderr)
      return 1
    generated = self.generated()
    if action == "render":
      for relative, content in generated.items():
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)
      print(f"Rendered {len(generated)} files")
      return 0
    errors.extend(self.check_files(generated))
    if errors:
      print("\n".join(errors), file=sys.stderr)
      return 1
    digest = hashlib.sha256((self.root / "catalog.json").read_bytes()).hexdigest()
    published = sum(bool(task["github_url"]) for task in self.tasks.values())
    print(f"PASS: {len(self.requirements)} REQ, {len(self.criteria)} AC, {len(self.tasks)} tasks, {len(self.screens)} screens; coverage, DAG, RU/EN presence, generated consistency and links valid; {published} GitHub mappings")
    print(f"Catalog SHA256: {digest}")
    return 0


if __name__ == "__main__":
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument("action", choices=("render", "check"))
  args = parser.parse_args()
  sys.exit(SpecCatalog(Path(__file__).resolve().parents[1]).run(args.action))
