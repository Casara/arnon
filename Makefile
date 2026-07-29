# Dev tools, version-pinned via `go run` instead of a globally
# installed binary or `go get -tool` in go.mod: arnon is a library, so
# it doesn't make sense to pull linter/mutation-tester transitive
# dependencies into the go.mod/go.sum of the module that consumes
# arnon too.
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
GO_ARCH_LINT  := github.com/fe3dback/go-arch-lint@v1.16.0
GREMLINS      := github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0

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

.PHONY: help build run fmt lint lint-fix lint-md arch-lint test test-race coverage \
	test-mutation test-mutation-dry-run check clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

build: ## Build the example in examples/cmd/basic (or: make build EXAMPLE=name)
	@go build -o bin/$(EXAMPLE) ./examples/cmd/$(EXAMPLE)

run: ## Run the example in examples/cmd/basic (or: make run EXAMPLE=name)
	@go run ./examples/cmd/$(EXAMPLE)

fmt: ## Format the code (gofmt/gofumpt/goimports/gci/golines, via golangci-lint)
	@go run $(GOLANGCI_LINT) fmt

lint: ## Run the linter (golangci-lint v2)
	@go run $(GOLANGCI_LINT) run

lint-fix: ## Run the linter and apply available auto-fixes
	@go run $(GOLANGCI_LINT) run --fix

lint-md: ## Lint markdown files (markdownlint-cli2, requires Node.js >= 20)
	@npx --yes $(MARKDOWNLINT) "**/*.md" "#NOTES.md"

arch-lint: ## Check the package dependency graph (.go-arch-lint.yml)
	@go run $(GO_ARCH_LINT) check

# examples/ is demo code, tested end-to-end with hurl against a real
# running server (see examples/cmd/*/requests.hurl), not with go test
# - so it's excluded from test/test-race/coverage, the same criterion
# test-mutation already used (-E 'examples/.*' below).
LIB_PACKAGES = $$(go list ./... | grep -v /examples)

test: ## Run the tests
	@go test $(LIB_PACKAGES)

test-race: ## Run the tests with the race detector
	@go test -race $(LIB_PACKAGES)

coverage: ## Generate coverage.out and coverage.html with the coverage report
	@go test -coverprofile=coverage.out $(LIB_PACKAGES)
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Report at coverage.html"

test-mutation: ## Run mutation tests (gremlins), writes mutation.json
# gremlins doesn't handle Go's "./..." pattern well (silently reports
# nothing); passing "." makes it recurse through the whole module on
# its own. examples/ is excluded since it's demo code, not the library.
	@go run $(GREMLINS) unleash -E 'examples/.*' -o mutation.json .

test-mutation-dry-run: ## List the mutants without running the tests (much faster)
	@go run $(GREMLINS) unleash --dry-run -E 'examples/.*' .

check: lint arch-lint test-race ## Run lint + arch-lint + tests with the race detector (the minimum CI should run)

clean: ## Remove build/test artifacts
	@rm -rf bin coverage.out coverage.html mutation.json
