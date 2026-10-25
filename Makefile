SHELL := /bin/bash
TF_DIRS := $(wildcard infra/envs/staging/base infra/envs/staging/compute infra/envs/prod)

.DEFAULT_GOAL := help
.PHONY: help require-go fmt fmt-check vet lint test arch harness-test check tf-fmt tf-check e2e

help: ## List targets
	@grep -E '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

require-go:
	@test -f go.mod || { echo "no go.mod yet: run step 1 of docs/plans/active/fase-1-fundacao.md"; exit 1; }

fmt: require-go ## Format Go code
	gofmt -w $$(git ls-files '*.go' ':!:**/testdata/**')

fmt-check: require-go
	@out=$$(gofmt -l $$(git ls-files '*.go' ':!:**/testdata/**')); test -z "$$out" || { echo "gofmt needed:"; echo "$$out"; exit 1; }

vet: require-go ## go vet
	go vet ./...

lint: require-go ## golangci-lint
	golangci-lint run ./...

test: require-go ## Unit tests with race detector (pipeline gate)
	go test ./... -race -count=1

arch: ## Architecture rules from CLAUDE.md
	@scripts/archcheck.sh

harness-test: ## Tests for the harness itself (hooks, archcheck)
	@scripts/test-harness.sh

check: fmt-check vet lint arch harness-test test ## Everything that must be green before a PR

tf-fmt: ## Format Terraform
	terraform fmt -recursive infra

tf-check: ## terraform fmt -check and validate (no backend, no apply)
	@test -d infra || { echo "no infra/ yet"; exit 0; }
	terraform fmt -check -recursive infra
	@for d in $(TF_DIRS); do \
		echo "validate $$d"; \
		terraform -chdir=$$d init -backend=false -input=false >/dev/null && terraform -chdir=$$d validate || exit 1; \
	done

e2e: require-go ## E2E suite against staging (CI only; needs env vars)
	go test -tags e2e ./e2e/... -count=1 -v
