"""Checks for the decision-complete task-0.10 readiness artifacts."""

from decimal import Decimal, ROUND_HALF_EVEN, getcontext
import json
from pathlib import Path
import re
import unittest


class ReadinessContractTest(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    cls.root = Path(__file__).resolve().parents[1]
    cls.catalog = json.loads((cls.root / "catalog.json").read_text())

  def document(self, name):
    return (self.root / name).read_text()

  def test_ready_artifacts_and_decisions_are_bilingual(self):
    self.assertEqual("ready_for_development", self.catalog["readiness"]["status"])
    for name in ("plan.md", "plan.en.md", "evidence/task-0.10-readiness.md", "evidence/task-0.10-readiness.en.md"):
      self.assertTrue((self.root / name).is_file(), name)
    for decision in ("D-37", "D-38", "D-39", "D-40", "D-41", "D-42", "D-43"):
      self.assertIn(decision, self.document("proposal.md"))
      self.assertIn(decision, self.document("proposal.en.md"))

  def test_ready_plan_mentions_every_task_in_both_languages(self):
    expected = {task["id"] for task in self.catalog["tasks"]}
    for name in ("plan.md", "plan.en.md"):
      actual = set(re.findall(r"task-\d+\.\d+", self.document(name)))
      self.assertEqual(set(), expected - actual, name)

  def test_public_states_and_command_retention_are_fixed(self):
    states = (
      "source_partial",
      "source_ambiguous",
      "valuation_unavailable",
      "quote_unavailable",
      "command_expired",
      "provider_not_admitted",
    )
    for name in ("contracts.md", "contracts.en.md"):
      contract = self.document(name)
      for state in states:
        self.assertIn(state, contract)
      for retention in ("90", "400", "/commands/recent"):
        self.assertIn(retention, contract)
      for contract_marker in (
        "commandId",
        "deploymentGate.status",
        "admissionRevision",
        "adapterBuildDigest",
        "backend/internal/connections/admission/",
        "camtCrossReportFingerprint",
        "rLow",
        "max(1",
        "ROUND_HALF_EVEN",
      ):
        self.assertIn(contract_marker, contract)

  def test_provider_admission_acceptance_covers_owning_tasks(self):
    expected_tasks = {
      "task-0.10",
      "task-1.2",
      "task-1.3",
      "task-3.2",
      "task-3.3",
      "task-4.1",
      "task-4.2",
      "task-4.3",
      "task-4.4",
      "task-4.5",
      "task-4.6",
      "task-4.7",
      "task-8.1",
      "task-9.1",
      "task-9.2",
    }
    actual_tasks = {
      task["id"]
      for task in self.catalog["tasks"]
      if "AC-106" in task["acceptance"]
    }
    self.assertEqual(expected_tasks, actual_tasks)

  def test_end_to_end_issue_body_stays_publishable(self):
    task = next(task for task in self.catalog["tasks"] if task["id"] == "task-9.1")
    self.assertEqual("ids", task["requirements_display"])
    body = self.document("tasks/task-9.1.md")
    self.assertLess(len(body), 60_000)
    for requirement in {
      requirement
      for acceptance_id in task["acceptance"]
      for requirement in self.acceptance(acceptance_id)["requirements"]
    }:
      self.assertIn(f"`{requirement}`", body)

  def test_xirr_reference_vector_and_ambiguity_examples(self):
    getcontext().prec = 40
    rate = Decimal("0.1")
    npv = Decimal("-1000") + Decimal("1100") / (Decimal(1) + rate)
    self.assertEqual(Decimal(0), npv)
    getcontext().prec = 80
    irregular_rate = (
      (Decimal("1.05").ln() * (Decimal(365) / Decimal(182))).exp()
      - Decimal(1)
    )
    self.assertEqual(
      Decimal("0.102795595422"),
      irregular_rate.quantize(Decimal("0.000000000001"), rounding=ROUND_HALF_EVEN),
    )
    self.assertEqual(1, self.sign_transitions((Decimal("-1000"), Decimal("1100"))))
    self.assertEqual(2, self.sign_transitions((Decimal("-100"), Decimal("230"), Decimal("-132"))))
    self.assertEqual(0, self.sign_transitions((Decimal("1"), Decimal("2"))))

  @staticmethod
  def sign_transitions(flows):
    signs = [flow > 0 for flow in flows if flow]
    return sum(left != right for left, right in zip(signs, signs[1:]))

  def acceptance(self, identifier):
    for requirement in self.catalog["requirements"]:
      if requirement["acceptance"]["id"] == identifier:
        return {"requirements": [requirement["id"]]}
    return next(
      acceptance
      for acceptance in self.catalog["extra_acceptance"]
      if acceptance["id"] == identifier
    )


if __name__ == "__main__":
  unittest.main()
