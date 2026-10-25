# Progresso

Diário de sessões, mais recente em cima. Cada entrada tem no máximo ~10 linhas.
Atualize pela skill `/handoff` ao fim de cada sessão de trabalho.

Formato:

```
## AAAA-MM-DD · <branch> · <feature id(s)>
- Feito: ...
- Verificado com: <comando ou run da esteira>
- Pendente / bloqueios: ...
- Próximo passo: ...
```

---

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
