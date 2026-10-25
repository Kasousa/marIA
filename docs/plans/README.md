# Planos de execução

- `active/`: planos em andamento. Idealmente um ou dois ao mesmo tempo.
- `completed/`: planos concluídos, movidos para cá pela skill `/handoff` quando todas as features deles estiverem em `verified_staging` ou depois.
- `TEMPLATE.md`: modelo para planos novos. A skill `/plan-phase` usa ele.

Planos são versionados com o código: o PR que implementa um passo também marca o checkbox.
