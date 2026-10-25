# Harness engineering na marIA

Este documento explica o que é harness engineering, por que ele existe, como
está montado neste repositório e como trabalhar com ele no dia a dia. Ele é
escrito para o administrador (humano), mas os agentes também o leem.

Sumário:

1. [O que é harness engineering](#1-o-que-é-harness-engineering)
2. [Por que um agente precisa de harness](#2-por-que-um-agente-precisa-de-harness)
3. [Princípios](#3-princípios)
4. [Anatomia do harness da marIA](#4-anatomia-do-harness-da-maria)
5. [Quando usar cada peça](#5-quando-usar-cada-peça)
6. [De onde vêm os agents e skills](#6-de-onde-vêm-os-agents-e-skills)
7. [O ciclo de trabalho](#7-o-ciclo-de-trabalho)
8. [Exemplo de ponta a ponta](#8-exemplo-de-ponta-a-ponta)
9. [A escada de enforcement](#9-a-escada-de-enforcement)
10. [Ciclo de vida de uma feature](#10-ciclo-de-vida-de-uma-feature)
11. [Manutenção do harness](#11-manutenção-do-harness)
12. [Anti-padrões](#12-anti-padrões)
13. [Como saber se o harness está funcionando](#13-como-saber-se-o-harness-está-funcionando)
14. [Glossário](#14-glossário)

---

## 1. O que é harness engineering

Um agente de código como o Claude Code é **modelo + harness**.

- O **modelo** é o motor: lê, raciocina e escreve código.
- O **harness** (arreio, em inglês) é tudo em volta que direciona esse motor:
  as instruções que ele recebe, os arquivos que consegue ler, as ferramentas
  que pode usar, as ações que estão proibidas, os testes que dizem se ele
  acertou e a memória que sobrevive de uma sessão para outra.

**Harness engineering** é projetar esse entorno de forma deliberada, como se
projeta o código. O termo ganhou força em 2025 e 2026, quando equipes que
passaram a escrever a maior parte do código com agentes perceberam que o
gargalo deixou de ser "o modelo sabe programar?" e passou a ser "o ambiente
deixa o modelo acertar sozinho, de forma repetível?".

A mudança de papel é esta:

| Sem harness | Com harness |
| --- | --- |
| O humano escreve prompts longos a cada sessão | O contexto mora no repositório e é carregado sozinho |
| O humano revisa cada linha | O humano revisa o plano e o resultado; regras mecânicas pegam o resto |
| O erro é corrigido no código e volta a acontecer | O erro vira uma regra, um teste ou um hook e não se repete |
| Cada sessão começa do zero | Cada sessão começa de onde a anterior parou |

Em resumo: **o humano decide o quê e o porquê, o agente executa o como, e o
harness garante que o como respeite o quê.**

## 2. Por que um agente precisa de harness

Agentes falham de formas previsíveis. O harness existe para neutralizar cada
uma delas:

| Falha típica do agente | O que acontece | Peça do harness que resolve |
| --- | --- | --- |
| **Amnésia entre sessões** | A janela de contexto acaba ou a sessão fecha; a próxima não sabe o que foi feito | `docs/progress.md`, `docs/features.json`, hook `SessionStart` |
| **Declarar vitória cedo** | "Pronto!" sem ter rodado testes ou com o critério de aceite pela metade | `features.json` com status que exige evidência, `make check`, E2E em staging |
| **Deriva de arquitetura** | Um atalho "inofensivo": o conversation lê a tabela do reminders | `scripts/archcheck.sh`, `architecture-guardian` |
| **Ações perigosas** | `terraform apply` da máquina local, push na `main` | Hooks em `.claude/hooks/`, proteção de branch |
| **Excesso de contexto** | Um CLAUDE.md gigante; o agente se perde ou ignora regras | CLAUDE.md curto como índice, detalhes em `docs/` e skills |
| **Reinventar a roda** | Cada serviço novo com uma estrutura diferente | Skills `/new-service`, `/new-event`, `/bedrock-tool` |
| **Autoavaliação enviesada** | Quem escreveu o código revisa o próprio código | Subagentes revisores com contexto separado |
| **Entropia** | Docs envelhecem, planos ficam abandonados, a spec diverge do código | `/garden`, `/handoff` |

## 3. Princípios

### 3.1 O repositório é a fonte da verdade
O que não está no repositório não existe para o agente. Uma decisão tomada
numa conversa de WhatsApp ou na sua cabeça precisa virar texto na spec, num
ADR ou num plano. Por isso tudo aqui é versionado: spec, planos, progresso,
decisões, regras e até as instruções dos agentes.

### 3.2 O CLAUDE.md é um mapa, não uma enciclopédia
O CLAUDE.md é carregado em **toda** sessão e ocupa contexto o tempo todo.
Ele deve ter só o que vale sempre: regras invioláveis e ponteiros. O detalhe
fica em arquivos que o agente abre quando precisa (a spec, um plano, uma
skill). Isso se chama *revelação progressiva*: o agente recebe o mínimo e
sabe onde buscar o resto. Meta: CLAUDE.md abaixo de ~90 linhas.

### 3.3 Prefira regra mecânica a regra escrita
"Nunca logar telefone em claro" escrito no CLAUDE.md é um pedido. Um script
que falha o build quando encontra `"phone"` num log sem `Mask()` é uma
garantia. Toda regra importante deve migrar de texto para verificação
automática (veja a [escada de enforcement](#9-a-escada-de-enforcement)).

### 3.4 Feedback rápido e que ensina
O agente aprende com a mensagem de erro. Por isso as mensagens do harness
dizem **o que** está errado e **o que fazer**:
`internal/conversation must not import internal/reminders (use pkg/events or the reminders API)`.
Um erro genérico ("falhou") faz o agente tentar às cegas.

### 3.5 Estado fora da janela de contexto
A janela de contexto é memória de curto prazo e vai ser resumida ou perdida.
O estado que importa (o que está pronto, o que falta, o que foi decidido)
mora em arquivos: `features.json`, `progress.md`, planos.

### 3.6 Uma coisa de cada vez
Agentes erram mais quando tentam fazer tudo de uma vez. Uma feature por vez,
uma branch por fatia, passos que terminam com `make check` verde.

### 3.7 Corrija o harness, não só o resultado
Quando o agente errar, pergunte: "que regra, doc, teste ou hook teria evitado
isso?". Corrigir só o código resolve hoje; corrigir o harness resolve para
sempre. Os planos e o template de PR têm a seção "Lições para o harness" para
registrar isso.

### 3.8 O humano dirige
O agente propõe, o humano aprova o plano, o agente executa, o harness
verifica, o humano aprova o merge. Decisões de produto, custo e segurança
sempre voltam para o humano (a seção "Decisões em aberto" dos planos existe
para isso).

## 4. Anatomia do harness da marIA

```
CLAUDE.md                         camada 1: instruções sempre carregadas
infra/CLAUDE.md                   camada 1: regras de Terraform (carregadas ao trabalhar em infra/)
.github/CLAUDE.md                 camada 1: regras dos workflows
e2e/CLAUDE.md                     camada 1: regras da suíte de staging
HARNESS.md                        este guia
docs/
  spec/                           camada 2: fonte da verdade do produto (um arquivo por área)
    README.md                     índice, visão geral, glossário do domínio e roteiro
  features.json                   camada 3: estado do roteiro
  progress.md                     camada 3: memória entre sessões (fase atual)
  progress/fase-<n>.md            camada 3: histórico de fases concluídas
  plans/active/ completed/        camada 3: planos de execução
  plans/TEMPLATE.md
  adr/                            camada 2: decisões e seus porquês
  runbooks/                       camada 2: operação
.claude/
  settings.json                   camada 4: permissões e registro dos hooks
  hooks/                          camada 4: guarda-corpos automáticos
  agents/                         camada 5: especialistas (subagentes)
  skills/                         camada 5: receitas
Makefile, .golangci.yml           camada 6: verificação local
scripts/archcheck.sh              camada 6: regras de arquitetura
scripts/test-harness.sh           camada 6: testes do próprio harness (hooks e archcheck)
scripts/testdata/                 camada 6: fixtures do archcheck (violações e caso limpo)
.github/workflows/ (Fase 1)       camada 7: esteira e portões finais
.github/pull_request_template.md  camada 7: checklist de PR
```

### Camada 1: instruções (`CLAUDE.md`)
- **O que é:** um arquivo que o Claude Code injeta no início de toda sessão.
- **Aqui:** stack, estrutura, regras de arquitetura, de produto e de segurança, a esteira e a seção "Harness" com os ponteiros.
- **Cuidado:** cada linha custa contexto em toda sessão. Se uma regra só vale para uma tarefa (por exemplo, como criar uma ferramenta do Bedrock), ela vai para uma skill.
- **CLAUDE.md por pasta:** o Claude Code também carrega o `CLAUDE.md` de uma subpasta quando trabalha em arquivos dela. As regras de Terraform ficam em `infra/CLAUDE.md`, as dos workflows em `.github/CLAUDE.md` e as do E2E em `e2e/CLAUDE.md`. Assim, a raiz guarda só o que vale sempre e não cresce a cada área nova. Regra para pastas novas: se as regras de uma pasta só importam para quem trabalha nela, elas vão para um CLAUDE.md da pasta.

### Camada 2: conhecimento (`docs/spec/`, `docs/adr/`, `docs/runbooks/`)
- **Spec:** descreve o que a marIA faz, dividida por área (produto, arquitetura, WhatsApp, Bedrock, infra, esteira), com índice, visão geral, glossário e roteiro no `README.md`. O agente abre só o arquivo de que a tarefa precisa. Mudou o comportamento, muda a spec primeiro (ou no mesmo PR).
- **ADRs:** registram decisões difíceis de reverter e o porquê. Evitam que o agente (ou você daqui a 3 meses) reabra uma discussão já resolvida.
- **Runbooks:** passo a passo de operação. Também deixam claro o que o agente **não** faz (bootstrap do state, cadastro de templates na Meta).

### Camada 3: estado e memória
- **`docs/features.json`:** as 36 features do roteiro com critérios de aceite, status e evidência. Fica em JSON e não em markdown porque agentes tendem a reescrever, resumir ou "arrumar" listas em markdown; um arquivo estruturado, com instrução explícita de não mexer nos critérios, é mais estável.
- **`docs/progress.md`:** diário curto, mais recente em cima. Cada entrada responde: o que foi feito, como foi verificado, o que ficou pendente e qual o próximo passo.
- **`docs/plans/`:** um plano por fatia de trabalho, com passos, decisões em aberto, verificação, riscos, registro de execução e lições. O plano é revisado por você **antes** do código e marcado no mesmo PR que implementa cada passo.

### Camada 4: guarda-corpos (`.claude/settings.json` e `.claude/hooks/`)
Hooks são scripts que o próprio Claude Code executa em momentos do ciclo do
agente. Não dependem de o modelo "lembrar" de nada.

| Hook | Momento | O que faz |
| --- | --- | --- |
| `session-context.sh` | `SessionStart` | Mostra branch, aviso de back-merge pendente, planos ativos, contagem de features e último progresso |
| `guard-bash.sh` | `PreToolUse` em Bash | Bloqueia `terraform apply/destroy/import/state`, `docker push`, AWS CLI que altera recursos, escrita no S3, leitura de segredos, commit/merge estando na `main` ou na `develop`, push fora de `feature/*`, `release/x.y.z` e `hotfix/x.y.z`, force push, merge de PR pelo agente |
| `guard-files.sh` | `PreToolUse` em Read/Edit/Write | Bloqueia `.env*`, `*.tfstate`, `*.pem`, `*.key` |
| `format.sh` | `PostToolUse` em Edit/Write | Roda `goimports`/`gofmt` em `.go` e `terraform fmt` em `.tf` |

Um hook que sai com código 2 bloqueia a ação, e a mensagem dele volta para o
agente, que ajusta o plano. O `settings.json` também tem `permissions`: uma
lista do que roda sem pedir permissão (testes, lint, `git diff`) e do que é
sempre negado.

### Camada 5: especialistas e receitas (`.claude/agents/`, `.claude/skills/`)
- **Subagentes** são instâncias separadas do agente, cada uma com a própria janela de contexto, o próprio prompt e uma lista restrita de ferramentas. O agente principal delega a tarefa e recebe só o resultado. Vantagens:
  - **Revisão sem viés:** o revisor não "viu" o raciocínio de quem escreveu.
  - **Economia de contexto:** a varredura do revisor não polui a sessão principal.
  - **Menor privilégio:** a maioria dos revisores só lê e não edita.
- **Skills** são receitas em markdown. Só o nome e a descrição ficam no contexto; o corpo é carregado quando a skill é chamada (por você com `/nome`, ou pelo agente quando a descrição combina com a tarefa).

### Camada 6: verificação local
- `make check`: `gofmt -l`, `go vet`, `golangci-lint`, `scripts/archcheck.sh` e `go test ./... -race`. Precisa estar verde antes de qualquer PR.
- `make harness-test`: testa o próprio harness. Roda os hooks contra dezenas de comandos num repositório descartável (o que deve passar e o que deve ser bloqueado) e roda o archcheck contra as fixtures de `scripts/testdata/`. Cada linha marcada com `want:` precisa ser reportada, e o caso limpo precisa passar. Faz parte do `make check` e, na Fase 1, do `ci`. **O harness também é código:** mudou um hook ou uma regra, o teste muda no mesmo PR.
- `make tf-check`: `terraform fmt -check` e `terraform validate` sem backend.
- `.golangci.yml`:
  - proíbe o pacote `log` (use `slog` via `pkg/logging`), o SDK da AWS v1 e `fmt.Print*`;
  - restringe `os.Getenv` aos arquivos `config.go`;
  - liga `gosec`, `errorlint`, `noctx` e outros.
- `scripts/archcheck.sh`: as regras de arquitetura do CLAUDE.md transformadas em checagens. Um serviço não importa outro, só o outbound envia à Meta, cada serviço só toca a própria tabela, `graph.facebook.com` só em `pkg/meta`, relógio injetado em vez de `time.Now()`, logs de texto e telefone sempre com `Mask()`. Para uma exceção justificada, use o comentário `archcheck:allow <motivo>` na linha.

### Camada 7: esteira (Fase 1)
O modelo de branches é o Git Flow padrão (ADR 0002 e `docs/spec/esteira.md`):

```
main      ●────────────────────●──────────────●────────▶  prod (tags v0.1.0, v0.1.1)
           \                  / \            /  \
hotfix      \                /   ●──────────●    \         hotfix/0.1.1 → staging → main
             \              /                \    \
release       \      ●─────●                  \    \       release/0.1.0 → staging → main
               \    /       \ back-merge       \    \ back-merge
develop  ───────●──●─────────●──────────────────●────●──▶  staging
                 \  /
feature           ●●                                        feature/* → PR → develop
```

| Evento | Workflow | Resultado |
| --- | --- | --- |
| PR para `develop` ou `main` | `ci` | Lint, testes (portão), build, `terraform plan`; para a `main`, valida origem e `staging-ok` |
| Push na `develop`, `release/*` ou `hotfix/*` | `deploy-staging` | Deploy e E2E em staging; status `staging-ok` no commit |
| Push na `main` (merge da release ou hotfix) | `deploy-prod` | Aprovação no environment `production`, mesma imagem em prod, tag `vx.y.z` |
| Depois de prod | você | PR de back-merge `main` → `develop` |

- **Proteção de branch:** ninguém dá push em `develop` ou `main`; tudo entra por PR, e o merge é seu.
- **Back-merge:** é o erro clássico do Git Flow. O hook de início de sessão avisa quando a `main` tem commits que a `develop` não tem.
- **Por que é a última camada:** é a única que nem você nem o agente conseguem contornar sem querer.

## 5. Quando usar cada peça

| Você quer... | Use | Por quê |
| --- | --- | --- |
| Uma regra que vale em toda sessão | `CLAUDE.md` | Sempre carregado |
| Um procedimento para uma tarefa específica | Skill | Só carrega quando precisa |
| Uma segunda opinião ou uma varredura grande | Subagente | Contexto isolado, sem viés do autor |
| Impedir uma ação, sem exceção | Hook | Executado pelo Claude Code, não pelo modelo |
| Garantir uma propriedade do código | Lint, archcheck ou teste | Falha o build |
| Registrar o porquê de uma decisão | ADR | Evita reabrir a discussão |
| Saber onde paramos | `progress.md` e `features.json` | Sobrevivem ao fim da sessão |
| Organizar um trabalho de vários passos | Plano em `docs/plans/active/` | Revisável antes do código |

Regra prática: se você explicou a mesma coisa ao agente duas vezes, ela
deveria estar numa dessas peças.

## 6. De onde vêm os agents e skills

Há três origens diferentes:

**1. Criados para este projeto (estão neste repositório).** Não são produtos
de mercado: são arquivos markdown escritos para a marIA, com as regras da
spec embutidas. Você pode e deve editá-los quando o projeto mudar.

| Tipo | Nome | Arquivo |
| --- | --- | --- |
| Agent | `architecture-guardian` | `.claude/agents/architecture-guardian.md` |
| Agent | `persona-reviewer` | `.claude/agents/persona-reviewer.md` |
| Skill | `/plan-phase` | `.claude/skills/plan-phase/SKILL.md` |
| Skill | `/new-service` | `.claude/skills/new-service/SKILL.md` |
| Skill | `/new-event` | `.claude/skills/new-event/SKILL.md` |
| Skill | `/bedrock-tool` | `.claude/skills/bedrock-tool/SKILL.md` |
| Skill | `/handoff` | `.claude/skills/handoff/SKILL.md` |
| Skill | `/garden` | `.claude/skills/garden/SKILL.md` |

**Critério para criar um agent próprio:** só quando o conhecimento dele é
específico da marIA e um agente genérico aprovaria uma violação. Um revisor
genérico de Go não sabe que o conversation não pode ler a tabela do reminders,
e um revisor genérico de texto não sabe que a marIA nunca pede confirmação.
Por isso existem só esses dois. Testes, Terraform e segurança são cobertos
pelas regras das seções "Testes" e "Infra" do CLAUDE.md e pelos revisores
embutidos (`/code-review`, `/security-review`). Uma tarefa periódica, como a
jardinagem dos docs, é uma skill (`/garden`), não um especialista.

Antes de criar um agent ou uma skill nova, pergunte:
- Já existe algo embutido ou de mercado que resolve?
- O que faltaria nele cabe numa linha do CLAUDE.md?

Só crie se as duas respostas forem "não".

Um agent é um arquivo com frontmatter (`name`, `description`, `tools`,
`model`) e o prompt do especialista. A `description` é o que o agente
principal usa para decidir quando delegar, então ela diz **quando** usar.
Uma skill é uma pasta com `SKILL.md`, frontmatter (`name`, `description`) e
o passo a passo; pode ter scripts e arquivos de apoio na mesma pasta.

**2. Embutidos no Claude Code (vêm prontos, de qualquer projeto).**
- `/code-review`: revisão do diff.
- `/security-review`: revisão de segurança.
- `/simplify`: limpeza de código.
- `claude-api`: referência da API do Claude, útil para o Converse e o tool use.
- `/schedule`: agentes recorrentes na nuvem, por exemplo o `/garden` semanal.
- `/loop`: repetir um comando, por exemplo acompanhar uma run da esteira.
- `/fewer-permission-prompts`: reduz pedidos de permissão.
- `skill-creator`: ajuda a criar e testar skills novas.
- Os subagentes genéricos `Explore`, `Plan` e `general-purpose`.

**3. Externos (plugins e servidores MCP que você instala).** Exemplos úteis
para a marIA: o servidor MCP do Terraform da HashiCorp (documentação de
providers e módulos) e o servidor MCP de documentação da AWS. Eles são de
terceiros: avalie o que cada um pode acessar antes de instalar, e prefira os
que só leem.

## 7. O ciclo de trabalho

```
 ┌────────────┐   ┌───────────┐   ┌──────────┐   ┌───────────┐   ┌──────────┐
 │ 1. Início  │──▶│ 2. Plano  │──▶│ 3. Fatia │──▶│ 4. Verif. │──▶│5. Handoff│
 │ (hook)     │   │/plan-phase│   │ código + │   │make check │   │ /handoff │
 │            │   │ você      │   │ testes   │   │revisores  │   │          │
 │            │   │ aprova    │   │          │   │           │   │          │
 └────────────┘   └───────────┘   └──────────┘   └───────────┘   └────┬─────┘
       ▲                                                              │
       └───────────────── 6. lições viram regras ◀────────────────────┘
```

1. **Início.** Abra o Claude Code na raiz. O hook mostra o estado. Diga o que
   quer em uma frase ("vamos fazer a P2-03") ou deixe o agente propor a
   próxima feature pendente.
2. **Planejar.** Ative o plan mode (Shift+Tab) e use `/plan-phase`. O agente
   lê a spec, as features e o progresso, escreve o plano e lista as decisões
   em aberto. **Você responde as decisões e aprova.** Nenhum código antes disso.
3. **Executar.** Branch `feature/<nome>` a partir da `develop`. O agente
   implementa passo a passo, com testes no mesmo passo, usando as skills
   quando couber.
4. **Verificar.**
   - `make check` verde (o agente roda e mostra o resultado).
   - Peça os revisores pelo que mudou: código em `internal/`, `pkg/` ou `cmd/` vai para o `architecture-guardian`; texto para a cliente, para o `persona-reviewer`.
   - `/code-review` no diff; `/security-review` se tocou infra, webhook ou IAM.
5. **Handoff e PR.** `/handoff` atualiza `features.json`, o plano e o
   `progress.md`, faz o commit na `feature/*` e pergunta se pode dar push.
   Com a sua autorização, o agente dá push e abre o PR para a `develop`. O
   `ci` roda no PR; você revisa e faz o merge, que leva o código para
   staging. A feature só vira `verified_staging` com o E2E verde.
   Para levar a prod, você cria a `release/x.y.z` (veja "Ciclo de uma
   versão" em `docs/spec/esteira.md`).
6. **Melhorar o harness.** Leia a seção "Lições para o harness" do plano.
   Transforme as lições baratas em regra agora; as caras viram item de
   backlog.

**Dicas de sessão:**
- Uma sessão por feature. Sessões longas acumulam contexto irrelevante; prefira encerrar com `/handoff` e abrir outra.
- Se o agente estiver patinando (três tentativas no mesmo erro), pare, pergunte o que falta de contexto e coloque isso no harness.
- Peça ao agente para dizer o plano antes de mexer em arquivos (já está no CLAUDE.md).

## 8. Exemplo de ponta a ponta

Implementar a feature **P2-03 (whatsapp-gateway)**:

1. Você: "vamos começar a P2-03". O hook já mostrou que a P2-01 e a P2-02
   estão `done_local`.
2. Você: `/plan-phase P2-03`. O agente lê `docs/spec/whatsapp.md` e
   `docs/spec/arquitetura.md` e cria `docs/plans/active/P2-03-gateway.md`.
   - **Passos:** validar a assinatura, responder 200, aplicar a allowlist, publicar `MessageReceived`, módulo Terraform.
   - **Decisões em aberto:** "a allowlist é recarregada do Secrets Manager a cada N minutos ou só no boot?".
3. Você responde "a cada 5 minutos" e aprova. O agente marca a P2-03 como
   `in_progress`.
4. O agente usa `/new-service gateway` e escreve o handler com testes
   table-driven (assinatura válida, inválida, ausente, número fora da
   allowlist, payload duplicado).
5. Ao tentar logar o número recebido, o `archcheck` falha:
   `log attribute looks like message text or phone; wrap it with logging.Mask()`.
   O agente corrige sozinho.
6. `make check` verde. O `architecture-guardian` aponta que o handler
   publica no SNS de forma síncrona antes de responder 200, o que pode
   estourar os 5s da Meta. O agente ajusta.
7. `/handoff`: P2-03 vai para `done_local`, com a evidência
   "make check ok em 2026-11-03". O progresso registra "próximo passo: P2-04".
   No plano entra a lição "o limite de 5s não estava no CLAUDE.md".
8. Você aprova a lição: uma linha nova no CLAUDE.md ("webhook responde 200
   em menos de 5s; processamento é assíncrono"). Na próxima vez, o erro do
   passo 6 não acontece.
9. Você autoriza o push de `feature/p2-03-gateway`, e o agente abre o PR
   para a `develop`. Com o `ci` verde, você faz o merge; o deploy e o E2E em
   staging passam, e a P2-03 vira `verified_staging`.
10. Fim da fase 2: você cria a `release/0.2.0` a partir da `develop`. Ela
   passa por staging, e você abre o PR para a `main`. Depois do merge, aprova
   o deploy no environment `production`, a tag `v0.2.0` é criada e você faz
   o PR de back-merge para a `develop`. A P2-03 vira `released`.

## 9. A escada de enforcement

Cada regra pode viver num degrau. Quanto mais alto, menos depende de alguém
lembrar.

| Degrau | Mecanismo | Força | Exemplo na marIA |
| --- | --- | --- | --- |
| 1 | Texto no CLAUDE.md ou na spec | O agente *deveria* seguir | "Nunca exigir confirmação da cliente" |
| 2 | Subagente revisor | Alguém *verifica* | `persona-reviewer` checa tom e conselho médico |
| 3 | Lint, archcheck, teste | O build *falha* | Só o outbound importa `pkg/meta` |
| 4 | Hook do Claude Code | A ação *nem acontece* | `terraform apply` bloqueado |
| 5 | Esteira e proteção de branch | O código *não entra* | `go test -race` como portão; sem push na `main` |

Nem toda regra consegue subir. Tom de voz, por exemplo, dificilmente vira
lint, e fica no degrau 2. Mas sempre vale perguntar se dá para subir um degrau.

Candidatos naturais para subir com o tempo:
- "Consumidor é idempotente": um teste genérico que entrega a mesma mensagem duas vezes a cada consumidor.
- "Datas em ISO 8601 com offset": um tipo próprio em `pkg/` cujo `MarshalJSON` sempre emite o offset, e lint proibindo `time.Time` cru nos eventos.
- "Mensagem com no máximo 3 frases": um teste sobre as variações fixas de texto.

## 10. Ciclo de vida de uma feature

```
todo ──▶ in_progress ──▶ done_local ──▶ verified_staging ──▶ released
         (plano            (make check     (run da esteira      (em prod,
          aprovado)         verde)          com E2E verde)       tag vx.y.z)
```

- O status só avança com evidência, registrada no campo `evidence` (comando, data, link da run ou do PR).
- O agente nunca pula etapas: `verified_staging` exige uma run real, não "deve funcionar".
- Critérios de aceite (`acceptance`) só mudam com o seu pedido, e a mudança aparece no diff do PR.
- Uma feature que voltou a quebrar volta de status, com uma nota no `progress.md`.

## 11. Manutenção do harness

Sem manutenção, o harness apodrece e passa a enganar o agente, o que é pior do
que não ter harness. Por isso:

- **A cada sessão:** `/handoff`.
- **A cada fase ou semana:** a skill `/garden`. Ela procura:
  - divergências entre a spec e o código;
  - features com status incoerente;
  - planos parados;
  - CLAUDE.md inchado;
  - regras que já poderiam virar lint.

  Dá para agendar com `/schedule`.
- **Quando a spec mudar:** atualize no mesmo PR o CLAUDE.md (se for regra
  geral), os agents e skills afetados e os critérios em `features.json`.
- **Ao concluir uma fase:** o `/handoff` move as entradas dela do `progress.md` para `docs/progress/fase-<n>.md`, e o arquivo principal fica curto.
- **Quando uma regra mecânica der falso positivo:** ajuste a regra ou use
  `archcheck:allow <motivo>`. Nunca desligue a checagem inteira.

## 12. Anti-padrões

- **CLAUDE.md enciclopédia.** Colar a spec inteira no CLAUDE.md. Use ponteiros.
- **Confiar no "pronto" do agente.** Sem `make check` e sem evidência, não está pronto.
- **Pular o plano.** Codar direto em tarefas de vários passos gera retrabalho e decisões que o agente tomou por você.
- **Corrigir no chat e não no repo.** "Não faça mais isso" numa conversa some quando a sessão acaba.
- **Sessões gigantes.** Várias features na mesma sessão misturam contexto; uma por vez.
- **Desligar o guarda-corpo para ir mais rápido.** Se um hook atrapalha um caso legítimo, ajuste o hook com cuidado e registre o porquê.
- **Subagente para tudo.** Cada subagente custa tempo e tokens. Use nos pontos de revisão, não em cada edição.
- **Docs que ninguém atualiza.** Um doc errado é pior que um doc ausente, porque o agente confia nele.

## 13. Como saber se o harness está funcionando

Observe ao longo das fases:

- **Correções por PR.** Quantas vezes você precisou corrigir o agente. Deve cair.
- **Perguntas repetidas.** O agente pergunta a mesma coisa em sessões diferentes? Falta contexto no repo.
- **Retomada.** Uma sessão nova consegue continuar o trabalho só com o hook de início e o `progress.md`?
- **Falhas pegas cedo.** Quantas violações o archcheck e o lint pegaram antes da revisão humana.
- **Retrabalho na esteira.** PRs que quebram em staging e não quebraram local apontam para testes unitários fracos ou fakes que não refletem a AWS.

## 14. Glossário

| Termo | Significado |
| --- | --- |
| Harness | Tudo em volta do modelo que direciona o agente: contexto, ferramentas, regras, verificação, memória |
| Contexto / janela de contexto | O que o modelo "vê" numa sessão; é limitado e pode ser resumido |
| Revelação progressiva | Dar ao agente o mínimo e indicar onde buscar o resto |
| Hook | Script que o Claude Code executa em eventos (início de sessão, antes ou depois de uma ferramenta) |
| Subagente | Instância do agente com contexto, prompt e ferramentas próprios, para uma tarefa delegada |
| Skill | Receita carregada sob demanda, chamada por `/nome` ou pelo agente |
| Plan mode | Modo do Claude Code em que o agente só investiga e planeja, sem editar |
| Evidência | Prova verificável de que um critério foi cumprido (saída de comando, run da esteira) |
| Garbage collection | Manutenção periódica que remove divergências e docs envelhecidos |
| Portão (gate) | Etapa que bloqueia a esteira se falhar |
