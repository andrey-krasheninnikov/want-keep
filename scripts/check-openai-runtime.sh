#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
expected="$root/backend/internal/gateways/openai/runtime_contract.json"
temporary=$(mktemp)
receipt_temporary=$(mktemp)
trap 'rm -f "$temporary" "$receipt_temporary"' EXIT HUP INT TERM

python3 "$root/scripts/generate-openai-runtime.py" --output "$temporary" --receipt-output "$receipt_temporary"
if ! cmp -s "$expected" "$temporary"; then
  echo "OpenAI runtime contract is stale. Run: make generate-ai-runtime-contract" >&2
  diff -u "$expected" "$temporary" || true
  exit 1
fi

if ! cmp -s "$root/backend/internal/gateways/openai/receipt_contract.json" "$receipt_temporary"; then
  echo "Receipt runtime contract is stale. Run: make generate-ai-runtime-contract" >&2
  exit 1
fi
