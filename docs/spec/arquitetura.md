# Arquitetura

Serviços, eventos, tabelas e agendamentos. Leia antes de mexer em internal/, pkg/ ou em contratos de eventos.

## Arquitetura

A marIA usa microserviços orientados a eventos (event-driven), no estilo coreografia: cada serviço publica fatos num barramento e reage aos fatos que interessam a ele, sem um orquestrador central. Só o gateway recebe da Meta e só o outbound envia para ela, o que concentra as regras do WhatsApp em dois pontos.

&#91;embedded content: arquitetura da marIA v1 · 5 serviços, 1 barramento de eventos\]

**Como os serviços se comunicam**

- **Assíncrono (padrão):** SNS `maria-events` faz o fan-out e cada serviço consumidor tem sua fila SQS com DLQ. Isso dá desacoplamento, retry automático e absorção de falhas da Meta ou do Bedrock sem perder mensagem.
- **Síncrono (exceção):** o conversation chama a API HTTP interna do reminders quando o Bedrock escolhe uma ferramenta, porque precisa da resposta para confirmar à cliente. A chamada passa pelo ECS Service Connect, só dentro da VPC.
- **Eventos são fatos no passado** (`MessageReceived`, `MessageSent`), com exceção de `SendMessage`, que é um comando para o outbound.

| Serviço | Responsabilidade | Publica | Consome |
| --- | --- | --- | --- |
| whatsapp-gateway | Recebe webhook, valida assinatura, filtra allowlist | `MessageReceived` | — |
| conversation | Personalidade, Bedrock, tool use, histórico | `SendMessage` | `MessageReceived` |
| reminders | Consultas, remédios, agendamentos | `SendMessage` | `ReminderDue` (Scheduler) |
| outbound | Envio à Meta, template ou texto livre, imagens do S3, retry | `MessageSent` | `SendMessage` |
| relay | Fila de pendentes e aviso ao administrador | `SendMessage` | `MessageReceived`, `MessageSent` |

## Modelo de dados (DynamoDB)

Cada serviço é dono da sua tabela, em modo on-demand, com uma cópia por ambiente (sufixo `-staging` ou `-prod`). Nenhum serviço lê a tabela de outro: a comunicação é por eventos ou pela API interna.

| Tabela | Dono | PK | SK | Atributos principais |
| --- | --- | --- | --- | --- |
| `maria-users` | conversation | `USER#<id>` | `PROFILE` | nome, telefone (E.164), papel (`primary` ou `admin`), fuso |
| `maria-conversations` | conversation | `USER#<id>` | `MSG#<iso8601>#<wamid>` | direção (`in` ou `out`), tipo, texto, `ttl` de 90 dias |
| `maria-reminders` | reminders | `USER#<id>` | `APPT#<iso8601>` | especialidade, local, texto original, `schedule_name`, status |
| `maria-reminders` | reminders | `USER#<id>` | `MED#<slug>` | nome, horário (HH:MM), data de fim, ativo |
| `maria-outbound` | outbound | `CONTACT#<telefone>` | `WINDOW` | `last_inbound_at`, para decidir entre template e texto livre |
| `maria-outbound` | outbound | `MEDIA` | `SENT#<s3_key>` | `last_sent_at`, para não repetir imagem em 14 dias |
| `maria-relay` | relay | `ADMIN#<id>` | `PENDING#<iso8601>#<wamid>` | direção, texto, status (`pending` ou `delivered`) |
| `maria-relay` | relay | `ADMIN#<id>` | `NOTICE` | `notice_pending` (bool), `notice_sent_at` |
| `maria-idempotency` | todos | `<servico>#<message_id>` | — | `processed_at`, `ttl` de 7 dias |
| maria-reminders | reminders | USER#\<id> | WATER | ativo (pausado ou não), quantidade por dia, janela, intervalo mínimo |
| maria-reminders | reminders | USER#\<id> | WATER#\<data>#\<hora> | horário sorteado do dia, schedule\_name, ttl de 2 dias |

**Decisões**

- Datas sempre em ISO 8601 com offset de São Paulo (`2026-11-12T15:00-03:00`).
- O `wamid` (id da mensagem no WhatsApp) é a chave de idempotência: a Meta pode entregar o mesmo webhook mais de uma vez.
- Conversas expiram em 90 dias via TTL, por serem dados pessoais de saúde.
- Criptografia com chave KMS gerenciada pelo cliente em todas as tabelas e no bucket de imagens.

## Agendamentos (EventBridge Scheduler)

Todos os lembretes usam EventBridge Scheduler com fuso `America/Sao_Paulo`, num schedule group `maria-reminders`. O alvo de todos é a fila SQS `reminder-due`, consumida pelo reminders-service.

| Tipo | Expressão | Nome | Após disparar |
| --- | --- | --- | --- |
| Consulta | `at(2026-11-12T14:00:00)` (1h antes) | `appt-<user>-<epoch>` | Apaga sozinho (`ActionAfterCompletion: DELETE`) |
| Remédio | `cron(0 13 * * ? *)` | `med-<user>-<horario>` | Recorrente; `EndDate` se houver prazo |
| Planejamento da água | cron(0 6 \* \* ? \*) | water-plan-\<user> | Recorrente; sorteia os horários do dia |
| Lembrete de água | at(\<horário sorteado>) | water-\<user>-\<epoch> | Apaga sozinho (ActionAfterCompletion: DELETE) |

**Regras**

- Um schedule de remédio por horário, não por remédio: o payload aponta para o horário, e o serviço busca no DynamoDB quais remédios ativos caem nele. Assim, remédios no mesmo horário viram uma mensagem só.
- Payload do evento: `{"type": "medication" | "appointment", "user_id": "...", "ref": "MED-13:00" | "APPT#..."}`.
- Ao receber `ReminderDue`, o reminders-service confere no DynamoDB se o item ainda está ativo. Se foi cancelado, descarta.
- Um `ReminderDue` que chega com mais de `REMINDER_MAX_DELAY` (padrão 15 min) de atraso em relação ao horário agendado é descartado e registrado em log, sem mensagem. Isso cobre a janela em que prod fica desligado (00:00 às 05:50) e longas interrupções do Fargate Spot (ADR 0004).
- Retry do Scheduler: 3 tentativas, com DLQ `reminder-due-dlq` e alarme quando ela tiver mensagens.

A antecedência do lembrete de consulta e o horário padrão dos remédios ficam em variáveis de ambiente (`APPOINTMENT_LEAD_TIME`, `MEDICATION_DEFAULT_TIME`). Em prod valem 1h e 13:00; em staging a antecedência pode cair para 2 minutos, para testar o fluxo inteiro sem esperar.

Os lembretes de água usam `WATER_REMINDERS_PER_DAY` (padrão 3), `WATER_WINDOW` (padrão 09:00-21:00) e `WATER_MIN_GAP` (padrão 2h). Em staging, a janela e o intervalo podem ser encurtados para testar vários disparos em poucos minutos. O payload do evento ganha o tipo `"water"`; se a água estiver pausada quando o evento chegar, ele é descartado.
