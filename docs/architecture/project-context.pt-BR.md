# Foundation Go - Contexto do Projeto

*[Read in English](project-context.md)*

## Visão Geral

`arnon` é uma foundation moderna para APIs e microsserviços em Go, com foco em:

* Excelente experiência para desenvolvedores.
* Forte integração com OpenAPI.
* Observabilidade de primeira classe.
* Pouco boilerplate.
* Convenções sensatas.
* Componentes independentes e desacoplados.
* Facilidade de testes.

Distribuída como biblioteca open source para a comunidade Go. Essa é a
primeira versão pública, ainda sem histórico de uso em produção — ver
README.md pra uma comparação detalhada com alternativas mais maduras
antes de adotar pra algo crítico pro negócio.

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

* É a primeira versão pública do `arnon`, sem consumidores anteriores
  pra migrar.
* É possível adotar recursos mais modernos da especificação.
* O suporte de ferramentas à 3.2 (parsers, geradores de client, UIs de
  documentação) deve crescer com o tempo — ver a nota em "Stoplight"
  abaixo pra onde isso está hoje.

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
aplicação. Um único registro alimenta três pontos ao mesmo tempo:

1. **Runtime**: a `Func` da regra é aplicada automaticamente a todo
   validador criado por `validation.New()`/`validation.Default()` a
   partir do momento do registro.
2. **Mapeamento de erro**: `Code` e `Message` da regra definem o
   código/detail retornados em `problem.ValidationError` quando a regra
   falha, em vez do fallback genérico `validation_failed`.
3. **OpenAPI**: `Schema` (um `*validation.SchemaEffect` com `Format` e/ou
   `Pattern`) enriquece o schema gerado para campos que usam a tag,
   assim como acontece hoje para `email`/`uuid`/`url`.

A instância interna de `validator.Validate` usada por
`PlaygroundValidator` propositalmente não é exposta para registro
direto: a geração de OpenAPI reprocessa a tag `validate` de forma
independente do validador de runtime, então uma regra registrada
apenas na instância crua valeria em tempo de requisição mas ficaria
invisível pro schema gerado. `RegisterCustomRule` é o único ponto que
mantém os três sincronizados.

### Por que `ValidationError` tem `detail` + `code` + `source` + `meta`

Cada campo tem um papel deliberadamente diferente, não é redundância:

* `detail` — texto humano em inglês. Não é contrato estável: quem
  consome a API não deve fazer parsing dele.
* `code` — vocabulário estável e i18n-friendly (`required`, `min_length`,
  ...). É dissociado de propósito dos nomes internos de tag do
  `validator/v10`, pra não vazar detalhe de implementação nem quebrar
  contrato se a lib de validação por trás for trocada um dia.
* `meta` — valores estruturados da regra (ex. `min`) pra quem consome
  montar a própria mensagem localizada, sem precisar fazer parsing de
  `detail`.

### `min`/`max` é comprimento, não valor numérico

`validate:"min=1,max=100"` num `int` é um erro semântico comum:
`min`/`max` do `validator/v10` sempre significam comprimento de
string/slice/map, nunca o valor numérico em si — `validation/mapper.go`
mapeia as duas pra `ValidationCodeMinLength`/`MaxLength` incondicionalmente,
com mensagem "must contain at least/most N characters", mesmo aplicadas a
um campo numérico. Pra restringir o *valor* de um número, a tag certa é
`gt`/`gte`/`lt`/`lte`.

### Mapeamento de erro do validator: regras explícitas + fallback

`mapFieldError` (`validation/mapper.go`) mapeia um conjunto fixo de tags
conhecidas explicitamente; qualquer tag não coberta (built-in do
`validator/v10` sem mapeamento dedicado, ou uma regra custom registrada
direto no `*validator.Validate` subjacente em vez de via
`validation.RegisterCustomRule`) cai num fallback genérico
(`ValidationCodeValidationFailed`, com `meta.rule`/`meta.param`). O
fallback existe de propósito pra nunca expor a string de erro crua do
`validator/v10` (formato tipo `Key: 'Foo.Bar' Error:Field validation...`)
como `detail` — isso vazaria detalhe de implementação e quebraria a
garantia de `code` ser vocabulário estável.

---

# Sanitização

`sanitize.Apply` roda entre `binding.Decode` e a validação, dentro de
`httpx.Endpoint`, pra que um validador tipo `required`/`min` veja o
valor que o cliente realmente pretende, não bytes crus que por acaso
satisfazem a checagem sem ter esse sentido (ex.: `"C "` passando em
`min=2` pelo comprimento sem trim, mesmo o valor pretendido sendo um
único caractere).

O escopo é deliberadamente estreito: um sanitizer só faz a limpeza que
um validador precisa pra decidir corretamente. Qualquer coisa que não
afete se a validação passa - formatação de exibição, campo computado,
mascarar dado sensível numa resposta - é responsabilidade do handler,
não daqui. É também por isso que não existe um equivalente a "Output
Transformation" do fuego: o handler já tem controle total de escrita
sobre o tipo de resposta antes do `Endpoint` serializar - mascarar ou
computar um campo ali é só código Go, sem precisar de hook nenhum do
framework.

## Sintaxe da tag

`sanitize:"trim,lower"` encadeia transformações nomeadas, resolvidas
contra o registro que `sanitize.RegisterFunc` alimenta - o mesmo padrão
de registro único de `validation.RegisterCustomRule`. Built-in: `trim`
(`strings.TrimSpace`) e `email` (trim + lowercase). Deliberadamente
mínimo - não vem `lower`/`upper`/`title` etc. por padrão, já que isso é
decisão de negócio (deixar `Name` em lowercase seria errado), não
normalização universal; registre o seu via `RegisterFunc`.

`sanitize.FromRegexp(pattern)` retorna um sanitizer que remove toda
substring que casa com um `*regexp.Regexp` já compilado - o equivalente
ao `CustomCompiled` do mrz1836/go-sanitize, sem embutir o padrão dentro
da string da tag (o que colidiria com a própria sintaxe separada por
vírgula da tag, e forçaria recompilar a cada request, do jeito que o
`Custom` daquela lib faz).

## Recursão

Um campo struct é sempre recursado, do mesmo jeito que o
`validator/v10` já desce em struct aninhado automaticamente (sem
precisar de tag no campo struct em si). Um campo slice/array/map
precisa que a tag comece com `dive` pra aplicar os tokens restantes em
cada elemento, seguindo a mesma convenção do `validate`
(`sanitize:"dive,trim"` num `[]string`, `sanitize:"dive"` sozinho num
`[]SomeStruct` pra recursar nos próprios campos tagueados de cada
elemento). Limitado pelo mesmo tipo de profundidade
(`sanitize.maxDepth`, 16) que `validation.maxFieldMapDepth`, pelo mesmo
motivo: garantir término contra um struct auto-referente em vez de
recursar até estourar a pilha.

Restrito a campo `string` (e `*string`) na v1 - sem diretivas
numéricas/bool que algumas libs de sanitizer oferecem (`max`/`min`/
`def`): isso sobrepõe `validate:"gt/gte/lt/lte"` e conflitaria com o
que a tag `default` do OpenAPI já significa.

## Falha rápido, não em silêncio

`httpx.Endpoint` chama `sanitize.Prepare(reflect.TypeFor[TRequest]())`
uma vez, no momento da construção, e dá panic se alguma tag `sanitize`
alcançável a partir do tipo referenciar uma função não registrada -
espelhando a disciplina "panic na construção, nunca no meio de uma
request" do `middleware.BuildChain`. `Prepare` caminha o
`reflect.Type` estruturalmente em vez de um valor vivo justamente pra
que um campo pointer-pra-struct aninhado seja checado mesmo quando
seria nil em runtime - `Apply` (que roda por request, contra o valor
real, possivelmente nil) não consegue oferecer essa garantia, e ignora
silenciosamente uma tag não reconhecida em vez de dar erro em toda
request.

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

`required` no schema vem exclusivamente da tag `validate:"required"` —
nunca de `json:"...,omitempty"`. São preocupações independentes:
`omitempty` só controla serialização JSON (omitir campo zero-value),
não é usado como proxy de "campo opcional" no schema gerado (diferente
de alguns outros frameworks Go).

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

### `dive` redireciona constraint pro schema do elemento

```go
Tags []string `validate:"dive,min=2"`
```

↓

```yaml
type: array
items:
  type: string
  minLength: 2   # não minItems
```

`applyValidationTags` (`openapi/validation.go`) rastreia se já passou
por um `dive` na tag `validate`; a partir daí, `min`/`max`/`len`/`gt`/
`gte`/`lt`/`lte`/`oneof`/`email`/`uuid`/`url`/regra customizada
redirecionam pro `schema.Items` em vez do schema do campo — sem isso,
`applyMin`/`applyMax` só olham `schema.Type` (`"array"` com ou sem
`dive`), então `dive,min=2` virava `minItems: 2` (array com 2+
elementos) em vez de `minLength: 2` em cada elemento (o schema mentia
sobre o próprio contrato: o runtime já validava certo, só o schema
documentado é que estava errado). `dive,dive` (slice de slice) desce
dois níveis de `Items`, e um `required` depois de `dive` não marca o
campo como obrigatório no schema (não tem equivalente OpenAPI pra
"nenhum elemento pode ser zero-value").

---

### Não inferidos automaticamente: `Pattern` e `Tags`

`Schema.Pattern` só é setado quando uma regra customizada declara isso
explicitamente via `SchemaEffect.Pattern` de `RegisterCustomRule` (ver
"Custom Validators" acima) — uma tag embutida como
`validate:"regexp=^[a-z]+$"` não tem caso dedicado em
`applyValidationTags` e não produz constraint nenhum de schema por
conta própria. `Operation.Tags` funciona do mesmo jeito: sempre setado
explicitamente por operação (ver "Tags OpenAPI" abaixo), nunca
derivado automaticamente de um grupo de rota, prefixo de path ou nome
de handler.

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

## Campos de Nível de Documento

`openapi.NewGenerator(info, opts...)` recebe `GeneratorOption`s (espelha
`routing.Option`/`routing.WithOpenAPI`) pra setar campos de nível de
documento além de `Info`:

* `WithServers(...Server)` — seta `Document.Servers`, a(s) URL(s) base
  da API.
* `WithExternalDocs(*ExternalDocs)` — seta `Document.ExternalDocs`.

Veja `examples/cmd/basic` pra `WithServers` em uso.

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

## O que ainda não está implementado

Auditado contra o Object Model do OpenAPI 3.2; agrupado por quanto cada
gap importa:

### Gaps reais, vale fechar (ainda não feito)

* **`Components`** só implementa `Schemas` — que é genuinamente usado
  (registrado por tipo e referenciado via `$ref`, ver
  `Generator.registerSchema`). O Components Object da spec também
  cobre `responses`, `parameters`, `examples`, `requestBodies`,
  `headers`, `securitySchemes`, `links`, `callbacks`, `pathItems` —
  nenhum desses existe, nem como tipo não usado.
* **`Response.Headers`/`Header`/`Link` nunca são preenchidos pelo
  gerador.** `Header` (`Description`, `Required bool`, `Schema
  *Schema`) é usável por conta própria — um caller já pode preencher
  `Operation.Responses["200"].Headers[...]` na mão hoje — mas nada em
  `generator.go`/`reflection.go` o preenche automaticamente.
  `RateLimit`, `ETag`, `ServiceDesc`, `CORS`, entre outros, adicionam
  headers de resposta de verdade (`X-RateLimit-*`, `ETag`, `Link`,
  `Access-Control-Allow-Origin`, ...), e nada disso aparece no
  documento gerado. Não é um campo rápido de adicionar: o gerador
  baseado em reflection só olha pro struct Go de request/response de
  um endpoint, sem visibilidade de quais middlewares estão ativos
  naquela rota — fechar isso precisa de uma decisão de design real
  sobre como uma middleware declararia "eu adiciono esse header de
  resposta" pro gerador.

### Dependem de Auth, que já está adiado

* **`Operation.Security`/`Document.Security`** (requisito de segurança
  por operação e global) e **`Components.SecuritySchemes`** (não
  existe tipo `SecurityScheme` nenhum) — os três só ficam úteis juntos,
  e só quando o `arnon` tiver *algum* conceito de esquema de
  autenticação, que ainda não existe (Bearer/Basic auth já está
  rastreado como adiado em outro lugar). Implementar só o campo em
  `Operation` sem o resto produziria um schema que sempre parece
  não-autenticado, o que é pior que não ter o campo de jeito nenhum.
* **`Document.Webhooks`** (3.1+) e **`Operation`/`Components.Callbacks`**
  (callbacks fora de banda, estilo webhook, ligados a uma operação) —
  não existe mecanismo de registro pra nenhum dos dois, e nunca se
  discutiu o `arnon` expor endpoint/webhook de callback — prioridade
  mais baixa que Security, já que pelo menos um recurso real pedido por
  usuário (auth) já motiva fechar aquele gap, enquanto nada hoje motiva
  callbacks.
* **`Document.JSONSchemaDialect`** (3.1+, declara qual versão do JSON
  Schema os schemas do documento seguem) — sem uso já que o `arnon`
  ainda não emite palavra-chave de schema específica de um dialeto (ver
  abaixo).

### Aceito, não planejado

* **Dialeto completo do JSON Schema 2020-12**
  (`oneOf`/`anyOf`/`allOf`/`not`, `const`, `discriminator`, `xml`,
  `prefixItems`, `contentEncoding`/`contentMediaType`, `title` no nível
  do schema) — `Schema` só cobre a forma "struct/slice/map/primitivo
  simples" que um gerador code-first baseado em reflection produz
  naturalmente. Go não tem sum type nativo pra mapear pra `oneOf`,
  então esse gap rastreia uma limitação real da própria abordagem
  code-first, não um descuido — revisitar só se um caso de uso
  concreto precisar.
* **`Parameter` `style`/`explode`/`allowReserved`/`allowEmptyValue`/
  `content`** (regras de serialização de array/objeto em query
  próprias do OpenAPI) — o binding do `arnon` (`httpx/binding`) nunca
  consulta isso; ele faz bind de valor de query/header direto a partir
  da tag do struct Go, independente do que o schema gerado diz.
  Implementar isso só afetaria o que o documento *descreve*, não o que
  o `arnon` de fato aceita — baixo valor até o binding do `arnon`
  ganhar parsing consciente de `style`. `Parameter.In` também nunca
  produz `"cookie"` (binding de cookie não é implementado) nem
  `"body"` (não é uma localização válida de Parameter Object pela
  spec, pra começo de conversa; campo de body vai pra `RequestBody`).
* **`Operation.Servers`/`Operation.ExternalDocs`** (override, por
  operação, dos campos no nível do documento) — gaps reais, mas baixa
  prioridade; `ExternalDocs` em particular seria barato de adicionar (o
  tipo já existe, só não está exposto em `Operation`) se surgir a
  necessidade.

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

`field` só usa sintaxe de JSON Pointer (RFC 6901: `/name`,
`/address/city`, `/items/0/name`, `/tags/1`, `/meta/x~1y`) quando `in`
é `"body"` — é o único `in` hierárquico. O nome de cada segmento (tag
`json`, não o nome do campo Go) é escapado por `~0`/`~1` conforme RFC
6901 §3 (`validation.escapeJSONPointerToken`) — sem isso, um campo
chamado literalmente `"a/b"` viraria `/a/b`, indistinguível de dois
segmentos; o mesmo vale pra chave de map (`Meta["x/y"]` → `/meta/x~1y`,
diferente de índice de slice, que nunca precisa escapar por ser sempre
dígito). `buildFieldMap` (`validation/field_map.go`) caminha o *valor*
real da request (não só o tipo — precisa do tamanho de verdade de
slice/array/map), recursando em struct aninhado (valor ou ponteiro),
elemento de slice/array (struct ou primitivo) e entrada de map com
chave string (struct ou primitivo), compondo o pointer nível a nível —
inclusive slice/map de primitivo com `dive` (`Tags[1]`, `Meta["x"]`),
já que o `validator/v10` reporta erro de elemento sem segmento de
campo depois do índice/chave, então precisa de entrada própria no
mapa em vez de só recursão. O cruzamento com o erro do `validator/v10`
usa `FieldError.StructNamespace()` com o nome do tipo raiz removido
(`validation.structFieldNamespace`), não `StructField()` (só dá o
nome do campo folha, sem caminho); o formato de `StructNamespace()`
pra elemento de slice/map (chave crua, sem escaping, no namespace — só
o pointer final é escapado) foi confirmado empiricamente, não
assumido. Limitado a `maxFieldMapDepth` (16) níveis (struct, índice e
chave contam pro mesmo limite), só pra garantir término mesmo com um
struct auto-referente. Como agora caminha o valor de verdade, o custo
escala com o tamanho de slice/map alcançável na request — só no
caminho de erro (`mapValidationErrors` só roda depois que já existe
pelo menos um erro), não afeta request bem-sucedida. **Limite atual**:
chave de map não-string (`map[int]T`) cai no fallback de nome de campo
— JSON só tem chave string de qualquer forma, caso raro em DTO de
request. Pra
`path`/`query`/`header`
(`NewPathError`/`NewQueryError`/`NewHeaderError` em `problem/validation.go`),
`field` é sempre o nome cru do campo (`id`, `page`, `Authorization`), sem
prefixo `/` (não é JSON Pointer, não tem por quê escapar). Detalhe da
RFC em [docs/architecture/rfc-compliance.md](rfc-compliance.md).

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

* **CORS** — configurável (`CORSConfig.AllowedOrigins`, etc). Só
  intercepta `OPTIONS` com `204` quando é um preflight de verdade
  (`Access-Control-Request-Method` presente, Fetch spec §4.1); um
  `OPTIONS` "nu" cai pro `next`, chegando no `mux` (que devolve
  `405`+`Allow` real refletindo os métodos registrados pro path, ou
  aciona um handler `OPTIONS` explícito do usuário, se houver). Sempre
  adiciona `Vary: Origin` (a resposta sempre depende do `Origin` da
  request, já que `Access-Control-Allow-Origin` ecoa o valor recebido
  em vez de usar um `*` literal — necessário pra suportar
  `AllowCredentials`).
* **Logging** — logger estruturado (`slog`), enriquecido com
  `request_id`/`real_ip`/`trace_id`/`span_id` quando os middlewares
  correspondentes estão instalados.
* **RealIP** — extrai IP do cliente, checando nesta ordem:
  `Forwarded` (RFC 7239, o padrão IETF) → `X-Forwarded-For` →
  `X-Real-IP` → `RemoteAddr`. Disponível via `RealIPFromContext`.
* **RequestID** — gera/propaga `X-Request-Id`, disponível via
  `RequestIDFromContext`.
* **Recover** — recupera de panics, converte em Problem Details 500
  (`problem.NewInternal("")`, detail genérico) e loga o valor do
  panic via `observability.LoggerFromContext` — nunca inclui o valor
  bruto do panic na resposta (RFC 9457 §3.1.5).
* **Timeout** — timeout de requisição; implementação própria (não usa
  mais `http.TimeoutHandler` da stdlib) que responde com Problem
  Details em vez de texto puro no timeout.
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
  `Accept-Encoding` é interpretado de verdade (`acceptsGzip`,
  parseando `;q=` e o coringa `*`, RFC 9110 §12.5.3), não com um
  simples `strings.Contains` — um `gzip;q=0` explícito é recusa, não
  aceite. Ausência do header continua significando "não comprime"
  (default conservador que já existia, não muda com essa precisão).
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
* **ETag** — conditional GET (RFC 9111/9110 §13). Só atua em
  `GET`/`HEAD` e em respostas 2xx; bufferiza o corpo inteiro do
  handler (precisa do corpo completo pra hashear), calcula um ETag
  forte via FNV-1a 64-bit (`hash/fnv` da stdlib) e compara contra
  `If-None-Match` usando comparação fraca (ignora prefixo `W/` de
  qualquer lado, conforme RFC 9110 §13.1.2). Em caso de match (ou
  `If-None-Match: *`), responde `304 Not Modified` sem corpo; senão,
  responde o corpo completo com o header `ETag`. Um `ETag` já setado
  pelo handler é respeitado em vez de recalculado. Combinar com
  `Compress`: instale `ETag` antes (mais externo), pra hashear os
  bytes já comprimidos, consistente com o `Vary: Accept-Encoding` que
  `Compress` já seta. Combinar com `NoCache` na mesma rota anula o
  propósito dos dois. Implementação em `httpx/middleware/etag.go`.
* **ServiceDesc** — adiciona `Link: <path>; rel="service-desc"`
  (RFC 8631) em toda resposta, apontando pro documento OpenAPI (ex.
  `/openapi.json`), permitindo descoberta automática por um
  cliente/ferramenta genérico que já entende `Link` headers. Usa
  `header.Add`, não `Set`, então soma a outros `Link` que já existam
  em vez de substituí-los. Implementação em
  `httpx/middleware/service_desc.go`.

## Ordem dos middlewares

A ordem relativa das middlewares globais (`Router.Use`) importa —
várias têm dependências reais umas nas outras (contexto que uma
popula e outra lê, bytes que uma precisa ver antes da outra
transformar). Duas formas de aplicar isso, por ordem de preferência:

### `middleware.BuildChain` — ordem garantida por código, não por disciplina

`middleware.BuildChain(config middleware.ChainConfig) []routing.Middleware`
(`httpx/middleware/chain.go`) monta a cadeia global recomendada na
ordem certa, sempre — cada campo de `ChainConfig` é
opcional/independente (nil ou `false` = "não mencionado", não
"desabilitado"), mas a posição relativa de quem for incluído nunca
muda, porque quem decide a ordem é o código do `BuildChain`, não quem
chama `router.Use(...)`. Uso:

```go
router.Use(middleware.BuildChain(middleware.ChainConfig{
    Recover:       true,
    RealIP:        true,
    RequestID:     true,
    SecureHeaders: &middleware.SecureHeadersConfig{},
    RateLimit:     &middleware.RateLimitConfig{ /* ... */ },
    ETag:          true,
    Compress:      &middleware.CompressConfig{},
    CORS:          &middleware.CORSConfig{ /* ... */ },
    ServiceDescPath: "/openapi.json",
    Logger:        logger,
})...)
```

`BuildChain` também **impede em runtime** a única combinação
mutuamente exclusiva que existe hoje: setar `StripSlashes` e
`RedirectSlashes` juntos causa panic imediato (na criação da cadeia,
não no meio de uma request).

O que `BuildChain` garante e o que não garante: qualquer chamada com
o mesmo subconjunto de campos preenchidos sempre produz a mesma ordem
relativa entre eles — isso é testado de verdade em
`httpx/middleware/chain_test.go` (não só documentado), verificando
comportamento observável (`RequestID` aparecendo no log do
`Logging`, `ETag` hasheando bytes já comprimidos pelo `Compress`,
`Recover` pegando panic de qualquer lugar da cadeia, `SecureHeaders`
aparecendo mesmo numa resposta `429` do `RateLimit`). O que não é
garantido: uma cadeia montada manualmente com `router.Use(mw1, mw2,
...)`, totalmente fora do `BuildChain`, continua sendo
responsabilidade de quem escreve — não existe (nem seria razoável
construir, dado que `routing.Middleware` é só
`func(http.Handler) http.Handler`, sem identidade própria em runtime)
uma validação estática que barre qualquer chamada manual malformada.
`BuildChain` (incluindo `Extra`, abaixo) é o caminho recomendado
justamente para não precisar disso na maioria dos casos.

Middlewares de grupo (`AllowContentType`, `MaxBodyBytes`, `NoCache`)
ficam de fora do `BuildChain` de propósito: são escopados a um grupo
específico (ex. só `/api`, não `/openapi.json`/`/docs`) por design,
não fazem sentido como parte da cadeia global. Não há ordem relevante
entre eles (são independentes), então não precisam de um builder
próprio — use `group.Use(...)` diretamente.

### Middleware customizada com requisito de ordem — `ChainConfig.Extra`

`BuildChain` só conhece os middlewares embutidos do `arnon` — se uma
middleware customizada ou de terceiros precisar rodar numa posição
específica relativa a um embutido (ex. "depois do `RateLimit`, antes
do `ETag`"), isso dá pra fazer de dois jeitos:

**1. Dividir a chamada.** Como `Router.Use(...)` acumula a cada
chamada (a ordem de chamada é preservada) e cada campo do
`ChainConfig` é independente dos outros, dá pra chamar `BuildChain`
duas vezes com subconjuntos complementares de campos, com a
middleware customizada entre elas:

```go
router.Use(middleware.BuildChain(middleware.ChainConfig{
    Recover: true, RealIP: true, RequestID: true, RateLimit: &cfg,
})...)
router.Use(xpto.Middleware()) // precisa vir depois do RateLimit, antes do ETag
router.Use(middleware.BuildChain(middleware.ChainConfig{
    ETag: true, Compress: &cCfg, CORS: &corsCfg, Logger: logger,
})...)
```

**2. `ChainConfig.Extra` — mesmo resultado, numa chamada só.** Cada
posição no `BuildChain` tem um `ChainAnchor` nomeado
(`AnchorRecover`, `AnchorTimeout`, `AnchorStripSlashes`,
`AnchorRedirectSlashes`, `AnchorRealIP`, `AnchorRequestID`,
`AnchorSecureHeaders`, `AnchorRateLimit`, `AnchorThrottle`,
`AnchorETag`, `AnchorCompress`, `AnchorCORS`, `AnchorServiceDesc`,
`AnchorLogging`, na mesma ordem da lista abaixo). Um
`ExtraMiddleware{Middleware: ..., Before: Anchor...}` ou `{...,
After: Anchor...}` insere a middleware customizada logo antes/depois
daquele ponto:

```go
router.Use(middleware.BuildChain(middleware.ChainConfig{
    Recover: true, RealIP: true, RequestID: true,
    RateLimit: &cfg,
    Extra: []middleware.ExtraMiddleware{
        {Middleware: xpto.Middleware(), After: middleware.AnchorRateLimit},
    },
    ETag: true, Compress: &cCfg, CORS: &corsCfg, Logger: logger,
})...)
```

Um `ChainAnchor` nomeia uma *posição*, não a presença de uma
middleware específica — `Extra` ancorado em `AnchorETag` continua
caindo no lugar certo mesmo que `ChainConfig.ETag` seja `false`
naquela chamada. `BuildChain` valida cada `ExtraMiddleware` e entra
em panic (na criação da cadeia, não no meio de uma request) se: nem
`Before` nem `After` forem setados, os dois forem setados ao mesmo
tempo, ou a âncora referenciada não for uma das constantes
`AnchorXxx` — esse último caso existe porque um typo no nome da
âncora, sem essa validação, simplesmente descartaria a middleware
customizada da cadeia em silêncio. Múltiplas entradas de `Extra`
ancoradas no mesmo ponto empilham na ordem em que aparecem no slice.

As duas formas produzem o mesmo resultado; `Extra` só evita ter que
dividir a chamada e decorar quais campos vão em cada metade. Ambas
continuam sendo, no fim, "onde no código a middleware é chamada" —
`Extra` não adiciona nenhuma verificação além de "essa âncora existe
e está bem formada", não valida se a middleware customizada em si é
segura para rodar naquela posição (isso continua sendo julgamento de
quem escreve, como em qualquer outra linguagem sem sistema de tipos
que modele "ordem de execução").

### A ordem em si, e por quê

Da mais externa (roda primeiro, envolve tudo) pra mais interna (roda
por último, mais perto do handler):

1. **`Recover`** — precisa envolver literalmente tudo abaixo pra
   pegar panic de qualquer middleware, não só do handler final.
   Trade-off aceito: por rodar antes de `RequestID`/`Logging`, não
   tem `request_id`/`trace_id` no log do panic, a menos que seja
   reposicionado pra depois desses dois (ver nota em `Recover`,
   abaixo).
2. **`Timeout`** — o prazo deve valer pra cadeia inteira abaixo, e o
   próprio `Timeout` relança (`panic`) o panic do handler pra fora,
   esperando um `Recover` mais externo pra capturar. Sem nenhum
   `Recover` na cadeia, esse repanic ainda assim não derruba o
   processo: a recuperação própria do `net/http` por conexão
   (`net/http.conn.serve`) captura, loga o stack trace no error log do
   servidor e fecha só aquela conexão — o resto das requisições em
   andamento não é afetado, mas o cliente vê a conexão cair em vez de
   uma resposta Problem Details, e o log não tem correlação de
   `request_id`/`trace_id`. Instalar `Recover` é o que transforma isso
   num 500 de verdade.
3. **`StripSlashes`/`RedirectSlashes`** (mutuamente exclusivos) —
   precisa normalizar o path antes de qualquer coisa que dependa
   dele, incluindo o próprio roteamento do `mux`.
4. **`RealIP`** — popula contexto que `RateLimit` (chave por IP) e
   `Logging` (`real_ip` no log) leem depois.
5. **`RequestID`** — popula contexto que `Logging` (`request_id` no
   log) lê depois.
6. **`SecureHeaders`** — barato, quer aparecer em toda resposta,
   incluindo erros gerados por qualquer middleware abaixo (um `429`
   do `RateLimit`, um `404` do `mux`).
7. **`RateLimit`**/**`Throttle`** — rejeitar cedo, antes de qualquer
   trabalho real (inclusive antes de `ETag`/`Compress` gastarem CPU
   numa resposta que nem vai ser aceita).
8. **`ETag`** — precisa vir antes de `Compress` pra hashear os bytes
   que de fato saem na rede (já comprimidos), não a versão anterior à
   compressão — consistente com o `Vary: Accept-Encoding` que o
   `Compress` seta.
9. **`Compress`**.
10. **`CORS`** — intercepta preflight (`OPTIONS` com
    `Access-Control-Request-Method`) antes do `mux`; um `OPTIONS` que
    não é preflight cai pro `mux`, então a posição aqui não bloqueia
    o `405`+`Allow` real discutido em `docs/architecture/rfc-compliance.md`.
11. **`ServiceDesc`** — só adiciona um header, sem dependência de
    posição forte; fica perto do fim por convenção.
12. **`Logging`** — mais interna do grupo acima de propósito: só
    monta os atributos do log (incluindo o que `RealIP`/`RequestID`
    populararam) uma vez, antes de chamar `next`, então precisa ser a
    última pra já ver tudo que as outras deixaram no contexto.

Nota sobre `Recover` + correlação: como ele é o mais externo (item 1),
ele *não* enxerga o `request_id`/`trace_id` que `RequestID`/`Logging`
(itens 5 e 12) só populam depois dele já ter rodado sua lógica de
pré-processamento. Quem precisar disso tem que abrir mão de
`BuildChain` pra essa parte específica e montar `Recover` manualmente
depois de `RequestID` — uma troca real (perde a garantia de capturar
panic de tudo, ganha correlação no log do panic), não uma
configuração que dê pra ter dos dois jeitos ao mesmo tempo.

**Manutenção**: toda middleware global nova precisa ganhar um campo
em `ChainConfig`, um `ChainAnchor` correspondente (adicionado em
`validChainAnchors` também) e uma chamada `appendStage(...)` na
posição certa dentro de `BuildChain` — senão ela fica inacessível via
`BuildChain`/`Extra` e essa seção de doc fica desatualizada. Middleware
de grupo (`AllowContentType`-like) não precisa disso.

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

## `WriteProblem` exige `*http.Request` para auto-popular `Problem.Instance`

`httpx.WriteProblem(writer, request, problemInstance)` recebe a
request desde 2026-07-15 (mudança de assinatura — aceitável porque o
framework ainda não teve release pública). Se `problemInstance.Instance`
estiver vazio, é preenchido com `request.URL.Path` antes de
serializar, nunca sobrescrevendo um valor setado via
`.WithInstance(...)`. `httpx` não pode depender de
`httpx/middleware`/`observability` (ver grafo de dependências acima),
então não dá pra usar `request_id`/`trace_id` aqui — path é o que dá
pra fazer sem alargar essa fronteira. Todo novo call site de
`WriteProblem` precisa passar a request.

## `httpx.Endpoint` é JSON-only por design; negociação de `Accept` formaliza isso

`Endpoint()` checa o header `Accept` (`httpx/accept.go`, `acceptsJSON`)
antes de fazer qualquer binding e responde `406 Not Acceptable`
(Problem Details) quando o cliente exclui explicitamente
`application/json` (ex. `Accept: application/xml` sozinho, ou
`application/json;q=0`). Um `Accept` ausente, vazio, ou que inclua
`application/json`/`application/*`/`*/*` com `q > 0` passa normal —
RFC 9110 §12.5.1 diz que header ausente significa "aceita qualquer
coisa". O parser segue a regra "match mais específico decide": uma
entrada exata bate antes de `application/*`, que bate antes de `*/*`.

Isso não é (nem deveria virar) negociação de múltiplas representações
do mesmo endpoint — `Endpoint()` continua só produzindo JSON, sempre.
Quem precisa servir XML, PDF, CSV ou qualquer outro
formato/arquivo monta um `http.Handler` comum via
`Router.GET`/`POST`/etc, exatamente como qualquer outra rota; nenhuma
middleware do framework (`Compress`, `ETag`, `SecureHeaders`, ...) é
acoplada a JSON. Não crie uma segunda abstração de endpoint tipado
"genérico em formato" pra cobrir esse caso — o padrão já é usar
`http.Handler` puro.

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

Exceção deliberada: tipos-valor pequenos e imutáveis, sem identidade
própria (ex. `openapi.Tag`, que é fluente e retorna novos valores a cada
`With*`; `problem.ValidationErrorCode`, um enum) usam receiver por
**valor** de propósito — não é inconsistência a corrigir. A heurística:
tipo com identidade/mutação/builder → ponteiro; tipo pequeno,
imutável, comportando-se como valor → valor.

---

## Stoplight

Foi escolhido Stoplight Elements ao invés de Swagger UI — encaixa melhor
com a proposta de "fundação" pouco opinativa (visual mais neutro,
apresentação em formato de doc/portal em vez de console de teste).

Ponto de atenção revisado em 2026-07-16: Swagger UI (a partir da
`swagger-ui-dist@5.32.0`, fev/2026) passou a ter suporte a OpenAPI 3.2.0;
o Stoplight Elements, até a mesma data, documenta suporte oficial só até
3.1. Como o `arnon` gera documentos `"openapi": "3.2.0"`
(`openapi.OpenAPIVersion3_2`), isso pode significar que o Stoplight
Elements não reconheça recursos novos da 3.2 (a maior parte das mudanças
de 3.2 sobre 3.1 é aditiva, então a renderização geral deve continuar
funcionando). Não verificado empiricamente num navegador real ainda —
antes de trocar o padrão ou expor a UI como configurável, vale essa
validação.

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
automatizado de `observability`/`observability/otel` (0% hoje), não a
funcionalidade em si.

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
  `httpx`, `httpx/binding`, `httpx/middleware` e `httpx/routing`.
* `httpx`: testes fim-a-fim via `httptest`, cobrindo binding, validação,
  mapeamento de erro (default e customizado) e o caminho de sucesso.

Planejado:

### OpenAPI (Golden Tests)

* Golden tests: gerar o documento OpenAPI pra um conjunto fixo de
  endpoints e comparar contra um arquivo de referência versionado no
  repo, de forma que qualquer drift não intencional na saída do
  gerador (um campo que sumiu, uma mudança de formato, ...) quebre o
  teste em vez de só ser percebido por alguém lendo um diff à mão.
  Ainda não implementado — os testes atuais de `openapi` fazem
  assertion em campos individuais do documento gerado, não num
  snapshot do documento completo.

### Unitários (pendente)

* cobertura de `observability`/`observability/otel` (0% hoje)

### Mutação

* `gremlins` já está integrado (`make test-mutation`, ver
  `CLAUDE.md`); falta uma rodada completa pelo código pra encontrar e
  tratar os mutantes sobreviventes.

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
* [docs/architecture/rfc-compliance.md](rfc-compliance.md): referência
  de conformidade com as RFCs relevantes pra uma fundação HTTP
  (RFC 9457, RFC 9110, RFC 9111, RFC 7239, RFC 6585,
  RFC 8288/8631/8615, RFC 8259), separando o que já é conforme, o que
  é uma decisão de escopo deliberada e o que é lacuna real — incluindo
  por que `Recover()` nunca deixa o detail de um panic recuperado
  chegar na resposta e por que `Timeout()` responde em Problem Details
  em vez de texto puro (os dois caminhos que, sem isso, quebrariam a
  própria garantia "RFC 9457 é o único formato de erro"), e por que o
  parsing de `Forwarded`/`X-Forwarded-For` só considera o primeiro
  hop.
