SHELL := /bin/sh

GO ?= go
NPM ?= npm
PYTHON ?= python3

PKG ?= ./...
FILTER ?= all
AREA ?=
PROVIDER ?=
SCENARIO ?=
SUITE ?=
MODE ?=
E2E_WEB_DIR ?= web

.DEFAULT_GOAL := help

.PHONY: help bootstrap check format-check format-check-go lint typecheck test test-tooling build docs-check check-contracts generate-contracts generate-ai-runtime-contract check-ai-runtime-contract test-go test-web test-collector test-integration test-storage-race test-identity-race test-household-race test-accounts-race test-ledger-race test-audit-race test-matching-race test-categories-race test-family-allocation-race test-family-reimbursements-race test-reconciliation-race test-jobs-race test-ingestion-race test-ai-budget-race test-contract e2e eval-ai check-deploy backup-check restore-check

help:
	@echo "Want Keep repository commands"
	@echo "  make bootstrap"
	@echo "  make check"
	@echo "  make test-go PKG=./internal/<feature>/..."
	@echo "  make test-web FILTER=<feature|all>"
	@echo "  make test-collector FILTER=<suite|all>"

bootstrap:
	cd backend && $(GO) mod download
	$(NPM) ci --prefix api
	$(NPM) ci --prefix web
	$(NPM) ci --prefix collector

check: format-check lint typecheck test build docs-check check-contracts check-ai-runtime-contract

format-check: format-check-go
	$(NPM) --prefix web run format:check
	$(NPM) --prefix collector run format:check

format-check-go:
	@if ! go_root="$$(cd backend && $(GO) env GOROOT)"; then \
		echo "Unable to resolve the module-selected Go toolchain." >&2; exit 1; \
	fi; \
	gofmt="$$go_root/bin/gofmt"; \
	if [ ! -x "$$gofmt" ]; then echo "Pinned gofmt is unavailable at $$gofmt." >&2; exit 1; fi; \
	if ! unformatted="$$(find backend -type f -name '*.go' -print0 | xargs -0 "$$gofmt" -l)"; then \
		echo "Pinned gofmt failed." >&2; exit 1; \
	fi; \
	if [ -n "$$unformatted" ]; then echo "Go files need formatting:"; echo "$$unformatted"; exit 1; fi

lint:
	cd backend && $(GO) vet ./...
	$(NPM) --prefix web run lint
	$(NPM) --prefix collector run lint

typecheck:
	$(NPM) --prefix web run typecheck
	$(NPM) --prefix collector run typecheck

test: test-tooling test-go
	$(NPM) --prefix web run test
	$(NPM) --prefix collector run test

test-tooling:
	sh scripts/check-make-contracts.sh
	$(PYTHON) -m unittest discover -s scripts -p 'test_*.py'

build:
	$(NPM) --prefix web run build
	$(NPM) --prefix collector run build

docs-check:
	$(PYTHON) spec/001-want-keep-mvp/tools/spec_tool.py check
	$(PYTHON) -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_*.py'

check-contracts:
	sh scripts/check-openapi-state.sh
	sh scripts/generate-ingestion-contracts.sh --check

generate-contracts:
	@if [ ! -f api/openapi.yaml ] || [ ! -f api/oapi-codegen.yaml ] || [ ! -f scripts/generate-openapi.sh ]; then \
		echo "OpenAPI generation is unavailable until task-1.2 supplies source, config and generator." >&2; exit 2; fi
	sh scripts/generate-openapi.sh
	sh scripts/generate-ingestion-contracts.sh

generate-ai-runtime-contract:
	$(PYTHON) scripts/generate-openai-runtime.py --output backend/internal/gateways/openai/runtime_contract.json

check-ai-runtime-contract:
	sh scripts/check-openai-runtime.sh

test-go:
	cd backend && $(GO) test $(PKG)

test-web:
	@if [ "$(FILTER)" = "all" ]; then \
		$(NPM) --prefix web run test; \
	else \
		$(NPM) --prefix web run test -- "$(FILTER)"; \
	fi

test-collector:
	@if [ "$(FILTER)" = "all" ]; then \
		$(NPM) --prefix collector run test; \
	else \
		$(NPM) --prefix collector run test -- "$(FILTER)"; \
	fi

test-integration:
ifeq ($(AREA),privacy)
	sh scripts/test-privacy.sh
else ifeq ($(AREA),all)
	@for directory in backend/test/integration/*; do if [ -d "$$directory" ]; then $(MAKE) test-integration AREA="$${directory##*/}" || exit $$?; fi; done
else
	@if [ -z "$(AREA)" ]; then echo "AREA=<suite|all> is required." >&2; exit 2; fi
	@if [ "$(AREA)" = "all" ]; then suite_path="backend/test/integration"; else suite_path="backend/test/integration/$(AREA)"; fi; \
		if [ ! -d "$$suite_path" ]; then echo "Integration suite '$(AREA)' is not implemented." >&2; exit 2; fi; \
		if ! find "$$suite_path" -type f -name '*_test.go' -print -quit | grep -q .; then echo "Integration suite '$(AREA)' has no tests." >&2; exit 2; fi; \
		cd backend && $(GO) test -count=1 -tags=integration "./$${suite_path#backend/}/..."

endif

test-household-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/household/...

test-identity-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/identity/...

test-storage-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/storage/...

test-contract:
	@if [ -z "$(PROVIDER)" ]; then echo "PROVIDER=<name> is required." >&2; exit 2; fi
	@if [ ! -d "collector/contracts/$(PROVIDER)" ]; then echo "Provider contract suite '$(PROVIDER)' is not implemented." >&2; exit 2; fi
	@if ! find "collector/contracts/$(PROVIDER)" -type f -name '*.test.ts' -print -quit | grep -q .; then echo "Provider contract suite '$(PROVIDER)' has no tests." >&2; exit 2; fi
	$(NPM) --prefix collector run test -- "contracts/$(PROVIDER)"

e2e:
ifeq ($(SCENARIO),all)
	@if ! find "$(E2E_WEB_DIR)/e2e" -type f -name '*.spec.ts' -print -quit | grep -q .; then echo "E2E suite has no scenarios." >&2; exit 2; fi
	@for suite in "$(E2E_WEB_DIR)"/e2e/*.spec.ts; do \
		name="$${suite##*/}"; \
		$(MAKE) e2e SCENARIO="$${name%.spec.ts}" || exit $$?; \
	done
else ifeq ($(SCENARIO),access)
	sh scripts/test-access.sh
else
	@if [ -z "$(SCENARIO)" ]; then echo "SCENARIO=<name|all> is required." >&2; exit 2; fi
	@if [ "$(SCENARIO)" = "all" ]; then suite_path="$(E2E_WEB_DIR)/e2e"; else suite_path="$(E2E_WEB_DIR)/e2e/$(SCENARIO).spec.ts"; fi; \
		if [ ! -e "$$suite_path" ]; then echo "E2E scenario '$(SCENARIO)' is not implemented." >&2; exit 2; fi; \
		if [ "$(SCENARIO)" = "all" ] && ! find "$$suite_path" -type f -name '*.spec.ts' -print -quit | grep -q .; then echo "E2E suite has no scenarios." >&2; exit 2; fi
	cd "$(E2E_WEB_DIR)" && $(NPM) exec playwright test -- $(if $(filter all,$(SCENARIO)),,"e2e/$(SCENARIO).spec.ts")

endif

eval-ai:
	@if [ -z "$(SUITE)" ]; then echo "SUITE=<name|all> is required." >&2; exit 2; fi
	@if [ "$(SUITE)" = "all" ]; then suite_path="backend/test/ai"; else suite_path="backend/test/ai/$(SUITE)"; fi; \
		if [ ! -d "$$suite_path" ]; then echo "AI evaluation suite '$(SUITE)' is not implemented." >&2; exit 2; fi; \
		if ! find "$$suite_path" -type f -name '*_test.go' -print -quit | grep -q .; then echo "AI evaluation suite '$(SUITE)' has no tests." >&2; exit 2; fi; \
		cd backend && $(GO) test "./$${suite_path#backend/}/..."

check-deploy:
	@if [ ! -f deploy/compose.yaml ]; then echo "Deployment configuration is not implemented." >&2; exit 2; fi
	docker compose -f deploy/compose.yaml config --quiet

backup-check:
	@if [ "$(MODE)" != "synthetic" ]; then echo "MODE=synthetic is required." >&2; exit 2; fi
	@if [ ! -f ops/backup/check.sh ]; then echo "Synthetic backup check is not implemented." >&2; exit 2; fi
	sh ops/backup/check.sh

restore-check:
	@if [ "$(MODE)" != "synthetic" ]; then echo "MODE=synthetic is required." >&2; exit 2; fi
	@if [ ! -f ops/restore/check.sh ]; then echo "Synthetic restore check is not implemented." >&2; exit 2; fi
	sh ops/restore/check.sh

test-accounts-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/accounts/...

test-ledger-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/ledger/...

test-audit-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/audit/...

test-matching-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/matching/...

test-categories-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/categories/...

test-family-allocation-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/family-allocation/...

test-family-reimbursements-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/family-reimbursements/...

test-reconciliation-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/reconciliation/...

test-jobs-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/jobs/...

test-ingestion-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/ingestion/...

test-ai-budget-race:
	cd backend && $(GO) test -count=1 -race -tags=integration ./test/integration/ai-budget/...
