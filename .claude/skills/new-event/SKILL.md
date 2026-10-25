---
name: new-event
description: Adiciona ou altera um evento no barramento maria-events (pkg/events), mantendo o contrato retrocompatível, os publicadores e consumidores, a filtragem por tipo no SNS e os testes. Use quando um serviço precisar publicar um fato novo ou um campo novo num evento existente.
---

# Novo evento ou mudança de contrato

1. Nome: fato no passado em PascalCase (`AppointmentRegistered`). Comando só se for para o outbound (como `SendMessage`); justifique no PR.
2. Em `pkg/events/`:
   - struct do payload com tags JSON em `snake_case`;
   - constante do tipo;
   - o envelope comum já traz `event_id`, `type`, `occurred_at` (ISO 8601 com offset de São Paulo), `env` e `version`.
3. Mudança em evento existente: só acrescente campos opcionais. Remover ou renomear campo exige nova `version` e período de convivência; registre um ADR.
4. Testes em `pkg/events`: ida e volta JSON (table-driven), e um teste de compatibilidade que decodifica um JSON da versão anterior guardado em `pkg/events/testdata/`.
5. Publicador: o serviço publica pelo publisher de `pkg/events` com o atributo de mensagem `type` (usado no filter policy do SNS).
6. Consumidores: filter policy da assinatura SQS no Terraform inclui o novo tipo; o consumidor é idempotente pelo `event_id`.
7. Atualize a tabela "Serviço / Publica / Consome" da `docs/spec.md` se o fluxo mudou.
8. `make check`; peça revisão ao `architecture-guardian`.
