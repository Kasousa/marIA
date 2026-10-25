# Fase 1: Fundação

- Features: P1-01 a P1-13
- Branch: uma `feature/*` por grupo de passos (ex.: `feature/fase-1-monorepo`, `feature/fase-1-terraform`, `feature/fase-1-workflows`)
- Spec: `docs/spec/infra.md`, `docs/spec/esteira.md`, `docs/spec/README.md` (Roteiro, item 1)
- Status: rascunho

## Objetivo
O serviço `hello` com testes sobe em staging por um merge na `develop`, uma `release/0.1.0` validada em staging chega em prod pelo merge na `main` e volta para a `develop` pelo back-merge, e um teste quebrado de propósito bloqueia o PR (Git Flow do ADR 0002).

## Fora do escopo
Qualquer serviço real (gateway, conversation...), DynamoDB de negócio, SNS/SQS, Bedrock.

## Decisões em aberto
- [x] Caminho do módulo Go: `github.com/Kasousa/maria`.
- [x] Região AWS: `us-east-1` (ADR 0003).
- [x] Contas AWS: uma só, a pessoal, para staging e prod (ADR 0003).
- [x] HTTPS do webhook: API Gateway HTTP API com a URL padrão `execute-api`, sem ALB; domínio próprio em melhorias futuras (ADR 0003).
- [ ] Nome do bucket de state (sugestão: `maria-tfstate-<id da conta>`).
- [ ] Custos em `us-east-1` (Fargate, API Gateway, Bedrock) dentro de R$150/mês: confirmar na calculadora da AWS antes do primeiro deploy.

## Passos

### Monorepo (P1-01 a P1-03)
- [ ] 1. `go mod init`, `cmd/hello/main.go` com `GET /healthz` em chi e `log/slog` JSON. Teste do handler.
- [ ] 2. `pkg/logging`: `Mask()` para telefone e texto, handler slog que nunca emite texto em claro. Testes table-driven.
- [ ] 3. `Dockerfile` multi-stage com `ARG SERVICE`, final `gcr.io/distroless/static`.
- [ ] 4. `make check` verde (fmt, vet, golangci-lint, archcheck, test -race).

### Terraform (P1-04 a P1-07)
- [ ] 5. `infra/bootstrap/`: bucket de state com versionamento e lock nativo do S3 (aplicado uma única vez pelo administrador, documentado em `docs/runbooks/bootstrap.md`).
- [ ] 6. Módulos `network`, `api-gateway` (HTTP API, VPC Link, namespace Cloud Map), `ecs-cluster`, `ecs-service` (registro SRV no Cloud Map), `github-oidc`, mais ECR e KMS.
- [ ] 7. `envs/staging/base`, `envs/staging/compute`, `envs/prod`. `terraform fmt -check` e `validate` verdes.
- [ ] 8. Budget alarm em 80% e 100%.

### Esteira (P1-08 a P1-13)
- [ ] 9. `ci.yml` (PRs para `develop` e `main`): lint, archcheck, `go test ./... -race`, build, `terraform plan` do destino; nos PRs para a `main`, valida origem `release/x.y.z` ou `hotfix/x.y.z`, status `staging-ok` e `git merge-base --is-ancestor origin/main HEAD`.
- [ ] 10. `deploy-staging.yml` (push na `develop`, `release/**`, `hotfix/**`): `concurrency` de staging, tag do hash da árvore, liga staging, deploy, E2E, tag e status `staging-ok`, step Playwright com `if: false`.
- [ ] 11. `deploy-prod.yml` (push na `main`): environment `production` com aprovação, sem rebuild, smoke test, tag `vx.y.z` a partir do nome da branch mesclada, release no GitHub e link do PR de back-merge no resumo.
- [ ] 12. `staging-power.yml` manual e com cron das 23h (02:00 UTC).
- [ ] 13. Proteção de `develop` e `main` (rulesets via `gh api` ou pela UI, documentado em `docs/runbooks/branch-protection.md`).
- [ ] 14. Prova do Git Flow: PR com teste quebrado é bloqueado; `release/0.1.0` passa por staging, chega em prod e volta para a `develop` pelo back-merge; uma `hotfix/0.1.1` de teste faz o mesmo caminho a partir da `main`.

## Como verificar
- Local: `make check`, `make tf-check`.
- Esteira: links das runs de `deploy-staging` e `deploy-prod` no campo `evidence` do `features.json`.

## Riscos
- Primeira aplicação do OIDC precisa ser manual (ovo e galinha): fica no runbook de bootstrap, é a única exceção à regra "Terraform só pela esteira".
- Fargate Spot pode interromper o staging no meio do E2E: a esteira deve tolerar uma nova tentativa.

## Registro de execução

## Lições para o harness
