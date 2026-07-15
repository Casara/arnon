# Ferramentas de desenvolvimento, com versão fixa via `go run` em vez de
# binário instalado globalmente ou de `go get -tool` no go.mod: o arnon é
# uma biblioteca, então não faz sentido puxar as dependências transitivas
# de linter/mutation-tester para dentro do go.mod/go.sum do módulo que
# quem consome o arnon também resolve.
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
GO_ARCH_LINT  := github.com/fe3dback/go-arch-lint@v1.16.0
GREMLINS      := github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0

# Pacote do exemplo executado por `make run`/`make build`, dentro de
# examples/cmd (ver examples/internal para o código compartilhado
# entre eles).
EXAMPLE ?= basic

.PHONY: help build run fmt lint lint-fix arch-lint test test-race coverage \
	test-mutation test-mutation-dry-run check clean

help: ## Exibe esta ajuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

build: ## Compila o exemplo em examples/cmd/basic (outro: make build EXAMPLE=nome)
	@go build -o bin/$(EXAMPLE) ./examples/cmd/$(EXAMPLE)

run: ## Executa o exemplo em examples/cmd/basic (outro: make run EXAMPLE=nome)
	@go run ./examples/cmd/$(EXAMPLE)

fmt: ## Formata o código (gofmt/gofumpt/goimports/gci/golines, via golangci-lint)
	@go run $(GOLANGCI_LINT) fmt

lint: ## Roda o linter (golangci-lint v2)
	@go run $(GOLANGCI_LINT) run

lint-fix: ## Roda o linter e aplica as correções automáticas possíveis
	@go run $(GOLANGCI_LINT) run --fix

arch-lint: ## Verifica o grafo de dependências entre pacotes (.go-arch-lint.yml)
	@go run $(GO_ARCH_LINT) check

test: ## Executa os testes
	@go test ./...

test-race: ## Executa os testes com o detector de race conditions
	@go test -race ./...

coverage: ## Gera coverage.out e coverage.html com o relatório de cobertura
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Relatório em coverage.html"

test-mutation: ## Executa testes de mutação (gremlins) e grava mutation.json
# gremlins não lida bem com o padrão "./..." do Go (silenciosamente não
# reporta nada); passar "." faz ele recursar no módulo inteiro sozinho.
# examples/ é excluído por ser código de demonstração, não a biblioteca.
	@go run $(GREMLINS) unleash -E 'examples/.*' -o mutation.json .

test-mutation-dry-run: ## Lista os mutantes sem rodar os testes (bem mais rápido)
	@go run $(GREMLINS) unleash --dry-run -E 'examples/.*' .

check: lint arch-lint test-race ## Roda lint + arch-lint + testes com race detector (o que o CI deveria rodar no mínimo)

clean: ## Remove artefatos de build/teste
	@rm -rf bin coverage.out coverage.html mutation.json
