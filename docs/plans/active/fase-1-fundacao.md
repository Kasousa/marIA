# Fase 1: Fundação

- Features: P1-01 a P1-13
- Branches: `feature/fase-1-monorepo` (passos 1-4), `feature/fase-1-terraform` (passos 5-10), `feature/fase-1-workflows` (passos 11-16); depois `release/0.1.0` (passo 17)
- Spec: `docs/spec/infra.md`, `docs/spec/esteira.md`, `docs/spec/README.md` (Roteiro, item 1); ADRs 0002, 0003 e 0004
- Status: em execução (aprovado em 2026-10-25)

## Objetivo
O serviço `hello` com testes sobe em staging por um merge na `develop`, uma `release/0.1.0` validada em staging chega em prod pelo merge na `main` e volta para a `develop` pelo back-merge, e um teste quebrado de propósito bloqueia o PR (Git Flow do ADR 0002).

## Fora do escopo
Qualquer serviço real (gateway, conversation...), DynamoDB de negócio, SNS/SQS, Scheduler, Bedrock, Secrets da Meta. O E2E desta fase é só o `/healthz` do `hello` pela URL do API Gateway.

## Decisões em aberto
- [x] Caminho do módulo Go: `github.com/Kasousa/maria`.
- [x] Região AWS: `us-east-1` (ADR 0003).
- [x] Contas AWS: uma só, a pessoal, para staging e prod (ADR 0003).
- [x] HTTPS do webhook: API Gateway HTTP API com a URL padrão `execute-api`, sem ALB; domínio próprio em melhorias futuras (ADR 0003).
- [x] **D1. Orçamento:** cada serviço na sua task, Fargate Spot nos dois ambientes, sem Lambda, prod desligado das 00:00 às 05:50, lembrete com mais de 15 min de atraso descartado, risco de custo aceito (~US$ 32/mês). ADR 0004.
- [x] **D2. Camada compartilhada:** `infra/envs/shared/` com ECR, OIDC e roles da esteira.
- [x] **D3. Arquitetura das imagens:** `linux/arm64` (Graviton).
- [x] **D4. E-mail do budget alarm:** passado por `-var` no bootstrap, aplicado pelo administrador; nunca entra no repositório.
- [x] **D5. Bucket de state:** `maria-tfstate-<id da conta>`.

## Pré-requisitos do administrador (fora do repositório)
- Atualizar o Terraform: a versão local é 1.5.7, e o lock nativo do S3 (`use_lockfile`) exige 1.10+. O Homebrew oficial parou na 1.5.7 por causa da licença; use `brew install hashicorp/tap/terraform`.
- MFA no usuário root da conta AWS e um usuário ou SSO seu com credencial temporária para o bootstrap.

## Passos

### Monorepo: `feature/fase-1-monorepo` (P1-01 a P1-03)
- [x] 1. `go.mod` (`go 1.27`), `cmd/hello/main.go` e `internal/hello/`: `GET /healthz` em chi, slog JSON via `pkg/logging`, shutdown gracioso no SIGTERM, configuração por env (`internal/hello/config.go`). Testes do handler e do config. P1-01 vira `done_local`.
- [x] 2. `pkg/logging`: `Mask()` para telefone E.164 e texto livre, e um handler slog que mascara atributos sensíveis. Testes table-driven. P1-02 vira `done_local`.
- [x] 3. `Dockerfile` multi-stage com `ARG SERVICE` e `TARGETARCH`, `CGO_ENABLED=0`, final `gcr.io/distroless/static:nonroot`; `.dockerignore`. Build local só para teste (`docker build`, sem push). P1-03 vira `done_local`.
- [ ] 4. `make check` verde de ponta a ponta. PR para a `develop`.

### Terraform: `feature/fase-1-terraform` (P1-04 a P1-07)
- [ ] 5. `infra/bootstrap/`: bucket de state (versionamento, criptografia, bloqueio de acesso público) e budget alarm 80% e 100% (D4). Runbook `docs/runbooks/bootstrap.md`. **Aplicado uma vez pelo administrador**, a única exceção à regra "Terraform só pela esteira".
- [ ] 6. `infra/modules/github-oidc` e `infra/envs/shared/` (D2): provedor OIDC, roles `maria-ci-staging` e `maria-ci-prod` com `sub` restrito às branches do ADR 0002, ECR `maria-hello` com lifecycle. Primeira aplicação manual (no mesmo runbook); depois só pela esteira.
- [ ] 7. Módulos `network` (VPC, 2 subnets públicas, sem NAT), `api-gateway` (HTTP API, VPC Link, namespace Cloud Map, rota `/healthz`), `ecs-cluster` e `ecs-service` (task ARM, logs com 14 dias, registro SRV no Cloud Map, circuit breaker opcional) e KMS. A forma do `ecs-service` depende da D1.
- [ ] 8. `infra/envs/staging/base` (rede, API Gateway, KMS) e `infra/envs/staging/compute` (cluster e serviço `hello` em Fargate Spot). P1-05 e P1-06 viram `done_local` com `make tf-check` verde.
- [ ] 9. `infra/envs/prod` (mesmos módulos, Spot ARM, circuit breaker com rollback, schedules 00:00 e 05:50 que zeram e restauram o `desiredCount`). P1-07 vira `done_local`.
- [ ] 10. Revisão: `/security-review` nas políticas IAM e trust policies. PR para a `develop`.

### Esteira: `feature/fase-1-workflows` (P1-08 a P1-12)
- [ ] 11. `.github/workflows/ci.yml`: lint, archcheck, `make harness-test`, `go test ./... -race`, build da imagem, `terraform plan` do destino; nos PRs para a `main`, valida origem `release/x.y.z` ou `hotfix/x.y.z`, status `staging-ok` e `git merge-base --is-ancestor origin/main HEAD`. P1-08.
- [ ] 12. `.github/workflows/deploy-staging.yml`: `concurrency` de staging, build e push com tag do hash da árvore, liga staging, deploy, smoke do `/healthz` pela URL do API Gateway (o E2E desta fase), tag e status `staging-ok`, step Playwright com `if: false`. P1-09.
- [ ] 13. `.github/workflows/deploy-prod.yml`: environment `production`, exige `staging-ok` pelo hash da árvore, deploy sem rebuild, smoke, tag `vx.y.z` do nome da branch, release no GitHub, link do back-merge no resumo. P1-10.
- [ ] 14. `.github/workflows/staging-power.yml`: manual (`on`/`off`) e cron `0 2 * * *` (23h em São Paulo). P1-11.
- [ ] 15. Configuração do GitHub (via `gh api`, mostrada antes de aplicar): environment `production` com você como revisor obrigatório e restrito à `main`; ruleset passa a exigir o check do `ci`. Runbooks `docs/runbooks/branch-protection.md` e `docs/runbooks/staging-power.md`. P1-12.
- [ ] 16. PR para a `develop`: o merge dispara o primeiro `deploy-staging` real. P1-05, P1-06, P1-08, P1-09 e P1-11 viram `verified_staging` com o link da run.

### Prova do Git Flow (P1-13)
- [ ] 17. PR com teste quebrado de propósito é bloqueado pelo `ci` (depois fechado). `release/0.1.0` passa por staging, PR para a `main`, aprovação no environment, prod, tag `v0.1.0`, back-merge. Uma `hotfix/0.1.1` de teste faz o caminho a partir da `main`. P1-07, P1-10, P1-12 e P1-13 viram `released`.

## Como verificar
- Local, a cada passo: `make check`; nos passos de infra, `make tf-check`.
- Staging: run verde do `deploy-staging` e `curl https://<api-id>.execute-api.us-east-1.amazonaws.com/healthz` respondendo 200.
- Prod: run verde do `deploy-prod`, tag `v0.1.0` no GitHub, mesmo digest de imagem em staging e prod.
- Evidências (links das runs) no campo `evidence` do `features.json`.

## Riscos
- Ovo e galinha do OIDC: o bootstrap e a primeira aplicação do `shared` são manuais; fica tudo no runbook.
- Fargate Spot pode interromper o staging no meio do smoke: a esteira tenta de novo uma vez.
- Custo acima do orçamento (~US$ 32/mês), aceito no ADR 0004; o budget alarm de 80% deve disparar.
- Falta de capacidade Spot pode deixar um serviço fora do ar por um tempo; a saída prevista é o gateway em Fargate on-demand.
- O IPv4 público é cobrado por hora e por task; entra na estimativa de custo.

## Registro de execução
- 2026-10-24: plano revisado pelo /plan-phase. Passos detalhados por branch, decisões D1-D5 e pré-requisitos adicionados.
- 2026-10-25: D1-D5 respondidas pelo administrador; ADR 0004 (Spot em prod e janela noturna).
- 2026-10-25: passos 1 e 2 concluídos e revisados pelo `architecture-guardian`; correções aplicadas. `make check` verde. Passo 3: Dockerfile com trava de arm64; build e container verificados com o Docker local (imagem de 3,6 MB).

## Lições para o harness
- O `forbidigo` não analisa expressões em nível de pacote (`var x = os.Getenv(...)`). Para variáveis de ambiente, virou a regra 7 do archcheck, com fixture. Para o `Recoverer` do chi, o uso real (`r.Use` dentro de função) é pego.
- A revisão do `architecture-guardian` achou vazamentos no `pkg/logging` que os testes não cobriam ([]byte, error, Stringer, LogValuer e maiúsculas sob chave sensível, telefone dentro de erro) e o `Recoverer` do chi imprimindo fora do slog. Corrigido com testes; o `Recoverer` do chi agora é proibido no lint.
- Pendente para o passo 12: fixar `golang` e `distroless` por digest no Dockerfile (precisa do Docker rodando).
- O Dockerfile tinha `ARG TARGETARCH=arm64`, que não tem efeito com BuildKit; a arquitetura vem do `--platform`. Regra para o passo 12: o build da esteira sempre passa `--platform linux/arm64` (anotar no .github/CLAUDE.md).
