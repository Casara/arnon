# Go Style Guide

*[Leia em português](coding-style.pt-BR.md)*

## Purpose

This document defines additional conventions adopted by the project beyond the rules already enforced automatically
by tools such as `gofmt`, `goimports`, and `golangci-lint`.

Whenever possible, decisions should prioritize:

* readability;
* consistency;
* maintainability;
* diff quality.

---

## Formatting

All code must be formatted with:

* gofmt
* goimports

Manual changes that go against the formatting produced by these tools must not be made.

---

## Readability over conciseness

Prefer explicit, easy-to-understand code over excessively compact versions.

Preferred:

```go
if err != nil {
    return err
}
```

Avoid:

```go
if err != nil { return err }
```

---

## Function calls

Simple calls should stay on a single line.

```go
logger := slog.Default()

responseRecorder := newResponseWriter(writer)
```

Calls with multiple arguments, options, or nested structures should use the vertical format.

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

## Diff quality

Whenever a construct has a natural tendency to grow, prefer the vertical format.

Example:

```go
attrs := []any{
    slog.String("method", request.Method),
    slog.String("path", request.URL.Path),
}
```

This format reduces merge conflicts and produces smaller diffs when new elements are added.

---

## Comments

Comments should explain:

* purpose;
* behavior;
* limitations;
* design decisions.

Comments that merely repeat the function name should be avoided.

Bad:

```go
// NewLogger creates a logger.
```

Better:

```go
// NewLogger creates the application logger.
//
// The returned logger writes structured JSON logs to stdout and is
// intended to be shared across the entire application.
```

---

## Error Handling (wrapcheck)

Errors returned by external dependencies or lower layers should receive additional context before being
propagated.

The goal is to make the origin of the failure evident in the logs and ease diagnosis in production.

### Rule

When returning an error received from another function, add context using `fmt.Errorf` and `%w`.

Correct:

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

Avoid:

```go
return err
```

---

### Error message

The message should describe the operation that failed, not repeat the original error's text.

Correct:

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

Avoid:

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

### Level of detail

Add only the new context introduced by the current layer.

Example:

```go
create counter "categories_created_total":
invalid instrument name
```

and afterward:

```go
initialize metrics:
create counter "categories_created_total":
invalid instrument name
```

Each layer adds relevant information without repeating context that's already present.

---

### When wrapping is not necessary

Do not wrap when:

* creating a new error;
* returning a sentinel error;
* the error already contains sufficient context and the current layer adds no relevant information.

Examples:

```go
return ErrNotFound
```

```go
return errors.New(
    "invalid category name",
)
```

---

### Guidance for AIs

When fixing `wrapcheck` violations:

1. Preserve the error chain using `%w`.
2. Describe the operation that failed.
3. Do not use generic messages such as:

   * "error"
   * "failed"
   * "unexpected error"
4. Do not repeat context already present in lower layers.
5. Prefer short, lowercase messages.
6. Include relevant identifiers when they add value:

```go
return fmt.Errorf(
    "create counter %q: %w",
    name,
    err,
)
```

---

## Dependencies

Prefer explicit dependencies via dependency injection over global variables.

Preferred:

```go
type CategoryHandler struct {
    metrics *Metrics
}
```

Avoid:

```go
var categoriesCreatedCounter ...
```

Unless the nature of the component clearly justifies a shared singleton.
