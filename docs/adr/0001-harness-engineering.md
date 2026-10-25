# ADR 0001: Desenvolver com harness engineering

- Data: 2026-10-24
- Status: aceita

## Contexto
O código da marIA será escrito majoritariamente por agentes (Claude Code), com o administrador revisando. Sem estrutura, cada sessão começa do zero, regras de arquitetura viram só texto e erros se repetem.

## Decisão
O repositório mantém um harness explícito: `CLAUDE.md` curto como índice, `docs/` como fonte da verdade (spec, features, progresso, planos, ADRs), regras mecânicas (lint, archcheck, hooks) e subagentes e skills em `.claude/`. O funcionamento está em `HARNESS.md`.

## Alternativas consideradas
- Só um `CLAUDE.md` grande: estoura o contexto e envelhece rápido.
- Sem agentes: perde o objetivo de estudo de trabalhar com agentes.

## Consequências
Toda mudança relevante atualiza docs junto com o código. Regras recorrentes devem migrar de texto para verificação automática.
