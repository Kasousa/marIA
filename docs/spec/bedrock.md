# Bedrock

Modelo, system prompt, ferramentas e guardrails do conversation-service.

## Bedrock: modelo, prompt e ferramentas

O conversation-service usa Claude Haiku 4.5 no Bedrock, em `us-east-1`, pela API Converse com tool use. O modelo nunca grava nada direto: ele escolhe uma ferramenta, e o serviço valida e chama o reminders-service.

**System prompt (resumo)**

- Você é a marIA, amiga carinhosa da usuária, fala português do Brasil, frases curtas.
- Use uma referência leve ao Snoopy e sua turma, sem repetir a mesma piada da mensagem anterior.
- Nunca cobre, nunca insista, nunca dê conselho médico.
- Data e hora atuais: `{{now}}` (America/Sao\_Paulo). Consultas e remédios ativos: `{{context}}`.
- Use as ferramentas para registrar, alterar ou cancelar. Se faltar informação, pergunte.

**Ferramentas**

| Ferramenta | Entrada | Efeito |
| --- | --- | --- |
| `register_appointment` | especialidade, datetime ISO, local? | Grava consulta e agenda lembrete 1h antes |
| `reschedule_appointment` | id ou especialidade, novo datetime | Atualiza item e schedule |
| `cancel_appointment` | id ou especialidade | Remove item e schedule |
| `register_medication` | nome, horário HH:MM (padrão 13:00), data de fim? | Grava remédio e garante o schedule do horário |
| `update_medication` | nome, novo horário | Move o remédio de horário |
| `stop_medication` | nome | Desativa o remédio |
| `list_reminders` | — | Devolve consultas futuras e remédios ativos |
| pause\_water\_reminders | — | Pausa os lembretes de água e cancela os do dia |
| resume\_water\_reminders | — | Retoma os lembretes de água a partir do próximo planejamento |

**Guardrails**

- Bedrock Guardrails com filtro de tópicos para conselhos médicos e dosagens.
- Validação de entrada das ferramentas com schema (datas no futuro, horário válido).
- Limite de 1.024 tokens de saída e de 20 mensagens de histórico por chamada.
