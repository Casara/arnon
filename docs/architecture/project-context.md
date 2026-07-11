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
sincronizados manualmente; ver `examples/basic/main.go`. Uma unificação
futura desses dois campos é candidata a melhoria.

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

## Implementado

### CORS

Configurável.

---

## Planejado

* Request ID
* Recovery
* Logger
* Compression
* Rate Limiting

---

# Decisões Importantes

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
* `examples/basic`: exemplo mínimo e executável (`go run
  ./examples/basic`), compilado e exercitado como parte da validação do
  projeto.

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
