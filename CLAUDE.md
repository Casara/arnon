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
make run           # go run ./examples/basic
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
  cheque o outro (ver `examples/basic/main.go`).
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
