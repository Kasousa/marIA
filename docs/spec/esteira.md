# Esteira e Git Flow

Branches, proteção, workflows e steps de staging e prod.

## Esteira de CI/CD (GitHub Actions + Git Flow)

O repositório segue o Git Flow padrão: a branch `develop` integra o trabalho e entrega em staging, e a branch `main` contém só o que está em prod. Todo deploy na AWS passa pelo GitHub Actions; ninguém aplica Terraform nem publica imagem da própria máquina. A esteira para no primeiro step que falhar. A decisão está registrada no ADR 0002.

**Branches**

| Branch | Sai de | Entra em | O que acontece |
| --- | --- | --- | --- |
| `feature/*` | `develop` | `develop` (PR) | Workflow `ci` no PR; o merge dispara o deploy em staging |
| `develop` | — | — | Cada push faz deploy em staging |
| `release/x.y.z` | `develop` | `main` (PR) e depois `develop` (back-merge) | Congela o escopo da versão; cada push faz deploy em staging; só entram correções, versão e changelog |
| `hotfix/x.y.z` | `main` | `main` (PR) e depois `develop` (back-merge) | Correção urgente; cada push faz deploy em staging antes de ir para prod |
| `main` | — | — | Cada push faz deploy em prod e cria a tag `vx.y.z` |

**Ciclo de uma versão**

1. Features entram na `develop` por PR, com o `ci` verde. Cada merge vai para staging.
2. Quando o escopo da versão está pronto, o administrador cria `release/x.y.z` a partir da `develop`. A esteira faz o deploy da release em staging e roda o E2E. Correções encontradas em staging são commitadas na própria release.
3. O administrador abre o PR da `release/x.y.z` para a `main`. Com o `ci` verde e o status `staging-ok` no último commit da release, faz o merge (merge commit, sem fast-forward).
4. O merge na `main` dispara o `deploy-prod`, que pede aprovação no environment `production`, entrega a mesma imagem validada em staging e cria a tag `vx.y.z`, com o número tirado do nome da branch.
5. **Back-merge:** o administrador abre o PR da `main` para a `develop`, para que as correções feitas na release ou na hotfix voltem para a `develop`. A release ou hotfix é apagada depois disso.

Uma hotfix segue os passos 2 a 5, mas sai da `main` em vez da `develop`.

**Proteção das branches**

- `develop` e `main` não aceitam push direto: só por pull request, com o workflow `ci` verde.
- A `main` só aceita PR vindo de `release/*` ou `hotfix/*` com nome no formato `x.y.z`. Um check do `ci` valida a branch de origem, o status `staging-ok` no commit de origem e se a branch de origem contém a `main` atual (`git merge-base --is-ancestor`).
- Merge na `main` sem fast-forward (merge commit), como manda o Git Flow.
- Como o projeto tem um único administrador e o GitHub não deixa ninguém aprovar o próprio PR, os PRs não exigem aprovação de revisor. A aprovação humana obrigatória fica no environment `production`, antes do deploy em prod.

**Por que o hash da árvore encontra a imagem certa:** a branch de origem contém a `main` atual (o `ci` verifica). Então o merge commit na `main` tem exatamente a mesma árvore de arquivos do último commit da release ou hotfix, e esse commit foi o que passou em staging.

**Workflows**

| Workflow | Quando roda | O que faz |
| --- | --- | --- |
| `ci` | Todo PR para `develop` ou `main` | Lint, testes unitários (portão), build da imagem e `terraform plan` do ambiente de destino; nos PRs para a `main`, também as checagens de origem |
| `deploy-staging` | Push na `develop`, em `release/**` ou em `hotfix/**` | Esteira de staging |
| `deploy-prod` | Push na `main` | Esteira de prod |
| `staging-power` | Manual e todo dia às 23h | Liga ou desliga a camada compute do staging |

Staging é um ambiente só: o `deploy-staging` usa `concurrency` para que um deploy da `develop` e um da release não rodem ao mesmo tempo. Enquanto uma release está em validação, convém não mesclar features na `develop`, para o staging ficar com a versão da release.

**Steps do `deploy-staging`**

1. **Lint e análise estática:** `go vet`, `golangci-lint`, `scripts/archcheck.sh`, `terraform fmt -check` e `terraform validate`.
2. **Testes unitários (portão obrigatório):** `go test ./... -race` em todos os serviços e pacotes. Se qualquer teste falhar, a esteira para aqui e nenhum step seguinte roda.
3. **Build e push:** imagem Docker de cada serviço, com a tag do hash da árvore do Git (`git rev-parse HEAD^{tree}`), push para o ECR.
4. **Liga staging:** se a camada compute estiver desligada, aplica o Terraform dela.
5. **Deploy em staging:** `terraform apply` de `staging/base` e `staging/compute` com a nova tag; espera os serviços ECS ficarem estáveis.
6. **Testes E2E em staging:** suíte em Go que envia webhooks assinados simulando o número simulador (consulta, remédio, água, resposta livre, mídia não suportada) e confere o resultado no DynamoDB, nos agendamentos e nas DLQs vazias.
7. **Testes funcionais com Playwright (futuro):** step já previsto, desligado na v1. Quando ativo, também é portão.
8. **Registro da validação:** a imagem recebe a tag adicional `staging-ok` no ECR, e o commit recebe o status `staging-ok`.

**Steps do `deploy-prod` (branch `main`)**

1. **Testes unitários (portão obrigatório):** rodam de novo sobre o merge commit; se falharem, nada segue.
2. **Localiza a imagem validada:** pelo hash da árvore do merge commit; a esteira exige a tag `staging-ok` e, sem ela, para.
3. **Aprovação manual:** o environment `production` do GitHub exige aprovação do administrador.
4. **Deploy em prod:** a mesma imagem é implantada em prod, sem rebuild. Se o health check falhar, o circuit breaker do ECS faz rollback.
5. **Smoke test em prod:** health check de cada serviço e verificação de que as DLQs continuam vazias.
6. **Tag da versão:** cria a tag Git `vx.y.z` (do nome da `release/*` ou `hotfix/*` mesclada) e uma release no GitHub com as mudanças.
7. **Lembrete de back-merge:** o resumo da run traz o link para abrir o PR da `main` para a `develop`.

**Acesso à AWS**

- Autenticação por OIDC do GitHub, com uma role por ambiente. A role de staging só pode ser assumida a partir da `develop`, de `release/*` e de `hotfix/*`; a de prod, só por jobs da `main` no environment `production`.
- Segredos da Meta ficam no Secrets Manager; o GitHub guarda só o ARN das roles.

O administrador recebe encaminhamentos dos dois ambientes ao mesmo tempo. Como cada ambiente tem seu número da marIA, as conversas chegam em chats separados no WhatsApp dele, além do prefixo `[STAGING]` ou `[PROD]` no texto.
