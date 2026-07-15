# Foundation Go - Contexto do Projeto

## Visão Geral

O objetivo deste projeto é criar uma foundation moderna para APIs e microsserviços em Go, com foco em:

* Excelente experiência para desenvolvedores.
* Forte integração com OpenAPI.
* Observabilidade de primeira classe.
* Compatibilidade com Clean Architecture, DDD e Hexagonal Architecture.
* Pouco boilerplate.
* Convenções sensatas.
* Componentes independentes e desacoplados.
* Facilidade de testes.
* Preparação para uso em produção.

A intenção futura é que a foundation possa ser distribuída como biblioteca open source para a comunidade Go.

---

# Princípios Arquiteturais

## Simplicidade antes de abstração

Abstrações só devem ser adicionadas quando houver ganho real.

Evitar over-engineering.

---

## Convenção sobre configuração

O framework deve inferir o máximo possível através de:

* reflection
* tags
* validators
* tipos Go

A configuração explícita deve existir apenas para sobrescrever comportamentos.

---

## OpenAPI híbrida

A documentação OpenAPI deve ser gerada automaticamente sempre que possível.

O desenvolvedor pode complementar ou sobrescrever metadados manualmente.

Exemplo:

* request body gerado automaticamente
* responses padrão geradas automaticamente
* schemas gerados automaticamente
* operation customizada quando necessário

---

## RFC 9457 como padrão de erro

Todos os erros HTTP devem convergir para Problem Details.

O framework utiliza RFC 9457 como padrão oficial de representação de erros.

---

## OpenAPI 3.2.0

A versão adotada é OpenAPI 3.2.0.

Motivos:

* O projeto ainda não é público.
* É possível adotar recursos mais modernos da especificação.
* Quando a foundation estiver madura, a versão deverá estar mais amplamente suportada.

---

# Estado Atual

## HTTP

Implementado:

* Router
* Route Groups
* Middleware Chain
* Endpoint helper
* Request binding
* Request validation
* JSON responses
* Problem Details
* Problem Mapper

---

## Endpoint Helper

Os endpoints utilizam uma assinatura tipada.

Exemplo conceitual:

```go
func(
    context.Context,
    RequestDTO,
) (
    ResponseDTO,
    error,
)
```

O endpoint realiza automaticamente:

* binding
* validação
* serialização
* tratamento de erros
* mapeamento para Problem Details

Os defaults (validador padrão, `DefaultProblemMapper`, status 200) vêm de
`EndpointConfig.WithDefaults()`, chamado internamente por `Endpoint()`.
Nenhum deles precisa ser configurado manualmente para o caso comum.

### Registro no OpenAPI é opt-in por endpoint

Diferente do binding/validação, a rota só entra no documento OpenAPI
gerado se `EndpointConfig.OpenAPI` for preenchido (mesmo que com um
`&openapi.Operation{}` vazio). Isso é intencional: o desenvolvedor decide
explicitamente quais rotas são públicas na documentação.

### `SuccessStatus` existe em dois lugares

`EndpointConfig.SuccessStatus` (o status HTTP que o handler de fato
retorna) e `openapi.Operation.SuccessStatus` (o status que o documento
OpenAPI gerado descreve como resposta de sucesso) são campos
independentes. Hoje é responsabilidade do desenvolvedor mantê-los
sincronizados manualmente; ver `examples/cmd/basic/main.go`. Uma
unificação futura desses dois campos é candidata a melhoria.

---

## Validation

A validação ocorre através de validators. O validador padrão
(`validation.Default()`) usa `github.com/go-playground/validator/v10`
por baixo.

As informações dos validators são reutilizadas na geração OpenAPI.

### Custom Validators

Regras de validação customizadas (tags que o `validator/v10` não conhece
nativamente) são registradas uma única vez, via
`validation.RegisterCustomRule(rule)`, tipicamente no bootstrap da
aplicação. Um único registro alimenta três pontos ao mesmo tempo, que
antes eram desconectados:

1. **Runtime**: a `Func` da regra é aplicada automaticamente a todo
   validador criado por `validation.New()`/`validation.Default()` a
   partir do momento do registro.
2. **Mapeamento de erro**: `Code` e `Message` da regra definem o
   código/detail retornados em `problem.ValidationError` quando a regra
   falha, em vez do fallback genérico `validation_failed`.
3. **OpenAPI**: `Schema` (um `*validation.SchemaEffect` com `Format` e/ou
   `Pattern`) enriquece o schema gerado para campos que usam a tag,
   assim como acontece hoje para `email`/`uuid`/`url`.

Antes dessa mudança, não havia acesso à instância interna do
`validator.Validate` usada por `PlaygroundValidator`, então não existia
forma de registrar uma regra customizada na aplicação; e mesmo que
existisse, a geração de OpenAPI (que reprocessa a tag `validate` de forma
independente) não teria como saber da nova regra.

---

# OpenAPI

## Estado Atual

Implementado:

* geração automática de schemas
* geração automática de request body
* geração automática de responses
* geração automática de parâmetros query
* geração automática de parâmetros path
* geração automática de parâmetros header
* geração automática de schemas de erro

---

## Recursos de Schema

Implementados:

* type
* format
* description
* nullable
* deprecated
* readOnly
* writeOnly
* default
* example
* enum
* required
* properties
* additionalProperties
* items
* minimum
* maximum
* exclusiveMinimum
* exclusiveMaximum
* minLength
* maxLength
* minItems
* maxItems
* pattern

---

## Inferência Automática

O framework infere informações automaticamente a partir dos validators.

Atualmente:

### email

```go
validate:"email"
```

↓

```yaml
format: email
```

---

### uuid

```go
validate:"uuid"
```

↓

```yaml
format: uuid
```

---

### url

```go
validate:"url"
```

↓

```yaml
format: uri
```

---

## Examples

Examples são convertidos para o tipo correto.

Exemplos:

```go
example:"1"
```

↓

```yaml
example: 1
```

---

```go
example:"true"
```

↓

```yaml
example: true
```

---

```go
example:"1.5"
```

↓

```yaml
example: 1.5
```

---

## Default

Defaults também são convertidos para o tipo correto.

Exemplos:

```go
default:"20"
```

↓

```yaml
default: 20
```

---

## Tags OpenAPI

Foi adotado o modelo OpenAPI 3.2.

Campos suportados:

* name
* summary
* description
* externalDocs
* parent
* kind

Kinds suportados:

* nav
* badge
* audience

---

## UI da Documentação

Ferramenta adotada:

Stoplight Elements

Motivos:

* melhor experiência visual
* suporte moderno à OpenAPI
* suporte mais avançado que Swagger UI

---

### Recursos atuais

* título customizável
* logo customizável
* favicon customizável
* modo embed
* modo CDN

---

# Problem Details

## Padrão

RFC 9457

---

## Schemas

Implementados:

### Problem

Representa um erro HTTP.

---

### ValidationError

Representa um erro individual de validação.

---

### ValidationSource

Representa a origem do erro.

Exemplo:

```json
{
  "in": "body",
  "field": "/name"
}
```

---

## Responses automáticas

Endpoints recebem automaticamente:

### 400

Bad Request

```http
application/problem+json
```

---

### 500

Internal Server Error

```http
application/problem+json
```

---

## Examples

Cada response possui seu próprio exemplo.

Exemplo:

400 → erro de validação

500 → erro interno

---

# Middleware

Todos em `httpx/middleware`, construídos como `routing.Middleware`
(`func(http.Handler) http.Handler`), aplicados via `Router.Use`
(global, roda antes do roteamento) ou `Group.Use` (por-grupo).

## Implementado

* **CORS** — configurável (`CORSConfig.AllowedOrigins`, etc).
* **Logging** — logger estruturado (`slog`), enriquecido com
  `request_id`/`real_ip`/`trace_id`/`span_id` quando os middlewares
  correspondentes estão instalados.
* **RealIP** — extrai IP do cliente (`X-Forwarded-For`, `X-Real-IP`,
  `RemoteAddr`), disponível via `RealIPFromContext`.
* **RequestID** — gera/propaga `X-Request-Id`, disponível via
  `RequestIDFromContext`.
* **Recover** — recupera de panics, converte em Problem Details 500.
* **Timeout** — timeout de requisição via `http.TimeoutHandler`.
* **StripSlashes** / **RedirectSlashes** — duas formas de lidar com
  barra final no path: `StripSlashes` normaliza em silêncio (sem round
  trip), `RedirectSlashes` redireciona (308, preserva método e body).
  Ambos precisam rodar como middleware *global* (pré-roteamento) pra
  funcionar — ver nota em "Decisões Importantes" sobre
  `Router.ServeHTTP`. Não instalar os dois ao mesmo tempo.
* **Compress** — `Compress(level int, types ...string)`, portado do
  `middleware.Compress` do chi. Só comprime quando o `Content-Type` da
  *resposta* (não da request) bate com `types` (ou a lista padrão de
  tipos textuais/JSON quando `types` é vazio; sufixo `/*` casa
  subtipos, ex. `text/*`) — evita gastar CPU comprimindo conteúdo que
  não se beneficia (imagens, etc). `level` inválido gera panic na
  criação do middleware (erro de configuração, não de runtime). Remove
  `Content-Length` da resposta quando compressão é aplicada.
* **NoCache** — portado do `middleware.NoCache` do chi: além dos
  headers de resposta (`Cache-Control` completo, `Pragma`,
  `X-Accel-Expires`, `Expires` no epoch Unix), também remove da
  *request* os headers condicionais (`ETag`, `If-Modified-Since`,
  `If-Match`, `If-None-Match`, `If-Range`, `If-Unmodified-Since`) antes
  de chamar o handler — evita que qualquer código downstream responda
  de forma condicional/cacheada, contradizendo a intenção do
  middleware.
* **AllowContentType** — allow-list de `Content-Type` aceito na
  request, 415 caso contrário. Requests sem `Content-Type` passam
  (binding já tolera corpo ausente).
* **MaxBodyBytes** — limite de tamanho de request body. Quando
  `Content-Length` é conhecido e já excede o limite, rejeita
  imediatamente com 413. Quando não (chunked, ou client mentindo sobre
  o tamanho), usa `http.MaxBytesReader` como segunda linha de defesa;
  o estouro só é percebido durante a leitura (dentro do binding JSON),
  mas ainda assim vira 413 corretamente, via
  `problem.ValidationErrorCode.StatusOverride()` — ver "Erros de
  binding não carregam status HTTP" em "Decisões Importantes".
* **SecureHeaders** — `X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy` sempre; `Strict-Transport-Security` só se
  configurado explicitamente (HSTS quebra desenvolvimento local em
  HTTP puro se ligado por padrão).
* **Throttle** — limite de requisições *concorrentes* (semáforo), com
  backlog opcional (`BacklogLimit`/`BacklogTimeout`) pra enfileirar em
  vez de rejeitar na hora. Não é rate limiting por tempo — ver
  `RateLimit` pra isso.
* **RateLimit** — rate limiting de verdade
  (`RequestLimit`/`WindowLength` por chave de cliente, `KeyFunc` com
  default `RealIPFromContext` → `RemoteAddr`, canonicalizada via
  `CanonicalizeIP`). Algoritmo sliding-window-counter adaptado do
  `go-chi/httprate`: duas janelas fixas (atual e anterior) por chave,
  com a contagem da janela anterior ponderada pela sobreposição com a
  janela deslizante atual. O algoritmo (`checkRateLimit`) é separado do
  storage pela interface `LimitCounter`
  (`Config`/`Increment`/`IncrementBy`/`Get`), espelhando de propósito a
  interface homônima de `go-chi/httprate` — um backend já escrito pra
  httprate (ex. `go-chi/httprate-redis`) precisa de mudanças triviais
  pra servir o arnon. `RateLimitConfig.Counter` nil usa o default em
  memória (`NewLocalLimitCounter`, exportada): memória fica limitada
  sozinha, janelas antigas são descartadas em bloco (não chave por
  chave) sempre que o tempo avança pra uma nova janela, então chaves
  inativas são removidas automaticamente em até duas janelas, sem
  precisar de eviction/TTL manual — mas só é correto pra uma instância
  única; deployments com múltiplas instâncias precisam de um
  `LimitCounter` com storage compartilhado (Redis, Valkey, Memcached,
  ...), implementado como módulo Go separado (o núcleo do arnon nunca
  depende de um backend de storage específico). Erro do `Counter`
  (`Get`/`IncrementBy`) vira `problem.Problem` via
  `RateLimitConfig.OnCounterError` (default: 503 Service Unavailable,
  sem vazar a mensagem do erro; configurável). `CanonicalizeIP` reduz
  endereços IPv6 ao prefixo /64 (um cliente IPv6 controla um /64 inteiro
  via SLAAC; sem isso ele rotacionaria endereço dentro do próprio
  bloco pra escapar do limite). Response inclui
  `X-RateLimit-Limit`/`X-RateLimit-Remaining`/`X-RateLimit-Reset`
  sempre, e `Retry-After` (RFC 6585) no 429. Implementação em
  `httpx/middleware/rate_limit.go`, sem dependência externa (só
  `sync`/`time`/`net`/`math` da stdlib no core; adaptadores de storage
  externo ficam fora do módulo).

## Planejado / adiado

* **Autenticação (Bearer/Basic)** — adiado, ver "Segurança" abaixo.

---

# Decisões Importantes

## Middleware global envolve o mux inteiro, não cada rota

`Router.Use` (middleware global) é aplicado em `Router.ServeHTTP`,
envolvendo o `mux` inteiro — não em `router.register`, por rota. Isso é
o que permite middleware pré-roteamento (`StripSlashes`,
`RedirectSlashes`) funcionar, e faz com que rotas não encontradas
(404) também passem por `RequestID`/`Logging`/`RateLimit`/etc.
Middleware de grupo (`Group.Use`) continua aplicado por-rota em
`router.register`, já que `net/http.ServeMux` não tem noção de
prefixo. Não volte a mesclar `router.middlewares` dentro de
`register()` — duplicaria a execução.

## Erros de binding não carregam status HTTP por padrão

`httpx.Endpoint` mapeia todo erro de `binding.Decode` pra 400
(`writeValidationProblem`, em `httpx/endpoint.go`), independente do
código específico do `problem.ValidationError`. A exceção é
`problem.ValidationErrorCode.StatusOverride()`
(`problem/validation_code.go`): se qualquer erro tiver um código com
override (hoje só `ValidationCodePayloadTooLarge` → 413), esse status
substitui o 400 padrão. É o que faz `MaxBodyBytes` conseguir devolver
413 mesmo quando o corpo estoura durante a leitura (chunked), sem
precisar mudar a assinatura de `binding.Decode`. Ao adicionar um novo
código de validação que deveria implicar um status diferente de 400,
adicione o caso em `StatusOverride()` em vez de inventar outro
mecanismo.

## Ponteiros em Schemas

Properties utilizam ponteiros.

Exemplo:

```go
Properties map[string]*Schema
```

Motivo:

Evitar cópias desnecessárias e permitir estruturas recursivas.

---

## AdditionalProperties

Utiliza:

```go
AdditionalProperties *Schema
```

---

## Receivers

Preferência por receivers de ponteiro.

Motivos:

* evitar cópias
* consistência
* compatibilidade com linter recvcheck

---

## Stoplight

Foi escolhido Stoplight Elements ao invés de Swagger UI.

---

## OpenAPI híbrida

A geração automática continua sendo a principal estratégia.

Customizações devem complementar a geração automática, nunca exigir repetição de configuração.

---

# Recursos Planejados

## Observabilidade (Prioridade Máxima)

### OpenTelemetry

Tracing:

* HTTP Server Tracing
* Trace Propagation
* Route Attribution
* Error Attribution

Metrics:

* Request Count
* Request Duration
* Active Requests

Contexto:

* Trace ID
* Span ID
* Request ID

**Nota**: apesar do título da seção, tudo acima já está implementado
(`observability`/`observability/otel`), não é mais "planejado" —
`routing.WithInstrumentation(otel.NewHandler)` dá tracing HTTP
automático (via `otelhttp`, com atribuição de rota e propagação de
contexto), `otel.Initialize` com `MetricsEnabled: true` habilita
métricas HTTP automáticas mais métricas de runtime do Go, e
`observability.TraceID`/`SpanID` correlacionam trace_id/span_id nos
logs estruturados (`httpx/middleware/logging.go`). Demonstrado e
validado fim a fim (trace exportado batendo com o log da aplicação,
métricas customizadas com exemplars apontando pro trace exato) em
`examples/cmd/observability`, incluindo um OTel Collector local via
Docker Compose. O que falta de verdade é cobertura de teste
automatizado de `observability`/`observability/otel` (0% hoje, ver
NOTES.md), não a funcionalidade em si.

---

## Health Endpoints

* /health
* /ready
* /live

Compatíveis com Kubernetes.

---

## Segurança

Autenticação:

* Bearer Token
* Basic Auth

Autorização:

* abstração de policies

---

## Configuração

* leitura de env vars
* defaults
* validação de configuração

---

## Testes

Implementado:

### Unitários

* cobertura dos pacotes centrais: `validation`, `openapi`, `problem`,
  `httpx` e `httpx/binding`.
* `httpx`: testes fim-a-fim via `httptest`, cobrindo binding, validação,
  mapeamento de erro (default e customizado) e o caminho de sucesso.

Planejado:

### OpenAPI (Golden Tests)

* Golden Tests

### Unitários (pendente)

* cobertura de `httpx/middleware` e `observability`/`observability/otel`

### Mutação

* validação de robustez

---

## Qualidade e Ferramentas

Implementado:

* `.golangci.yml`: conjunto curado de linters (não `--enable-all`),
  ajustado ao estilo do projeto (ex.: `funlen`/`cyclop` com limites
  compatíveis com o formato vertical adotado; `ireturn` permitindo os
  retornos de interface que são decisão de design, como
  `validation.Validator`).
* `.go-arch-lint.yml`: modela o grafo de dependências real entre os
  pacotes do `arnon` e falha o build se uma dependência não permitida
  for introduzida.
* `examples/`: exemplos executáveis organizados como `cmd`+`internal`.
  `examples/cmd/basic` (`go run ./examples/cmd/basic`) é o mínimo
  possível — endpoint tipado, validação, OpenAPI, zero middleware.
  `examples/cmd/middleware` (`go run ./examples/cmd/middleware`) é o
  mesmo endpoint com o stack completo de middlewares (CORS, rate
  limit, compressão, security headers, etc).
  `examples/cmd/observability` (`go run ./examples/cmd/observability`)
  é o mesmo endpoint com `routing.WithInstrumentation(otel.NewHandler)`,
  métricas customizadas (`observability.Counter`/`Histogram`) e logs
  correlacionados por trace_id/span_id, exportando de verdade via
  OTLP/gRPC pra um OTel Collector local subido por
  `docker compose -f examples/cmd/observability/docker-compose.yml up`
  (config em `otel-collector-config.yaml`, exporter `debug` — imprime
  cada trace/métrica recebido no próprio log do collector, sem precisar
  de Jaeger/Prometheus pra validar a integração). Também trata
  shutdown gracioso (`SIGINT`/`SIGTERM`) explicitamente, ao contrário
  dos outros dois exemplos: é o que garante o flush de spans/métricas
  pendentes no SDK antes do processo sair. Código comum aos três
  exemplos (logger, registro de custom validators, o handler de
  exemplo) mora em `examples/internal/*`, não importável de fora de
  `examples/` pela regra do Go. Todos compilados e exercitados via
  `hurl --test` como parte da validação do projeto; o
  `examples/cmd/observability` foi validado também com o collector de
  verdade rodando (trace exportado batendo bit a bit com o trace_id/
  span_id logado pela aplicação, métricas customizadas com exemplars
  apontando pro trace exato).

---

# Melhorias Futuras OpenAPI

Ainda não prioritárias.

* operationId
* examples múltiplos
* discriminator
* pattern automático
* security schemes
* callbacks
* webhooks
* links
* XML
* const
* automatic tags

---

# Objetivo de Curto Prazo

Implementar observabilidade baseada em OpenTelemetry.

Escopo inicial:

* tracing HTTP
* propagação de contexto
* trace id
* span id
* associação automática de rotas
* marcação automática de erros

Após tracing:

* métricas
* health endpoints

---

# Objetivo de Longo Prazo

Tornar a foundation uma alternativa moderna para construção de APIs e microsserviços em Go, oferecendo:

* OpenAPI de primeira classe
* observabilidade nativa
* baixo boilerplate
* excelente experiência de desenvolvimento
* componentes independentes
* forte integração com arquiteturas modernas
* preparação para produção desde o início
  """
