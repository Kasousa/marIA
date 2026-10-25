---
name: plan-phase
description: Cria ou atualiza um plano de execução em docs/plans/active/ para uma fase do roteiro da marIA ou para uma feature do docs/features.json. Use quando o administrador disser "vamos começar a fase N", "planeja a feature X" ou antes de qualquer trabalho que toque mais de um pacote.
---

# Planejar uma fase ou feature

1. Identifique o alvo: número da fase ou ids de feature (`P2-03`...). Se não estiver claro, pergunte.
2. Leia:
   - a seção correspondente de "Roteiro de implementação" em `docs/spec.md` e as seções técnicas que ela cita;
   - as features do alvo em `docs/features.json` (critérios de aceite);
   - `docs/progress.md` (última entrada) e planos em `docs/plans/active/`, para não duplicar.
3. Se já existir plano ativo para o alvo, atualize-o em vez de criar outro.
4. Copie `docs/plans/TEMPLATE.md` para `docs/plans/active/fase-<n>-<slug>.md` (ou `<feature-id>-<slug>.md`) e preencha:
   - Passos pequenos, cada um terminando com `make check` verde e, quando possível, com uma feature avançando para `done_local`.
   - "Decisões em aberto" com tudo que a spec não responde. Não invente respostas.
   - "Como verificar" com comandos concretos e o teste E2E que prova o critério em staging.
   - Arquivos e pacotes que cada passo vai tocar.
5. Marque as features do plano como `in_progress` em `docs/features.json` só quando o administrador aprovar o plano.
6. Mostre ao administrador um resumo de 5 a 10 linhas e as decisões em aberto. Não comece a codar na mesma resposta.
