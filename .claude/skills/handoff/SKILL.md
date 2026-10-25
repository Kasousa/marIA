---
name: handoff
description: Encerra uma sessão de trabalho deixando o estado pronto para a próxima - roda make check, atualiza docs/progress.md, docs/features.json e o plano ativo, e propõe a mensagem de commit. Use quando o administrador disser "fecha a sessão", "handoff", "vamos parar por hoje" ou ao terminar uma feature.
---

# Handoff de sessão

1. Rode `make check` (e `make tf-check` se tocou `infra/`). Guarde o resultado; não esconda falhas.
2. `docs/features.json`: para cada feature tocada, ajuste `status` só com evidência (`done_local` exige `make check` verde; `verified_staging` exige link de run da esteira) e preencha `evidence`.
3. Plano ativo: marque os passos concluídos, adicione uma linha datada em "Registro de execução" e, se houve erro do agente ou explicação repetida, uma linha em "Lições para o harness". Se o plano acabou, mova para `docs/plans/completed/`.
4. `docs/progress.md`: nova entrada no topo (logo abaixo do `---`) no formato do arquivo, no máximo ~10 linhas, com "Próximo passo" concreto o bastante para outra sessão começar sem perguntar nada.
5. Se uma "Lição para o harness" for barata de implementar agora (uma linha no CLAUDE.md, uma regra no archcheck), proponha a mudança ao administrador; não aplique sem aprovação.
6. Mostre o `git status` e proponha uma mensagem de commit em inglês no formato `type(scope): summary`. Commit só se o administrador pedir, e sempre numa `feature/*`, `release/x.y.z` ou `hotfix/x.y.z` (nunca em `main` ou `develop`).
7. Pergunte se pode dar push e abrir o PR (`gh pr create --base develop` para `feature/*`; `--base main` para release ou hotfix). Lembre que push em `release/*` ou `hotfix/*` faz deploy em staging. O merge do PR é sempre do administrador.
8. Se o trabalho foi uma release ou hotfix já mesclada na `main`, lembre o PR de back-merge `main` -> `develop`.
