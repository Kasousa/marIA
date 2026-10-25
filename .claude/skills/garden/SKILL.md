---
name: garden
description: Manutenção do harness - encontra divergências entre docs (spec, CLAUDE.md, features.json, planos, runbooks) e o código, planos parados e regras que deveriam virar verificação automática. Use ao fim de cada fase, uma vez por semana, ou quando o administrador pedir "faz a jardinagem" ou "revisa os docs".
---

# Jardinagem do harness

Leia `HARNESS.md` para entender o papel de cada arquivo. Não edite nada nesta skill; só relate.

Procure:

1. **Spec x código:** nomes de eventos, tabelas, variáveis de ambiente, ferramentas do Bedrock e templates citados em `docs/spec/` que não batem com o código (ou o contrário).
2. **features.json:** status incoerente com o código (feature `todo` com código pronto, `done_local` sem testes) e campos `evidence` vazios em features avançadas.
3. **Planos:** arquivos em `docs/plans/active/` sem atividade no git há mais de 14 dias (`git log -1 --format=%cs -- <arquivo>`), ou com todos os passos marcados mas não movidos para `completed/`.
4. **progress.md:** entradas com mais de ~10 linhas, sem "Próximo passo", ou de fases já concluídas que deveriam estar em `docs/progress/fase-<n>.md`.
5. **CLAUDE.md:** crescendo além de ~90 linhas, regras duplicadas da spec, ou regras que já podem virar lint, teste, archcheck ou hook (cite qual).
6. **Runbooks:** previstos em `docs/runbooks/README.md` mas ausentes para fases já concluídas.
7. **Agents e skills:** referências a arquivos, comandos ou regras que não existem mais.
8. **Harness sem teste:** regra de hook ou do archcheck sem caso correspondente em `scripts/test-harness.sh` ou `scripts/testdata/`.
9. **CLAUDE.md por pasta:** pastas novas com regras próprias que ainda estão na raiz (candidatas a um CLAUDE.md na pasta).

Responda com uma lista priorizada de correções (arquivo, problema, correção proposta) e no máximo três sugestões de novas regras mecânicas. Aplique as correções só se o administrador aprovar.
