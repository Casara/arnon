<!-- markdownlint-disable-next-line MD041 -- deliberadamente sem H1: o logo em imagem é o título. -->
<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./static/arnon-dark.svg" />
    <img src="./static/arnon-light.svg" height="105" alt="Arnon Logo" />
  </picture>
</div>

<div align="center">
  <strong>
    Uma fundação minimalista, conceitual e baseada em padrões para a borda.
  </strong>
</div>

<div align="center">

[![CI](https://github.com/casara/arnon/actions/workflows/ci.yml/badge.svg)](https://github.com/casara/arnon/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/casara/arnon.svg)](https://pkg.go.dev/github.com/casara/arnon)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

*[Read in English](README.md)*

## Por que Arnon

> "...porque o Arnom é o limite de Moabe, entre Moabe e os amorreus."
>
> — [Números 21:13](https://www.bibliaonline.com.br/ara/nm/21/13+) (ARA)

O Arnom é um rio de fronteira: a linha onde um território acaba e
começa o próximo. Uma API HTTP fica sobre esse mesmo tipo de linha — a
borda onde bytes sem tipo, vindos de fora, viram valores tipados
dentro, e onde uma falha precisa ser respondida num formato que o
outro lado já entende. É daí que vem o nome (grafado `Arnom` nas
traduções em português), é isso que a tagline quer dizer, e é por isso
que o logo são duas margens e a travessia entre elas.

`arnon` não começou como "vamos construir um framework". Começou como
uma API de um sistema financeiro real, sobre o fuego. Nesse projeto,
apontar o campo exato de um erro de validação via RFC 6901 (JSON
Pointer) — do jeito que a RFC 9457 (Problem Details) espera — não
encaixava na própria convenção de campo de erro do fuego
(`err.StructNamespace()`, nome de tipo/campo Go, não pensada pra
caminhos no formato de pointer). Isso foi um problema específico do
que esse projeto precisava, não um veredito sobre o fuego em geral. A
solução foi descer pra stdlib (`net/http`) direto, em vez de tentar
outro framework. A estrutura resultante cresceu reaproveitável o
bastante pra virar lib, depois framework — publicado pro caso de
outros esbarrarem na mesma necessidade; os trade-offs abaixo são pra
você julgar.

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
  [docs/architecture/rfc-compliance.pt-BR.md](docs/architecture/rfc-compliance.pt-BR.md).
* **Onde não existe RFC, ainda assim o padrão da linguagem — não algo
  reinventado.** `arnon` roda em cima de `net/http.Server` direto:
  `Router` implementa `http.Handler`, então é só
  `&http.Server{Handler: router}` (ver "Início rápido" abaixo) — sem
  motor HTTP próprio substituindo o da stdlib. Isso herda de graça
  manutenção/patch de segurança do próprio time do Go, compatibilidade
  com qualquer middleware `http.Handler` já existente, `httptest`,
  `net/http/pprof`, HTTP/2 automático com TLS.
* **Toda opinião embutida vem com uma forma barata de trocar.** Ordem
  de middleware: `middleware.BuildChain` garante a ordem recomendada
  por código, mas `ChainConfig.Extra`/`ChainAnchor` deixa inserir
  middleware customizada numa posição específica sem editar a cadeia
  embutida (detalhe na seção
  ["Ordem dos middlewares"](docs/architecture/project-context.pt-BR.md#ordem-dos-middlewares)).
  Validação: `validation.RegisterCustomRule`
  é o único mecanismo — não existe um segundo caminho paralelo pra
  registrar regra customizada. Rate limit: o algoritmo (janela
  deslizante) é separado do storage via a interface `LimitCounter`,
  trocável por um backend compartilhado (Redis etc.) sem tocar no
  algoritmo.

### Comparado a huma e fuego, especificamente

huma e fuego **não são os frameworks Go mais usados** — Gin domina a
adoção real (~48%, ~88k stars), seguido de Echo e Fiber, ordens de
grandeza à frente de huma (~4.3k stars) e fuego (~1.8k). A comparação
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
| Sanitização antes da validação | Não tem | Métodos de interface `InTransform`/`OutTransform` — código explícito por tipo, não recursivo em struct aninhado | Tag `sanitize` (`sanitize:"trim"`), recursiva em struct/slice/map aninhado, transformação customizada pelo mesmo padrão de registro único da validação |
| Formato de wire além de JSON | CBOR (RFC 8949) opt-in via o subpacote `formats/cbor` — isola a dependência `fxamacker/cbor/v2` ali, registrado na mesma negociação `huma.Format`/`Accept` que o JSON já usa | JSON/XML/YAML/HTML/plain text nativos via `Accept`; outros formatos exigem `WithContentTypeSerDes` escrito à mão — sem CBOR de fábrica | JSON only por design — sem negociação entre representações por endpoint; qualquer outra coisa é `http.Handler` puro (ver `examples/cmd/files`), não `httpx.Endpoint` |
| `PATCH` a partir de `GET`+`PUT` | `autopatch.AutoPatch(api)` — descobre o par automaticamente via o registro de operações próprio do huma, RFC 7386/RFC 6902 suportadas | Não tem | `httpx/patch.From(get, put, ...)` — explícito, sem introspecção de rota (`arnon` não tem uma sem violar camadas); as mesmas duas RFCs |
| Preconditions de escrita / concorrência otimista | `conditional.Params` embutido no struct de entrada + `input.PreconditionFailed(etag, modified)`, chamado pelo handler com seu próprio estado atual | Não tem | `httpx/precondition.Check`/`CheckRequest` — mesmo formato "handler fornece o estado atual" do huma, `412`/`428` opt-in; `httpx/patch.From` chama internamente sobre o `If-Match` do próprio cliente antes de aplicar qualquer patch |
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
go get github.com/casara/arnon
```

Requer Go 1.26 ou superior.

A integração com OpenTelemetry é um **módulo separado**, então o SDK do OTel e
os exportadores OTLP/gRPC ficam fora do seu grafo de dependências a menos que
você os peça:

```sh
go get github.com/casara/arnon/observability/otel
```

Você só precisa dele para exportar traces e métricas. O `observability` em si —
os contadores, histogramas e atributos que os middlewares registram — vem no
módulo principal. Ver
[docs/architecture/adr-0001-module-layout.md](docs/architecture/adr-0001-module-layout.md).

## Início rápido

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"
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
		routing.WithOpenAPI(generator),
	)

	router.POST("/users", httpx.Endpoint(
		createUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			OpenAPI: &openapi.Operation{
				Summary:       "Create a user",
			},
		},
	))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(openapi.DocsConfig{}))

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
* sanitização de campos (`sanitize:"trim"`) antes da validação;
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
* [examples/cmd/files](examples/cmd/files/main.go) — download/upload
  de arquivos (PDF, CSV, XML): `http.Handler`s simples montados
  direto no router, já que `httpx.Endpoint` é JSON-only por design.
* [examples/cmd/staticfiles](examples/cmd/staticfiles/main.go) —
  servindo assets estáticos via `http.FileServer`, com `ETag`/
  `Compress` aplicados globalmente: os dois se afastam de qualquer
  requisição com header `Range`, então o suporte nativo do
  `http.FileServer` a Range/conditional-GET continua funcionando sem
  interferência — confirmado empiricamente, ver os doc comments de
  `middleware.ETag`/`middleware.Compress`.
* [examples/cmd/patch](examples/cmd/patch/main.go) — `PATCH` derivado
  de um par `GET`+`PUT` já existente via `httpx/patch.From` (RFC 7386
  JSON Merge Patch e RFC 6902 JSON Patch, escolhido pelo
  `Content-Type`), sem mudar nenhum dos dois handlers.

```sh
go run ./examples/cmd/basic
# ou
go run ./examples/cmd/middleware
# ou (requer docker compose -f examples/cmd/observability/docker-compose.yml up)
go run ./examples/cmd/observability
# ou
go run ./examples/cmd/files
# ou
go run ./examples/cmd/staticfiles
# ou
go run ./examples/cmd/patch
```

## Visão geral

O `arnon` é composto por pacotes independentes, cada um com uma
responsabilidade única:

| Pacote                | Responsabilidade                                                                                                      |
| --------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `problem`             | Erros HTTP no formato RFC 9457 (Problem Details).                                                                     |
| `sanitize`            | Transformações de dados via struct tag (trim, funções customizadas), aplicadas antes da validação.                    |
| `validation`          | Validação de requests, com registro de regras customizadas.                                                           |
| `openapi`             | Geração de schemas e do documento OpenAPI a partir de tipos Go.                                                       |
| `httpx`               | Endpoint tipado: binding, validação, serialização e erros.                                                            |
| `httpx/binding`       | Binding de path, query, header e JSON body.                                                                           |
| `httpx/routing`       | Router baseado em `net/http.ServeMux`, com grupos e middleware.                                                       |
| `httpx/middleware`    | Middlewares padrão (CORS, recovery, request ID, logging, rate limiting, throttle, compressão, security headers, etc). |
| `httpx/patch`         | Deriva um handler `PATCH` a partir de um par `GET`+`PUT` já existente (RFC 6902 / RFC 7386).                          |
| `httpx/precondition`  | Preconditions de escrita (`If-Match`/`If-Unmodified-Since`, RFC 9110 §13.1.1/§13.1.4), `428` opt-in.                  |
| `observability`       | Abstrações finas sobre a API do OpenTelemetry.                                                                        |
| `observability/otel`  | Configuração e inicialização do SDK do OpenTelemetry.                                                                 |

O grafo de dependências permitido entre esses pacotes está documentado
em [.go-arch-lint.yml](.go-arch-lint.yml) (diagrama em
[AGENTS.md](AGENTS.md#package-dependency-graph)) e é
verificado em CI.

Mais contexto sobre decisões arquiteturais está em
[docs/architecture/project-context.pt-BR.md](docs/architecture/project-context.pt-BR.md).

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

func validateCPF(field validation.FieldContext) bool {
	return cpfPattern.MatchString(field.Field().String())
}
```

`FieldContext` é interface do próprio arnon, então uma regra nunca nomeia um
tipo da biblioteca de validação por baixo e continua compilando se essa
implementação for trocada. Ela expõe o valor do campo, o parâmetro da tag, e os
structs pai e raiz para checagem cross-field.

## Sanitização

Struct tags transformam o dado da request antes da validação rodar,
pra que uma checagem tipo `required`/`min` veja o valor que o cliente
pretende, não bytes crus que por acaso satisfazem sem ter esse sentido
(ex.: `"C "` passando em `min=2` pelo comprimento sem trim):

```go
type CreateUserRequest struct {
	Email string `json:"email" validate:"required,email" sanitize:"email"`
}
```

Built-in: `trim` e `email` (trim + lowercase). Registre o seu do mesmo
jeito que uma regra de validação customizada:

```go
sanitize.RegisterFunc(
	"digitsOnly",
	sanitize.FromRegexp(regexp.MustCompile(`[^0-9]`)),
)
```

Um campo struct é sempre recursado; um campo slice/array/map precisa
que a tag comece com `dive` (`sanitize:"dive,trim"` num `[]string`),
seguindo a mesma convenção do `validate`.

## Estabilidade da API

O `arnon` é publicado como **v0.x**, e em Semantic Versioning isso tem um
significado específico que vale dizer explicitamente em vez de deixar você
inferir: **um release minor pode quebrar a API pública.** Fixe uma versão, leia
o [CHANGELOG](CHANGELOG.md) antes de atualizar, e conte com pequenos ajustes
mecânicos quando fizer isso.

O que conta como API pública: todo símbolo exportado dos módulos publicados —
`problem`, `sanitize`, `validation`, `openapi`, `httpx` e seus subpacotes,
`observability`, e `github.com/casara/arnon/observability/otel`. Não é público:
`examples/` (módulo separado, não publicado), qualquer coisa sob um diretório
`internal/`, e o texto exato de mensagens de erro e de log.

Quebras aparecem em `### Changed` ou `### Removed` no changelog, com a migração
na mesma entrada. Onde um rename puder manter a grafia antiga compilando, ele
vai — como alias depreciado, removido não antes do minor seguinte.

O caminho até o v1.0.0 é o ponto em que a superfície para de se mexer, não um
marco de funcionalidades. Uma pergunta segue aberta, e é do tipo que fica cara
de mudar depois: se o `validation.WithConfigure` — o único lugar que ainda
nomeia um tipo do `go-playground/validator/v10` numa assinatura exportada —
deve seguir como escape hatch ou ir para um subpacote. Fechada essa, o v1 vem
em seguida.

## Storage do rate limit

O `RateLimit` mantém o algoritmo de janela deslizante no `arnon` e deixa as
contagens para um `LimitCounter` que você fornece. O default é em memória, então
só é correto para uma instância; qualquer coisa com mais de uma réplica precisa
de storage compartilhado.

**O `LimitCounter` do `arnon` é a mesma interface do
[`go-chi/httprate`](https://github.com/go-chi/httprate)**, método por método e
de propósito. Interfaces em Go são estruturais, então um backend existente do
httprate funciona aqui sem adaptador:

```go
counter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
	Host: "localhost",
	Port: 6379,
})
if err != nil {
	return err
}

router.Use(middleware.RateLimit(middleware.RateLimitConfig{
	RequestLimit: 100,
	WindowLength: time.Minute,
	Counter:      counter,
}))
```

Isso já cobre Redis, e qualquer coisa que fale o protocolo dele (Valkey, por
exemplo). É também por isso que este projeto não traz backend de storage
próprio: duplicar um pacote mantido adicionaria dependência, um release a
coordenar e superfície de CVE, sem ganho para quem usa. Uma asserção de
compilação em `httpx/middleware/limitcounter_compat_test.go` impede que as duas
interfaces divirjam.

Escrever o seu funciona na direção inversa — implemente os quatro métodos e o
resultado serve os dois projetos.

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
[docs/coding-style.pt-BR.md](docs/coding-style.pt-BR.md); fluxo de branch e
convenção de commit (Conventional Commits) em
[CONTRIBUTING.pt-BR.md](CONTRIBUTING.pt-BR.md).

Histórico de mudanças em [CHANGELOG.md](CHANGELOG.md). Pra reportar
vulnerabilidade, siga [SECURITY.md](SECURITY.md) (não abra issue
pública).
