# Contribuindo

*[Read in English](CONTRIBUTING.md)*

Este projeto segue o [Código de Conduta](CODE_OF_CONDUCT.md). Ao
participar, espera-se que você o siga.

## Antes de começar

Leia, nesta ordem:

1. [CLAUDE.md](CLAUDE.md) — comandos, grafo de dependências entre
   pacotes, decisões de design não óbvias.
2. [docs/architecture/project-context.md](docs/architecture/project-context.md) —
   especificação funcional/arquitetural e estado atual do projeto.
3. [docs/coding-style.md](docs/coding-style.md) — convenções de código.
4. [docs/architecture/rfc-compliance.md](docs/architecture/rfc-compliance.md) —
   se a mudança tocar formato de erro, negociação de conteúdo, cache
   ou qualquer coisa RFC-adjacente, o comportamento já documentado ali
   é o que não pode regredir.

## Ambiente

`make help` lista todos os comandos. Antes de abrir um PR, rode:

```sh
make check   # lint + arch-lint + testes com race detector
```

Isso é o mínimo que o CI roda em todo PR
([.github/workflows/ci.yml](.github/workflows/ci.yml)). Se a mudança
tocar comportamento — não só refatoração — inclua teste cobrindo o
caso novo (unitário em `_test.go`, ou um caso em
`examples/cmd/*/requests.hurl` se for algo observável de ponta a ponta
via HTTP) e, quando fizer sentido, atualize
`docs/architecture/project-context.md`/`rfc-compliance.md`.

## Fluxo de branch

Rebase, não merge commit: atualize sua branch com `git rebase` contra
a base antes de abrir/atualizar um PR, em vez de mesclar a base pra
dentro da sua branch. Histórico linear, sem commits de merge.

A convenção de commit abaixo é obrigatória em `main`. Uma branch de
trabalho que vai passar por squash ao ser mesclada não precisa segui-la
à risca — o histórico intermediário não é o que fica.

## Mensagens de commit

[Conventional Commits](https://www.conventionalcommits.org/), em
inglês, modo imperativo:

```
<tipo>(<escopo>): <descrição>
```

* `<descrição>` no imperativo, descrevendo a ação (`add`, `fix`,
  `remove`, `harden`, `resolve`, `guarantee`), não o estado
  resultante (`added`) nem o histórico (`fixed`, "adds/added").
* `<escopo>` é o pacote ou área afetada. Não é uma lista fechada, mas
  os mais comuns batem com os componentes de
  [.go-arch-lint.yml](.go-arch-lint.yml): `problem`, `validation`,
  `openapi`, `httpx`, `binding`, `routing`, `middleware`,
  `observability`, `otel`, `examples` — mais os transversais `release`,
  `docs`, `ci`, `deps`.

### Tipos

| Tipo | Quando usar |
| --- | --- |
| `feat` | Novo recurso (MINOR do versionamento semântico). |
| `fix` | Correção de bug (PATCH do versionamento semântico). |
| `docs` | Só documentação (README, `docs/`, comentários) — sem mudança de código. |
| `test` | Só teste (criação, alteração ou remoção) — sem mudança de código de produção. |
| `refactor` | Muda a forma como o código é escrito/organizado sem mudar comportamento observável. |
| `perf` | Mudança cujo objetivo é performance. |
| `style` | Formatação, ponto e vírgula, espaço em branco, lint — sem mudança de código. |
| `build` | Build e dependências (`go.mod`, `Makefile`, ferramentas). |
| `ci` | Integração contínua (`.github/workflows/`). |
| `chore` | Tarefas de manutenção que não se encaixam nos tipos acima (config, `.gitignore`, ...). |
| `cleanup` | Remove código comentado, morto ou desnecessário — sem mudar comportamento. |
| `remove` | Remove arquivo, diretório ou funcionalidade obsoleta/não usada. |
| `raw` | Mudança em arquivo de configuração/dado/parâmetro que não se encaixa nos tipos acima. |

### Exemplos (do próprio histórico do projeto)

```
feat(validation): resolve RFC 6901 pointers through array/slice indices
fix(routing): complete splitPattern's method whitelist
docs(readme): explain why arnon exists before comparing frameworks
test(openapi): cover dive-redirected schema constraints
```

## Pull requests

* `make check` verde é obrigatório, não opcional.
* PR pequeno e focado em uma mudança é preferível a um PR grande
  cobrindo várias coisas não relacionadas — mas isso é julgamento, não
  regra rígida (ex.: uma correção de bug encontrada testando uma
  feature nova pode ir junto, se documentada claramente na descrição
  do PR/commit).
* Se a mudança altera comportamento documentado, atualize a doc no
  mesmo PR — não é aceitável um PR deixar `project-context.md`/
  `rfc-compliance.md` desatualizados de propósito "pra depois".
