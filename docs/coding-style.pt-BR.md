# Guia de Estilo Go

*[Read in English](coding-style.md)*

## Objetivo

Este documento define convenções adicionais adotadas pelo projeto além das regras já aplicadas automaticamente por
ferramentas como `gofmt`, `goimports` e `golangci-lint`.

Sempre que possível, as decisões devem priorizar:

* legibilidade;
* consistência;
* facilidade de manutenção;
* qualidade dos diffs.

---

## Formatação

Todo código deve ser formatado com:

* gofmt
* goimports

Não devem ser realizadas alterações manuais para contrariar a formatação produzida por essas ferramentas.

---

## Legibilidade acima da concisão

Prefira código explícito e fácil de entender em vez de versões excessivamente compactas.

Preferível:

```go
if err != nil {
    return err
}
```

Evite:

```go
if err != nil { return err }
```

---

## Chamadas de função

Chamadas simples devem permanecer em uma única linha.

```go
logger := slog.Default()

responseRecorder := newResponseWriter(writer)
```

Chamadas com múltiplos argumentos, opções ou estruturas aninhadas devem utilizar o formato vertical.

```go
requestLogger.Info(
    "http request",
    slog.Int("status_code", statusCode),
    slog.Duration("duration", duration),
)
```

```go
return otelhttp.NewHandler(
    traceHandler,
    spanName,
    otelhttp.WithTracerProvider(
        otel.GetTracerProvider(),
    ),
)
```

---

## Qualidade dos diffs

Sempre que uma construção possuir tendência natural de crescimento, prefira o formato vertical.

Exemplo:

```go
attrs := []any{
    slog.String("method", request.Method),
    slog.String("path", request.URL.Path),
}
```

Esse formato reduz conflitos de merge e produz diffs menores quando novos elementos são adicionados.

---

## Comentários

Comentários devem explicar:

* propósito;
* comportamento;
* limitações;
* decisões de projeto.

Comentários que apenas repetem o nome da função devem ser evitados.

Ruim:

```go
// NewLogger creates a logger.
```

Melhor:

```go
// NewLogger creates the application logger.
//
// The returned logger writes structured JSON logs to stdout and is
// intended to be shared across the entire application.
```

---

## Tratamento de Erros (wrapcheck)

Erros retornados por dependências externas ou por camadas inferiores devem receber contexto adicional antes de
serem propagados.

O objetivo é tornar a origem da falha evidente nos logs e facilitar o diagnóstico em produção.

### Regra

Ao retornar um erro recebido de outra função, adicione contexto usando `fmt.Errorf` e `%w`.

Correto:

```go
return fmt.Errorf(
    "create trace exporter: %w",
    err,
)
```

```go
return fmt.Errorf(
    "start runtime metrics: %w",
    err,
)
```

Evite:

```go
return err
```

---

### Mensagem de erro

A mensagem deve descrever a operação que falhou, não repetir o texto do erro original.

Correto:

```go
return fmt.Errorf(
    "load configuration: %w",
    err,
)
```

```go
return fmt.Errorf(
    "create HTTP server: %w",
    err,
)
```

Evite:

```go
return fmt.Errorf(
    "error: %w",
    err,
)
```

```go
return fmt.Errorf(
    "failed: %w",
    err,
)
```

```go
return fmt.Errorf(
    "unexpected error: %w",
    err,
)
```

---

### Nível de detalhe

Adicione apenas o contexto novo introduzido pela camada atual.

Exemplo:

```go
create counter "categories_created_total":
invalid instrument name
```

e depois:

```go
initialize metrics:
create counter "categories_created_total":
invalid instrument name
```

Cada camada adiciona informação relevante sem repetir contexto já presente.

---

### Quando o wrap não é necessário

Não faça wrap quando:

* estiver criando um erro novo;
* estiver retornando um erro sentinela;
* o erro já contém contexto suficiente e a camada atual não adiciona informação relevante.

Exemplos:

```go
return ErrNotFound
```

```go
return errors.New(
    "invalid category name",
)
```

---

### Orientação para IAs

Ao corrigir violações de `wrapcheck`:

1. Preserve a cadeia de erro usando `%w`.
2. Descreva a operação que falhou.
3. Não utilize mensagens genéricas como:

   * "error"
   * "failed"
   * "unexpected error"
4. Não repita contexto já presente em camadas inferiores.
5. Prefira mensagens curtas em minúsculas.
6. Inclua identificadores relevantes quando agregarem valor:

```go
return fmt.Errorf(
    "create counter %q: %w",
    name,
    err,
)
```

---

## Dependências

Prefira dependências explícitas por injeção de dependência em vez de variáveis globais.

Preferível:

```go
type CategoryHandler struct {
    metrics *Metrics
}
```

Evite:

```go
var categoriesCreatedCounter ...
```

Salvo quando a natureza do componente justificar claramente um singleton compartilhado.
