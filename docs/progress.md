# Progresso

Diário de sessões, mais recente em cima. Cada entrada tem no máximo ~10 linhas.
Atualize pela skill `/handoff` ao fim de cada sessão de trabalho. Ao concluir uma fase, as entradas dela vão para `docs/progress/fase-<n>.md`.

Formato:

```
## AAAA-MM-DD · <branch> · <feature id(s)>
- Feito: ...
- Verificado com: <comando ou run da esteira>
- Pendente / bloqueios: ...
- Próximo passo: ...
```

---

## 2026-10-25 · feature/fase-1-monorepo · P1-01, P1-02, P1-03
- Feito: go.mod, serviço hello (/healthz, healthcheck, SIGTERM), pkg/logging (Mask e handler que mascara qualquer tipo sob chave sensível e telefones em erros), pkg/httpx.Recoverer, Dockerfile arm64 distroless, regra 7 do archcheck (env só no config.go). P1-01, P1-02 e P1-03 em done_local.
- Verificado com: `make check` (exit 0, lint 0 issues, harness-test 72), binário testado à mão, revisão do architecture-guardian com as correções aplicadas, imagem arm64 construída e testada no Docker local.
- Pendente / bloqueios: fixar imagens base por digest (passo 12). Terraform local 1.5.7 precisa ir para 1.10+ antes do passo 5.
- Próximo passo: merge do PR na develop; depois, feature/fase-1-terraform (passo 5, bootstrap).

## 2026-10-25 · feature/fase-1-monorepo · plano da Fase 1 e ADR 0004
- Feito: plano da Fase 1 revisado (/plan-phase), D1-D5 respondidas. ADR 0004: uma task por serviço em Fargate Spot ARM nos dois ambientes, prod desligado 00:00-05:50, ReminderDue com mais de 15 min de atraso descartado, risco de custo (~US$ 32/mês) aceito. Spec, CLAUDE.md, infra/CLAUDE.md e features P1-04, P1-07 e P3-05 atualizados.
- Verificado com: e-mail do budget fora do repositório (grep); features.json validado com jq.
- Pendente / bloqueios: aprovação do plano pelo administrador; Terraform local 1.5.7 precisa ir para 1.10+ antes do passo 5.
- Próximo passo: com o plano aprovado, passo 1 (go.mod e serviço hello) nesta branch.

## 2026-10-24 · feature/adr-0003-infra · melhorias do harness
- Feito: testes do harness no repo (scripts/test-harness.sh + fixtures em scripts/testdata, `make harness-test` no `make check`); spec dividida em docs/spec/ com glossário; CLAUDE.md por pasta (infra/, .github/, e2e/); rotação do progress por fase; catálogo de eventos gerado como critério da P2-01.
- Verificado com: `make harness-test` (71 casos), teste de mutação (regra de push quebrada de propósito, 12 falhas detectadas), `scripts/archcheck.sh` ok, nenhuma linha da spec antiga perdida na divisão.
- Pendente / bloqueios: os mesmos da entrada abaixo.
- Próximo passo: administrador aprova o PR; depois, /plan-phase 1 e feature/fase-1-monorepo.

## 2026-10-24 · feature/adr-0003-infra · decisões da Fase 1 (ADR 0003)
- Feito: região us-east-1, conta AWS única, entrada por API Gateway HTTP API + VPC Link + Cloud Map (sem ALB), domínio próprio no backlog. Spec, CLAUDE.md, plano da Fase 1 e features P1-05..P1-07 atualizados.
- Verificado com: revisão de texto; features.json validado com jq. Disponibilidade do Haiku 4.5 por região conferida na documentação da AWS.
- Pendente / bloqueios: nome do bucket de state; estimativa de custo em us-east-1. O administrador pediu para não iniciar a execução do plano ainda.
- Próximo passo: o administrador terminar de revisar o entendimento do projeto; depois, /plan-phase 1 e feature/fase-1-monorepo.

## 2026-10-24 · main · Git Flow (ADR 0002)
- Feito: adotado o Git Flow padrão (feature/* -> develop; release/x.y.z e hotfix/x.y.z -> main com back-merge). O fluxo simplificado testado antes foi revertido. Spec, CLAUDE.md, ADR 0002, features P1-04/P1-08..P1-13, plano da Fase 1, HARNESS.md, /handoff e template de PR atualizados.
- Verificado com: 32 casos de Git Flow no guard-bash.sh (incluindo redirecionamentos como 2>&1), aviso de back-merge do session-context.sh e as 21 checagens anteriores, todos verdes.
- Pendente / bloqueios: nenhum novo.
- Próximo passo: mesmo da entrada abaixo.

## 2026-10-24 · main · harness
- Feito: estrutura de harness criada (docs/, .claude/, Makefile, lint, archcheck, template de PR). Spec movida para docs/spec.md.
- Verificado com: `bash -n` nos hooks e scripts; JSON validado com jq.
- Pendente / bloqueios: Go, gh e golangci-lint não estão instalados na máquina. Sem branch `develop` ainda.
- Próximo passo: instalar as ferramentas, criar `develop`, abrir `feature/fase-1-monorepo` e executar `docs/plans/active/fase-1-fundacao.md` a partir do item P1-01.
