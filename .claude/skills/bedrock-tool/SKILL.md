---
name: bedrock-tool
description: Adiciona ou altera uma ferramenta que o Claude Haiku 4.5 pode chamar no conversation-service via Bedrock Converse (register_appointment, update_medication, pause_water_reminders...), com schema, validação, chamada à API interna do reminders e testes. Use ao implementar qualquer ferramenta da tabela de ferramentas da spec.
---

# Ferramenta do Bedrock

Contexto: seção "Bedrock: modelo, prompt e ferramentas" da `docs/spec.md`. O modelo nunca grava nada; ele escolhe a ferramenta, o conversation valida e chama o reminders.

1. **Definição** em `internal/conversation/tools/`: nome em `snake_case`, descrição curta em inglês dizendo quando usar e quando perguntar à cliente, e JSON Schema de entrada com `required` explícito. Datas em ISO 8601 com offset; horários `HH:MM`.
2. **Validação** no serviço, nunca confiando no modelo: data no futuro, horário válido, nome não vazio, prazo coerente. Entrada inválida volta ao modelo como `toolResult` com `status: error` e uma mensagem que o ajude a perguntar à cliente o que falta.
3. **Execução**: chamada HTTP à API interna do reminders (cliente em `internal/conversation/remindersclient`), com timeout e contexto. Erros do reminders também voltam como `toolResult` de erro; a marIA responde com carinho e sem expor detalhes técnicos.
4. **Loop de tool use**: no máximo N iterações por mensagem (configurável) para não entrar em laço; `maxTokens` 1024; histórico de no máximo 20 mensagens.
5. **Reminders**: o endpoint correspondente valida de novo (defesa em profundidade) e é idempotente pelo `wamid` da mensagem que originou a chamada.
6. **Testes** com um fake do cliente Bedrock que devolve `toolUse` roteirizado:
   - caminho feliz;
   - campo faltando (o modelo deve perguntar);
   - data relativa resolvida em America/Sao_Paulo ("quinta que vem" a partir de um relógio fixo);
   - consulta a menos do `APPOINTMENT_LEAD_TIME` (registra sem lembrete);
   - erro do reminders.
7. Se mudou o system prompt ou mensagens de confirmação, peça revisão ao `persona-reviewer`. Sempre peça ao `architecture-guardian`.
8. Para detalhes da API Converse e do formato de tool use, use a skill `claude-api`.
