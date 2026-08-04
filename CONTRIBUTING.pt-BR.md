# Contribuindo

*[Read in English](CONTRIBUTING.md)*

Este projeto segue o [Código de Conduta](CODE_OF_CONDUCT.md). Ao
participar, espera-se que você o siga.

## Antes de começar

Leia, nesta ordem:

1. [AGENTS.md](AGENTS.md) — comandos, grafo de dependências entre
   pacotes, decisões de design não óbvias.
2. [docs/architecture/project-context.pt-BR.md](docs/architecture/project-context.pt-BR.md) —
   especificação funcional/arquitetural e estado atual do projeto.
3. [docs/coding-style.pt-BR.md](docs/coding-style.pt-BR.md) — convenções de código.
4. [docs/architecture/rfc-compliance.pt-BR.md](docs/architecture/rfc-compliance.pt-BR.md) —
   se a mudança tocar formato de erro, negociação de conteúdo, cache
   ou qualquer coisa RFC-adjacente, o comportamento já documentado ali
   é o que não pode regredir.

## Ambiente

`make help` lista todos os comandos. Antes de abrir um PR, rode:

```sh
make check     # lint + arch-lint + verify-docs + testes com race detector
make lint-md   # lint de markdown (requer Node.js >= 20)
```

Isso é o mínimo que o CI roda em todo PR
([.github/workflows/ci.yml](.github/workflows/ci.yml)) — `check` e o
lint de markdown rodam como jobs separados. Se a mudança tocar
comportamento — não só refatoração — inclua teste cobrindo o
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

```text
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

```text
feat(validation): resolve RFC 6901 pointers through array/slice indices
fix(routing): complete splitPattern's method whitelist
docs(readme): explain why arnon exists before comparing frameworks
test(openapi): cover dive-redirected schema constraints
```

## Mudando a API pública

O projeto está em **v0.x**, então uma quebra é permitida — ver
[Estabilidade da API](README.pt-BR.md#estabilidade-da-api) para o que isso
significa para quem usa. Mesmo assim ela precisa ser deliberada e visível:

* Diga isso na descrição do PR, e adicione uma entrada `### Changed` ou
  `### Removed` no `CHANGELOG.md`, com a migração na mesma entrada.
* Onde um rename puder manter a grafia antiga compilando, mantenha como alias
  depreciado em vez de apagar de uma vez.
* Prefira mudança aditiva quando existir uma: campo novo numa struct de
  config, opção variádica, construtor novo ao lado do antigo.
* Tudo na lista de sincronização de docs do [AGENTS.md](AGENTS.md) se aplica.
  Uma função `Example` é a forma mais barata de provar que o formato novo
  funciona — o `make check` roda elas.

Mensagens de erro e de log explicitamente **não** fazem parte da API pública;
casar com o texto delas não é suportado.

## Como uma mudança é feita

O histórico deste repositório segue um ciclo, e vale dizê-lo porque ele não é
visível pelo código: **explorar, planejar, implementar, commitar.** Ler o
`project-context.md` e o `CLAUDE.md` do pacote antes de tocar em qualquer coisa
é o passo "explorar", e é onde a maior parte do custo é evitada — vários
invariantes daqui existem porque uma tentativa anterior errou neles.

Dois hábitos importam mais que o ciclo em si:

* **Entregue a funcionalidade, seu exemplo e sua documentação juntos.** O
  histórico mostra isso em trios: a mudança no pacote, depois `examples/cmd/*`,
  depois as docs. O PR que deixa o terceiro pra depois é o que fica obsoleto.
* **Prefira um `Example` a prosa.** O `make check` compila e roda todo
  `Example`, comparando o bloco `// Output:`, então é a única documentação que
  a toolchain consegue manter honesta. Escrever os deste repositório revelou
  três erros que estavam nas docs, compilando normalmente.

## Revisão

O projeto tem um mantenedor só, então "revisão" não é um portão que uma segunda
pessoa abre — é o que as checagens e o diff precisam deixar óbvio sozinhos.

* **O CI precisa estar verde antes do merge.** `check` (nos três módulos),
  `govulncheck` e `markdown lint`. Run vermelho não é mergeado e consertado
  depois.
* **O mantenedor faz o merge**, com squash, pra `main` manter um commit por
  mudança. É por isso que commit intermediário de branch de trabalho não
  precisa seguir a convenção à risca, enquanto a mensagem do squash precisa.
* **Mudança em símbolo exportado é lida contra a lista de sincronização de
  docs** do [AGENTS.md](AGENTS.md), não só contra os testes. O `make doc-sync`
  imprime o que um diff que mexeu na superfície pública deixou pra trás.

## Contribuindo com assistente de IA

Este projeto é construído com um, e não há o que esconder ou declarar sobre
isso: o código é julgado igual de qualquer forma, e um PR não é marcado.

O que se pede é o mesmo que se pede a qualquer um:

* **Entenda o que você está submetendo.** Se você não consegue explicar por que
  a mudança está certa, ela não está pronta — independente do que a escreveu.
* **Não deixe a ferramenta rediscutir decisão já tomada.** O `AGENTS.md` e os
  `CLAUDE.md` por pacote existem porque várias dessas decisões foram tomadas,
  revertidas e tomadas de novo. PR que reintroduz uma é fechado com um
  ponteiro, não com debate.
* **Rode as checagens localmente.** `make check` e `make lint-md`, antes de
  abrir o PR e não depois do CI reclamar.

O `AGENTS.md` é lido direto por Codex e Cursor; o `CLAUDE.md` o importa pro
Claude Code. Qualquer coisa que você adicione pra um deles pertence ao
`AGENTS.md`, pros outros receberem também.

## Pull requests

* `make check` verde é obrigatório, não opcional. Se a mudança tocar
  algum arquivo Markdown, `make lint-md` também.
* PR pequeno e focado em uma mudança é preferível a um PR grande
  cobrindo várias coisas não relacionadas — mas isso é julgamento, não
  regra rígida (ex.: uma correção de bug encontrada testando uma
  feature nova pode ir junto, se documentada claramente na descrição
  do PR/commit).
* Se a mudança altera comportamento documentado, atualize a doc no
  mesmo PR — não é aceitável um PR deixar `project-context.md`/
  `rfc-compliance.md` desatualizados de propósito "pra depois".
