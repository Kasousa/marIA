# ADR 0002: Git Flow padrão, com release e hotfix validadas em staging

- Data: 2026-10-24
- Status: aceita

## Contexto
A marIA é um projeto de estudo, e praticar um modelo de branches usado no mercado faz parte do objetivo. A spec inicial já usava Git Flow, mas com duas lacunas:
- a hotfix só passava por staging com um disparo manual;
- o back-merge para a `develop` não era uma etapa explícita.

## Decisão
Git Flow padrão:
- `main` (prod) e `develop` (integração e staging) são permanentes.
- `feature/*` sai da `develop` e volta por PR.
- `release/x.y.z` sai da `develop`, é estabilizada em staging e entra na `main` por PR.
- `hotfix/x.y.z` sai da `main`, passa por staging e entra na `main` por PR.
- Toda entrada na `main` é seguida de um PR de back-merge da `main` para a `develop`.
- `deploy-staging` roda em push na `develop`, em `release/**` e em `hotfix/**`.
- `deploy-prod` roda em push na `main`, com aprovação no environment `production`.
- A tag `vx.y.z` vem do nome da branch mesclada.

## Alternativas consideradas
- **Fluxo simplificado com branches por ambiente** (feature/* com merge automático na `develop`, PR `develop` → `main` aberto pela esteira, sem release nem hotfix). Foi aplicado e revertido no mesmo dia. É mais simples, mas não ensina release, hotfix nem back-merge, e uma correção urgente arrastaria tudo o que estivesse na `develop`.
- **GitHub Flow** (só `main`, ambientes como etapas da esteira). Seria o mais natural para entrega contínua com um único desenvolvedor, mas perde a separação `develop`/`main` e a prática de versionamento por release.

## Consequências
- **Tree-hash continua confiável:** o `ci` exige, nos PRs para a `main`, que a branch de origem contenha a `main` atual. Assim, a árvore do merge commit é a mesma do último commit validado em staging.
- **Aprovação humana:** com um único administrador, o GitHub não permite aprovar o próprio PR. Os PRs não exigem revisor, e a aprovação obrigatória fica no environment `production`.
- **Staging compartilhado:** é um só ambiente para `develop`, release e hotfix. A `concurrency` evita deploys simultâneos; durante uma release, evite mesclar features na `develop`.
- **Back-merge esquecido** é o erro clássico do Git Flow. O hook `SessionStart` avisa quando a `main` tem commits que a `develop` não tem, e o `deploy-prod` deixa o link do PR no resumo da run.
- **Guarda-corpo do agente:** o hook `guard-bash.sh` só permite push de `feature/*`, `release/x.y.z` e `hotfix/x.y.z`. Ele bloqueia commit e merge na `develop` e na `main` e bloqueia `gh pr merge`.
