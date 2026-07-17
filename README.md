<div align="center">
  <img src="./static/arnon.svg" height="105" alt="Arnon Logo" />
</div>

<div align="center">
  <strong>
    Uma fundação minimalista, conceitual e baseada em padrões para a borda.
  </strong>
</div>

---

## Por que Arnon

`arnon` não começou como "vamos construir um framework". Começou como
uma API de um sistema financeiro real, sobre o fuego. Em algum ponto
ficou difícil aplicar RFC 9457 (Problem Details) junto com RFC 6901
(JSON Pointer, pra apontar o campo com erro) do jeito que a spec
realmente define — o framework brigava com o padrão em vez de seguir
ele. Testar outro framework provavelmente só adiaria o mesmo tipo de
atrito pra depois, então a solução foi descer pra stdlib (`net/http`)
direto. A estrutura resultante cresceu reaproveitável o bastante pra
virar lib, depois framework.

O objetivo declarado é ser o menos opinativo possível — quem usa deve
conseguir manter as escolhas padrão do `arnon` ou trocá-las pelas
próprias, sem muita complexidade pra isso. Não achamos isso
inteiramente alcançável: no momento em que qualquer decisão de
composição é tomada (ex. `Endpoint()` juntar binding, validação, erro
e OpenAPI numa peça só), isso já é opinião. O critério que fica, então,
não é "opinativo ou não", é **de onde vem cada opinião** e **quão caro
é trocá-la**:

* **Opinião vem de RFC quando existe uma, não de gosto.** O campo que
  aponta a origem de um erro de validação usa RFC 6901 (JSON Pointer)
  de verdade, escaping de `~`/`/` incluído (RFC 6901 §3) — não uma
  notação própria inventada pro `arnon`. A conformidade também não
  para no formato de erro (RFC 9457): `Forwarded` (RFC 7239) em vez de
  só `X-Forwarded-For`, ETag/conditional GET de verdade (RFC 9111),
  negociação de `Accept`/`Accept-Encoding` com parsing real de `q=`
  (RFC 9110), semântica exata de preflight em `CORS`, OpenAPI 3.2.0
  desde o design inicial. Detalhe completo, RFC por RFC — incluindo o
  que ainda não está 100% conforme — em
  [docs/architecture/rfc-compliance.md](docs/architecture/rfc-compliance.md).
* **Onde não existe RFC, ainda assim o padrão da linguagem — não algo
  reinventado.** `arnon` roda em cima de `net/http.Server` direto:
  `Router` implementa `http.Handler`, então é só
  `&http.Server{Handler: router}` (ver o "Início rápido" acima) — sem
  motor HTTP próprio substituindo o da stdlib. Isso herda de graça
  manutenção/patch de segurança do próprio time do Go, compatibilidade
  com qualquer middleware `http.Handler` já existente, `httptest`,
  `net/http/pprof`, HTTP/2 automático com TLS.
* **Toda opinião embutida vem com uma forma barata de trocar.** Ordem
  de middleware: `middleware.BuildChain` garante a ordem recomendada
  por código, mas `ChainConfig.Extra`/`ChainAnchor` deixa inserir
  middleware customizada numa posição específica sem editar a cadeia
  embutida (detalhe em
  [docs/architecture/project-context.md](docs/architecture/project-context.md),
  seção "Ordem dos middlewares"). Validação: `validation.RegisterCustomRule`
  é o único mecanismo — não existe um segundo caminho paralelo pra
  registrar regra customizada. Rate limit: o algoritmo (janela
  deslizante) é separado do storage via a interface `LimitCounter`,
  trocável por um backend compartilhado (Redis etc.) sem tocar no
  algoritmo.

### Comparado a huma e fuego, especificamente

huma e fuego **não são os frameworks Go mais usados** — Gin domina a
adoção real (~48%, ~88k stars), seguido de Echo e Fiber, ordens de
grandeza à frente de huma (~4.2k stars) e fuego (~1.8k). A comparação
abaixo não é com eles porque são os mais populares, e sim porque são os
únicos dois com a mesma proposta central do `arnon` (handler tipado →
OpenAPI gerado automaticamente por reflection, sem passo de anotação
ou geração à parte). Gin/Echo/Fiber resolvem outro problema — roteamento
e middleware de baixo nível, binding manual (`c.ShouldBindJSON`), sem
RFC 9457 nativo, e OpenAPI (quando usado) vem de comentário-anotação no
código via `swaggo/swag`, não do tipo Go em si — então uma comparação
direta nos pontos abaixo não seria justa nem informativa pra eles;
comparar o `arnon` com o Gin nesses termos seria como comparar bicicleta
e carro pelo motor.

| Aspecto | [huma](https://github.com/danielgtaylor/huma) | [fuego](https://github.com/go-fuego/fuego) | `arnon` |
| --- | --- | --- | --- |
| Formato de erro | RFC 9457 nativo | RFC 9457 nativo | RFC 9457 nativo — **não é diferencial**, é piso esperado hoje |
| Campo do erro de validação | notação de ponto ad-hoc (`"body.title"`) | namespace interno do `validator/v10` (`err.StructNamespace()`, nome de tipo/campo Go, não a tag `json`) | RFC 6901 (JSON Pointer), com escaping |
| Versão OpenAPI gerada | 3.1 (`kin-openapi`, sem 3.2 ainda) | ~3.0 (3.1/3.2 não confirmado) | 3.2.0 — vantagem com prazo de validade curto, é questão de tempo até o resto do ecossistema alcançar |
| Servidor/router | bring-your-own — `net/http` via `humago`, mas também `fasthttp` via `humafiber` (aí perde a compatibilidade com o ecossistema `net/http`) | `net/http` direto, mesma escolha do `arnon` | `net/http` direto, mas router próprio — não plugável num `gin.Engine`/`echo.Echo` já existente |

### Onde o `arnon` não é a escolha certa hoje

* **Sem histórico de produção.** É um projeto novo, sem track record —
  diferente de huma (mais maduro, comunidade maior) ou fuego. Se
  maturidade/battle-testing pesa mais que as diferenças acima, considere
  as alternativas.
* **Router próprio é a opinião mais cara de trocar hoje.** Diferente
  das outras opiniões desta seção, essa não veio de RFC nenhuma (é
  arquitetura pura) e ainda não tem um caminho de baixo atrito pra
  evitá-la — ver tabela acima.

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

Exemplos completos e executáveis estão em `examples/cmd`:

* [examples/cmd/basic](examples/cmd/basic/main.go) — exatamente o
  handler acima, sem nenhum middleware.
* [examples/cmd/middleware](examples/cmd/middleware/main.go) — o
  mesmo handler com o stack completo de middlewares do arnon (CORS,
  rate limiting, compressão, security headers, etc).
* [examples/cmd/observability](examples/cmd/observability/main.go) —
  o mesmo handler com tracing e métricas via OpenTelemetry, exportando
  de verdade para um OTel Collector local (subido com
  `docker compose`).

```sh
go run ./examples/cmd/basic
# ou
go run ./examples/cmd/middleware
# ou (requer docker compose -f examples/cmd/observability/docker-compose.yml up)
go run ./examples/cmd/observability
```

## Visão geral

O `arnon` é composto por pacotes independentes, cada um com uma
responsabilidade única:

| Pacote                | Responsabilidade                                                                                                       |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `problem`             | Erros HTTP no formato RFC 9457 (Problem Details).                                                                      |
| `validation`          | Validação de requests, com registro de regras customizadas.                                                            |
| `openapi`             | Geração de schemas e do documento OpenAPI a partir de tipos Go.                                                        |
| `httpx`               | Endpoint tipado: binding, validação, serialização e erros.                                                             |
| `httpx/binding`       | Binding de path, query, header e JSON body.                                                                            |
| `httpx/routing`       | Router baseado em `net/http.ServeMux`, com grupos e middleware.                                                        |
| `httpx/middleware`    | Middlewares padrão (CORS, recovery, request ID, logging, rate limiting, throttle, compressão, security headers, etc). |
| `observability`       | Abstrações finas sobre a API do OpenTelemetry.                                                                         |
| `observability/otel`  | Configuração e inicialização do SDK do OpenTelemetry.                                                                  |

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

## Skill para assistentes de IA

[skills/using-arnon/SKILL.md](skills/using-arnon/SKILL.md) documenta,
num formato que ferramentas de IA conseguem carregar sob demanda
(Claude Skill), como usar o `arnon` idiomaticamente: padrão de
endpoint, binding, validação, erros RFC 9457 e ordem de middleware.
Pra usar num projeto que depende do `arnon`, copie o diretório pra
dentro dele:

```sh
cp -r skills/using-arnon <seu-projeto>/.claude/skills/using-arnon
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
