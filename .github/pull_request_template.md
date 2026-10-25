## O que muda

<!-- Uma ou duas frases. -->

- Tipo: feature -> develop | release -> main | hotfix -> main | back-merge main -> develop
- Features: <!-- ids do docs/features.json, ex.: P2-03 -->
- Plano: <!-- docs/plans/active/... -->

## Checklist (feature -> develop)

- [ ] `make check` verde (inclui `go test ./... -race` e `scripts/archcheck.sh`)
- [ ] `make tf-check` verde e `plan` revisado (se tocou `infra/`)
- [ ] Testes table-driven junto com o código
- [ ] Revisado por `architecture-guardian` (e `persona-reviewer` se mudou texto para a cliente)
- [ ] `/code-review` no diff; `/security-review` se tocou infra, webhook ou IAM
- [ ] `docs/features.json`, plano e `docs/progress.md` atualizados
- [ ] Spec ou ADR atualizados, se o comportamento ou uma decisão mudou
- [ ] Nenhum texto de mensagem ou telefone em claro em logs ou fixtures

## Checklist (release ou hotfix -> main)

- [ ] Branch no formato `release/x.y.z` ou `hotfix/x.y.z`, contendo a `main` atual
- [ ] Status `staging-ok` verde no último commit (deploy e E2E em staging)
- [ ] Testei no número simulador o que mudou para a cliente
- [ ] Features da versão em `verified_staging` no `docs/features.json`
- [ ] Depois do merge: aprovar o deploy no environment `production` e abrir o back-merge `main` -> `develop`

## Lições para o harness

<!-- Algo que o agente errou ou precisou perguntar e que deveria virar regra, doc ou skill? -->
