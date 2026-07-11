<div align="center">
  <img src="./static/arnon.svg" height="105" alt="Arnon Logo" />
</div>

<div align="center">
  <strong>
    Uma fundação minimalista, conceitual e baseada em padrões para a borda.
  </strong>
</div>

---

## Instalação

```sh
go get github.com/Casara/arnon
```

Requer Go 1.26 ou superior.

## Início rápido

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/openapi"
)

const readHeaderTimeout = 5 * time.Second

type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
	ID string `json:"id"`
}

func createUser(
	_ context.Context,
	request CreateUserRequest,
) (CreateUserResponse, error) {
	return CreateUserResponse{ID: "usr_123"}, nil
}

func main() {
	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	router := routing.NewRouter(
		routing.WithOpenAPI(openapi.NewRegistry(generator)),
	)

	router.POST("/users", httpx.Endpoint(
		createUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			OpenAPI: &openapi.Operation{
				Summary:       "Create a user",
				SuccessStatus: http.StatusCreated,
			},
		},
	))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(nil))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	log.Fatal(server.ListenAndServe())
}
```

Esse handler, sem nenhum código adicional, ganha automaticamente:

* binding de path, query, header e JSON body;
* validação da request (`validate:"required,email"`) com resposta
  `400 application/problem+json` no formato RFC 9457;
* mapeamento de erros do handler para Problem Details;
* schema OpenAPI 3.2 gerado por reflection para request e response;
* respostas padrão `400`/`500` documentadas automaticamente;
* UI de documentação (Stoplight Elements) em `/docs`.

Um exemplo completo e executável está em
[examples/basic](examples/basic/main.go):

```sh
go run ./examples/basic
```

## Visão geral

O `arnon` é composto por pacotes independentes, cada um com uma
responsabilidade única:

| Pacote                 | Responsabilidade                                             |
| ---------------------- | -------------------------------------------------------------|
| `problem`               | Erros HTTP no formato RFC 9457 (Problem Details).            |
| `validation`            | Validação de requests, com registro de regras customizadas.  |
| `openapi`               | Geração de schemas e do documento OpenAPI a partir de tipos Go. |
| `httpx`                 | Endpoint tipado: binding, validação, serialização e erros.   |
| `httpx/binding`         | Binding de path, query, header e JSON body.                  |
| `httpx/routing`         | Router baseado em `net/http.ServeMux`, com grupos e middleware. |
| `httpx/middleware`      | Middlewares padrão (CORS, recovery, request ID, etc).         |
| `observability`         | Abstrações finas sobre a API do OpenTelemetry.                |
| `observability/otel`    | Configuração e inicialização do SDK do OpenTelemetry.         |

O grafo de dependências permitido entre esses pacotes está documentado
em [.go-arch-lint.yml](.go-arch-lint.yml) e é verificado em CI.

Mais contexto sobre decisões arquiteturais está em
[docs/architecture/project-context.md](docs/architecture/project-context.md).

## Validação customizada

Regras de validação customizadas são registradas uma única vez, em
qualquer ponto de bootstrap da aplicação, e passam a valer para todo
validador criado a partir daí — tanto em runtime quanto na geração do
schema OpenAPI:

```go
validation.RegisterCustomRule(validation.CustomRule{
	Tag:  "cpf",
	Func: validateCPF,
	Schema: &validation.SchemaEffect{
		Format:  "cpf",
		Pattern: `^\d{11}$`,
	},
	Code: "invalid_cpf",
	Message: func(param string) string {
		return "invalid CPF"
	},
})
```

## Desenvolvimento

```sh
make help              # lista todos os comandos
make test              # testes
make test-race         # testes com detector de race conditions
make coverage          # gera coverage.html
make lint              # golangci-lint (v2, versão fixa)
make arch-lint         # valida o grafo de dependências entre pacotes
make test-mutation     # testes de mutação (gremlins), grava mutation.json
make check             # lint + arch-lint + testes com race (mínimo esperado antes de um PR)
```

As versões das ferramentas (`golangci-lint`, `go-arch-lint`, `gremlins`)
são fixas no `Makefile` via `go run pkg@versão`, para não precisar de
instalação global nem poluir o `go.mod` do módulo com dependências que
só existem em tempo de desenvolvimento.

Convenções de estilo estão documentadas em
[docs/coding-style.md](docs/coding-style.md).
