# Dev tools, version-pinned via `go run` instead of a globally
# installed binary or `go get -tool` in go.mod: arnon is a library, so
# it doesn't make sense to pull linter/mutation-tester transitive
# dependencies into the go.mod/go.sum of the module that consumes
# arnon too.
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
GO_ARCH_LINT  := github.com/fe3dback/go-arch-lint@v1.16.0
GREMLINS      := github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
# Unpinned on purpose: govulncheck is only useful when it knows about the
# vulnerabilities disclosed since the last release, and it reads the database
# over the network anyway. Pinning it would freeze the analyzer, not the data.
GOVULNCHECK   := golang.org/x/vuln/cmd/govulncheck@latest

# The one non-Go dev tool here: there's no Go implementation with
# matching rule fidelity to markdownlint, the reference implementation
# the project's own recommended VS Code extension
# (.vscode/extensions.json) already uses. Pinned to the same
# markdownlint-cli2 version .github/workflows/ci.yml's
# markdownlint-cli2-action resolves internally, run via npx instead of
# a global install. Requires Node.js >= 20.
MARKDOWNLINT := markdownlint-cli2@0.23.2

# Example package run by `make run`/`make build`, under examples/cmd
# (see examples/internal for the code shared between them).
EXAMPLE ?= basic

# The repository is three modules (see go.work and
# docs/architecture/adr-0001-module-layout.md). Neither `go test ./...` nor
# golangci-lint crosses a module boundary - not even inside a workspace - so
# every per-module target below has to iterate instead of relying on "./...".
#
# arch-lint is the exception: go-arch-lint walks the filesystem rather than the
# module graph, so a single run from the root still covers all three modules
# and .go-arch-lint.yml still describes the whole package graph.
MODULES     := . observability/otel examples
# Modules that make up the library itself. examples/ is demo code, tested
# end-to-end with hurl against a real running server (see
# examples/cmd/*/requests.hurl), not with go test.
LIB_MODULES := . observability/otel

.PHONY: help build run fmt lint lint-fix lint-md verify-docs doc-sync arch-lint vuln test test-race coverage \
	test-mutation test-mutation-dry-run check clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

build: ## Build the example in examples/cmd/basic (or: make build EXAMPLE=name)
	@go build -o bin/$(EXAMPLE) ./examples/cmd/$(EXAMPLE)

run: ## Run the example in examples/cmd/basic (or: make run EXAMPLE=name)
	@go run ./examples/cmd/$(EXAMPLE)

fmt: ## Format the code in every module (gofmt/gofumpt/goimports/gci/golines, via golangci-lint)
	@for module in $(MODULES); do \
		(cd $$module && go run $(GOLANGCI_LINT) fmt) || exit 1; \
	done

lint: ## Run the linter on every module (golangci-lint v2)
	@for module in $(MODULES); do \
		echo "==> lint $$module"; \
		(cd $$module && go run $(GOLANGCI_LINT) run) || exit 1; \
	done

lint-fix: ## Run the linter on every module and apply available auto-fixes
	@for module in $(MODULES); do \
		(cd $$module && go run $(GOLANGCI_LINT) run --fix) || exit 1; \
	done

lint-md: ## Lint markdown files (markdownlint-cli2, requires Node.js >= 20)
	@npx --yes $(MARKDOWNLINT) "**/*.md" "#NOTES.md"

verify-docs: ## Compile every self-contained Go program embedded in the Markdown
# Complements `make check`: that proves the code is correct, not that the
# documentation still describes it. Illustrative fragments (a bare type
# declaration, a struct literal with a literal `...`) can't compile standalone
# and are reported as unverified rather than silently assumed correct. The
# runnable Example functions in each package's example_test.go cover the rest,
# and run under `make test`.
	@bash scripts/verify-docs.sh

doc-sync: ## List documentation a change to the exported API has left behind
# Deliberately never fails: a diff can touch an exported symbol without any of
# the listed files needing an edit, and a check that cries wolf gets ignored.
# It covers the residue verify-docs cannot - fragments and prose.
	@bash scripts/doc-sync-check.sh

arch-lint: ## Check the package dependency graph (.go-arch-lint.yml)
# One run from the root covers all three modules: go-arch-lint resolves
# components from the filesystem, not the module graph, so the boundaries it
# enforces are unaffected by the split (verified by planting a deliberate
# observability/otel -> problem import and watching it fail).
	@go run $(GO_ARCH_LINT) check

vuln: ## Scan every module for known vulnerabilities (govulncheck)
# Reports only vulnerabilities actually reachable from this code, not every
# advisory touching a module in the graph - which is why a finding here is
# worth acting on rather than triaging away.
	@for module in $(MODULES); do \
		echo "==> vuln $$module"; \
		(cd $$module && go run $(GOVULNCHECK) ./...) || exit 1; \
	done

test: ## Run the tests
	@for module in $(LIB_MODULES); do \
		echo "==> test $$module"; \
		(cd $$module && go test ./...) || exit 1; \
	done

test-race: ## Run the tests with the race detector
	@for module in $(LIB_MODULES); do \
		echo "==> test -race $$module"; \
		(cd $$module && go test -race ./...) || exit 1; \
	done

coverage: ## Generate coverage.out and coverage.html with the coverage report
# One profile per module, concatenated: `go test -coverprofile` writes a "mode:"
# header line that must appear exactly once, so only the first module's header
# is kept.
	@echo "mode: set" > coverage.out
	@for module in $(LIB_MODULES); do \
		(cd $$module && go test -coverprofile=coverage.part ./... >/dev/null) || exit 1; \
		tail -n +2 $$module/coverage.part >> coverage.out; \
		rm -f $$module/coverage.part; \
	done
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Report at coverage.html"

test-mutation: ## Run mutation tests (gremlins) on the library modules, writes mutation.json
# gremlins doesn't handle Go's "./..." pattern well (silently reports
# nothing); passing "." makes it recurse through the whole module on
# its own. It also takes one module at a time, and -o overwrites rather than
# appends, so each module gets its own report file.
	@for module in $(LIB_MODULES); do \
		echo "==> mutation $$module"; \
		(cd $$module && go run $(GREMLINS) unleash -o mutation.json .) || exit 1; \
	done

test-mutation-dry-run: ## List the mutants without running the tests (much faster)
	@for module in $(LIB_MODULES); do \
		(cd $$module && go run $(GREMLINS) unleash --dry-run .) || exit 1; \
	done

check: lint arch-lint verify-docs test-race ## Run lint + arch-lint + verify-docs + tests with the race detector (the minimum CI should run)

clean: ## Remove build/test artifacts from every module
	@rm -rf bin coverage.out coverage.html
	@for module in $(MODULES); do \
		rm -f $$module/coverage.out $$module/coverage.part $$module/mutation.json; \
	done
