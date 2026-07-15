# arnon

Fundação HTTP minimalista para Go, com endpoint tipado, validação, geração
automática de OpenAPI 3.2 e erros no formato RFC 9457 (Problem Details).

Leia primeiro, nesta ordem:

1. [docs/architecture/project-context.md](docs/architecture/project-context.md) —
   especificação funcional/arquitetural e estado atual do projeto.
2. [docs/coding-style.md](docs/coding-style.md) — convenções de código
   (formatação, tratamento de erro/wrapcheck, dependências explícitas).
3. `docs/history/chatgpt-origin-transcript.md` — histórico de como o
   projeto surgiu; só vale consultar para entender *por que* uma decisão
   antiga foi tomada, não é referência de uso.

## Comandos

`make help` lista tudo. Os principais:

```sh
make test          # go test ./...
make test-race     # com detector de race conditions
make coverage      # gera coverage.html
make lint          # golangci-lint v2, versão fixa no Makefile
make arch-lint     # go-arch-lint check
make test-mutation # gremlins, grava mutation.json
make check         # lint + arch-lint + test-race (mínimo antes de commit)
make run           # go run ./examples/cmd/basic (outro: make run EXAMPLE=nome)
```

`golangci-lint` precisa da v2 (`.golangci.yml` usa `version: "2"`); a v1
falha ao carregar o config. O Makefile já usa `go run pkg@versão` fixa
para golangci-lint/go-arch-lint/gremlins, sem exigir instalação global.

`gremlins` (mutation testing) não lida com o padrão `./...` do Go —
silenciosamente reporta "No results to report" para múltiplos pacotes.
O Makefile já contorna isso passando `.` (ele recursa sozinho no módulo
inteiro); não troque para `./...` achando que é equivalente.

## Grafo de dependências entre pacotes

Documentado e verificado por `.go-arch-lint.yml`. Direção das setas =
"pode depender de":

```
problem  <-- validation <-- openapi
   ^             ^             ^
   |             |             |
httpx/binding    |             |
   ^             |             |
   +---------- httpx ----------+
                 ^
                 |
          httpx/routing  (única aresta "de baixo pra cima": routing
                           depende de httpx por causa da interface
                           httpx.OpenAPIProvider)
                 ^
                 |
          httpx/middleware

observability <-- observability/otel
```

Antes de adicionar um import entre pacotes internos, rode
`go-arch-lint check` — ele falha o build se a aresta não estiver
permitida em `.go-arch-lint.yml`.

## Decisões não óbvias

* **Middleware global (`Router.Use`) envolve o `mux` inteiro em
  `ServeHTTP`, não cada rota individualmente.** É o que faz middleware
  pré-roteamento (`StripSlashes`) funcionar e faz 404s passarem por
  `RequestID`/`Logging`/etc. Middleware de grupo (`Group.Use`) continua
  aplicado por-rota em `router.register`, já que `net/http.ServeMux`
  não tem noção de prefixo. Não volte a mesclar `router.middlewares`
  dentro de `register()` — duplicaria a execução.
* **RFC 9457 é o único formato de erro.** Todo erro HTTP vira
  `problem.Problem`. `httpx.WriteProblem`/`httpx.WriteJSON` sempre
  codificam a resposta num buffer antes de escrever qualquer header —
  não inverta essa ordem, é o que evita corromper a resposta quando a
  serialização falha.
* **`Endpoint()` sempre chama `EndpointConfig.WithDefaults()`.** Não
  reintroduza os `if config.Validator != nil` / `if config.ProblemMapper
  != nil` que existiam antes — depois de `WithDefaults()`, esses campos
  nunca são nil.
* **Registro no OpenAPI é opt-in por endpoint.** Uma rota só aparece no
  documento gerado se `EndpointConfig.OpenAPI` for preenchido (mesmo que
  com `&openapi.Operation{}` vazio).
* **`EndpointConfig.SuccessStatus` e `openapi.Operation.SuccessStatus`
  são campos independentes.** Nada sincroniza os dois hoje; ao mudar um,
  cheque o outro (ver `examples/cmd/basic/main.go`).
* **`examples/` segue o padrão `cmd/`+`internal/`.** Cada exemplo
  executável mora em `examples/cmd/<nome>` (`basic`: endpoint tipado +
  validação + OpenAPI, zero middleware; `middleware`: o mesmo endpoint
  com o stack completo de middlewares; `observability`: o mesmo
  endpoint com tracing/métricas via OpenTelemetry). Código
  compartilhado entre eles (logger, registro de custom validators, o
  handler de exemplo) mora em `examples/internal/*` — não importável
  de fora de `examples/` pela regra do Go, e por isso também precisou
  de `examples` na própria `mayDependOn` em `.go-arch-lint.yml` (senão
  o cross-import `cmd/* -> internal/*` é barrado mesmo os dois lados
  sendo o mesmo componente). Novo exemplo: crie `examples/cmd/<nome>`,
  reaproveite o que já existe em `examples/internal`, só duplique o
  que for específico daquele exemplo.
* **`examples/cmd/observability` precisa de shutdown gracioso pra
  fazer sentido.** É o único dos três exemplos que trata
  `SIGINT`/`SIGTERM` explicitamente (`signal.NotifyContext` +
  `server.Shutdown` + a função de shutdown que `otel.Initialize`
  devolve) — sem isso, spans e métricas ainda no buffer do SDK (o
  batch processor de trace, o periodic reader de métrica) se perdem
  quando o processo morre. `otel.Initialize` só devolve OTLP/gRPC como
  exporter (sem opção stdout), então o exemplo sobe um OTel Collector
  local via `docker compose` com exporter `debug` (imprime cada
  trace/métrica recebido no próprio log do collector) — validado de
  verdade rodando o collector, batendo o trace_id/span_id exportado
  contra o que a aplicação logou, e conferindo os exemplars das
  métricas customizadas apontando pro trace exato.
* **`middleware.Logging` só correlaciona trace_id/span_id se registrado
  depois do `routing.WithInstrumentation`.** `router.register` aplica
  `instrumentHandler` (que cria o span) por cima das middlewares de
  grupo/rota, mas só *dentro* do dispatch do mux — uma middleware
  *global* (`router.Use`) roda antes desse dispatch. `Logging`
  monta seus atributos (incluindo `observability.TraceID`/`SpanID`)
  antes de chamar `next`, então se ele for global, o contexto ainda não
  tem span nenhum. Por isso `examples/cmd/observability` registra
  `Logging` via `api.Use(...)` (grupo), não `router.Use(...)`.
* **Custom validators passam por um registry único
  (`validation.RegisterCustomRule`)**, não por configuração direta do
  `*validatorv10.Validate`. Um registro alimenta runtime, mapeamento de
  erro e geração de schema OpenAPI ao mesmo tempo — não adicione um
  novo mecanismo paralelo de registro de tags customizadas sem plugar
  nos três pontos (`validation/playground.go`, `validation/mapper.go`,
  `openapi/validation.go`).
* **`openapi/reflection_field.go` prioriza a tag `format` explícita**
  sobre o que `applyValidationTags` já inferiu, que por sua vez tem
  prioridade sobre o fallback de `inferFormatFromValidator`. Essa ordem
  é intencional (a mesma tag pode ser reconhecida nos dois lugares);
  não inverta.
* **Sem variáveis globais além de singletons deliberados**
  (`validation.Default()`, o registry de custom rules). Prefira
  injeção de dependência explícita para tudo o mais, conforme
  `docs/coding-style.md`.
* **Erros de binding não carregam status HTTP por padrão — mas podem
  ter override.** `httpx.Endpoint` (`writeValidationProblem`) fixa 400
  pra qualquer erro de `binding.Decode`, *exceto* quando algum
  `problem.ValidationError.Code` tem
  `StatusOverride() != 0` (`problem/validation_code.go`) — hoje só
  `ValidationCodePayloadTooLarge` → 413. É assim que `MaxBodyBytes`
  consegue devolver 413 mesmo quando o corpo estoura durante a leitura
  (chunked, tamanho desconhecido), sem mudar a assinatura de
  `binding.Decode`. Novo código que deveria implicar status diferente
  de 400: adicione o caso em `StatusOverride()`, não invente outro
  mecanismo paralelo.
* **`RateLimit` usa um sliding-window-counter (2 janelas), não um
  limiter por chave sem eviction.** Adaptado do `go-chi/httprate`,
  implementado do zero em `httpx/middleware/rate_limit.go` sem
  dependência externa (só `sync`/`time`). Janelas antigas são
  descartadas em bloco quando o tempo avança, então chaves inativas são
  removidas automaticamente em até duas janelas — não reintroduza a
  versão com `golang.org/x/time/rate` + `sync.Map` sem eviction que
  existiu brevemente aqui, tinha crescimento de memória sem limite.
* **`RateLimit` separa o algoritmo (janela deslizante) do storage via a
  interface `LimitCounter`.** Espelha de propósito a interface
  `LimitCounter` de `go-chi/httprate`, para que um backend já escrito
  pra httprate (ex. `go-chi/httprate-redis`) precise de mudanças
  triviais pra servir o arnon, e vice-versa. `RateLimitConfig.Counter`
  nil usa o default em memória (`NewLocalLimitCounter`, exportada —
  correto só pra instância única). Backends externos (Redis, Valkey,
  Memcached, ...) devem ser módulos Go separados, nunca dependência do
  módulo `arnon` em si — é por isso que só a interface + doc entraram
  aqui, sem nenhuma implementação de backend externo incluída. Erro do
  `Counter` vira `problem.Problem` via `RateLimitConfig.OnCounterError`
  (default: 503 Service Unavailable, configurável). Não reintroduza um
  segundo mecanismo de storage paralelo a `LimitCounter`.
