# ADR 0004: Uma task por serviço em Fargate Spot ARM, prod desligado de madrugada

- Data: 2026-10-25
- Status: aceita

## Contexto
A estimativa com preços de lista em `us-east-1` mostrou que 5 serviços sempre ligados em Fargate on-demand custariam ~US$ 64/mês, mais de duas vezes o orçamento de R$150 (~US$ 27). Os maiores custos por task são a computação e o IPv4 público (~US$ 3,65/mês por task, sem desconto).

## Decisão
1. **Cada microserviço continua na sua própria task**, com deploy independente. O administrador prefere manter o desenho de microserviços a juntar containers numa task só.
2. **Fargate Spot nos dois ambientes**, com imagens `linux/arm64` (Graviton). A AWS suporta Spot com ARM desde 2024.
3. **Sem Lambda:** a stack continua sendo ECS.
4. **Prod desligado das 00:00 às 05:50** (America/Sao_Paulo). Dois schedules do EventBridge Scheduler zeram e restauram o `desiredCount` dos serviços ECS. Religa às 05:50 para estar no ar antes do planejamento da água das 06:00.
5. **Lembrete atrasado é descartado:** um `ReminderDue` que chega com mais de `REMINDER_MAX_DELAY` (15 min) de atraso não gera mensagem e é registrado em log. Isso cobre a janela noturna e longas interrupções do Spot.
6. **Mensagens da madrugada:** a cliente pode escrever entre 00:00 e 05:50. O API Gateway responde erro, a Meta reenvia, e a resposta e o relay saem depois que prod religa. O administrador aceitou esse comportamento.
7. **Risco de custo aceito:** a estimativa fica em ~US$ 32/mês (~R$ 175), um pouco acima do orçamento. O budget alarm de 80% vai disparar; isso é esperado.

## Alternativas consideradas
- **Uma task por ambiente com os 5 containers:** ~US$ 12-18/mês, mas perde o deploy independente por serviço.
- **Subir o orçamento e manter on-demand 24h:** ~R$ 350.
- **Consumidores em Lambda:** mais barato, mas muda a stack de estudo.
- **Gateway em Fargate on-demand e o resto em Spot:** ~US$ 4/mês a mais, protege a entrada contra falta de capacidade Spot. Fica como opção se o Spot causar problema.

## Consequências
- **Interrupção do Spot:** uma interrupção dá 2 minutos de aviso, e o ECS sobe outra task. Os consumidores retomam da fila SQS. No gateway, a Meta reenvia o webhook, e a idempotência por `wamid` evita duplicidade.
- **Lembretes entre 00:00 e 05:50:** consultas ou remédios nesse horário nunca recebem lembrete, pela regra do descarte. Remédio cadastrado nesse horário merece um aviso na confirmação; a regra exata de texto fica para a Fase 3.
- **Custo de investigação futura:** com Spot, o IPv4 vira o maior custo por task. Vale investigar tasks só com IPv6 (depende de ECR, Bedrock e Meta aceitarem IPv6).
- **ECR e OIDC na camada `infra/envs/shared/`** (decisão D2 da Fase 1), porque prod não pode depender de `staging/base`.
