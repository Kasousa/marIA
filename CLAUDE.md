# marIA

Assistente 100% via WhatsApp que lembra a cliente de remédios, consultas e de
beber água, em tom leve com humor e imagens do Snoopy. Projeto de estudo de
microserviços orientados a eventos na AWS, operado por um administrador.
A especificação está em docs/spec/ (um arquivo por área; índice e glossário no README.md): leia a parte relevante antes de mudanças grandes.

## Stack
- Go (versão estável mais recente), net/http + chi, aws-sdk-go-v2, log/slog
- Docker multi-stage (imagem final distroless), ECS Fargate
- Terraform (infra/); deploy só pela esteira do GitHub Actions (OIDC, sem chaves)
- AWS (us-east-1, uma conta para staging e prod; ADR 0003): ECS, API Gateway HTTP API + VPC Link + Cloud Map, DynamoDB, SNS/SQS, EventBridge Scheduler, S3, Bedrock (Claude Haiku 4.5), Secrets Manager, CloudWatch Logs
- Sem ambiente local de integração: testes unitários com fakes localmente, integração em staging
- Não existe front-end. Não existe OAuth 2 nem Datadog na v1 (Datadog vem na v2).

## Estrutura
- cmd/<servico>/main.go : gateway, conversation, reminders, outbound, relay
- internal/<servico>/ : código de cada serviço
- pkg/ : eventos, idempotência, logging mascarado, cliente da Meta
- e2e/ : suíte de testes ponta a ponta para staging
- infra/modules/ e infra/envs/{staging/base,staging/compute,prod}/
- .github/workflows/: ci, deploy-staging, deploy-prod, staging-power

## Regras de arquitetura
- Microserviços orientados a eventos: SNS `maria-events` + uma fila SQS com DLQ por consumidor.
- Única chamada síncrona: conversation -> API interna do reminders (ECS Service Connect).
- Cada serviço é dono da sua tabela DynamoDB; nunca leia a tabela de outro serviço.
- Todo consumidor é idempotente pelo `wamid` ou `event_id` (tabela maria-idempotency).
- Só o outbound chama a API da Meta para enviar; ele decide template x texto livre pela janela de 24h e anexa imagens do S3.
- Fuso sempre America/Sao_Paulo; datas em ISO 8601 com offset.
- O Bedrock nunca grava direto: ele chama ferramentas, o serviço valida e executa.
- Configuração por variáveis de ambiente; a mesma imagem roda em staging e prod.

## Regras de produto
- Nunca exigir confirmação da cliente, nunca insistir, nunca dar conselho médico.
- Consulta: lembrete APPOINTMENT_LEAD_TIME antes (1h em prod); se faltar menos que isso, sem lembrete.
- Remédio: MEDICATION_DEFAULT_TIME (13:00), diário, remédios do mesmo horário numa mensagem só.
- Água: planejamento diário às 06:00 sorteia WATER_REMINDERS_PER_DAY (3) horários em WATER_WINDOW (09:00-21:00) com WATER_MIN_GAP (2h); pula se cair a menos de 30 min de outro lembrete; pausável pela conversa.
- Imagem do Snoopy em ~1/3 das mensagens proativas, sem repetir em 14 dias.
- Relay: no máximo um aviso pendente para o administrador; pendentes entregues em ordem, com prefixo [STAGING] ou [PROD].
- Só números na allowlist do ambiente são atendidos.
- Prod desligado das 00:00 às 05:50; ReminderDue com mais de REMINDER_MAX_DELAY (15 min) de atraso é descartado (ADR 0004).

## Segurança e logs
- Nunca logar texto de mensagem ou telefone em claro: use pkg/logging.Mask().
- Segredos só via Secrets Manager; nada de .env commitado.
- Validar X-Hub-Signature-256 em todo webhook.

## Testes
- Dependências externas (AWS, Bedrock, Meta) entram por interfaces pequenas; localmente só fakes.
- Relógio e aleatoriedade sempre injetados; datas fixas com offset -03:00 nos testes.
- Cubra sempre: mensagem duplicada, janela de 24h aberta e fechada, item cancelado quando o ReminderDue chega, entrada inválida das ferramentas do Bedrock.
- Nunca use telefone ou texto real de cliente em fixtures.

## Regras por pasta
- infra/CLAUDE.md (Terraform: IAM, OIDC, camadas do staging, custo), .github/CLAUDE.md (workflows) e e2e/CLAUDE.md (suíte de staging) são carregados quando você trabalha nessas pastas.

## Git Flow e esteira (ADR 0002)
- `main` = prod, `develop` = integração e staging.
- Todo trabalho de código começa numa `feature/*` criada da develop e volta por PR para a develop.
- Versão: `release/x.y.z` sai da develop; correção urgente: `hotfix/x.y.z` sai da main. As duas entram na main por PR e voltam para a develop por PR de back-merge.
- Nunca commit, merge ou push direto em develop ou main; o merge de PR é sempre do administrador.
- `go test ./... -race` é portão obrigatório: se falhar, a esteira para e o PR não pode ser mesclado.
- deploy-staging (push na develop, release/* ou hotfix/*): lint -> testes unitários -> build/push ECR (tag = hash da árvore do Git) -> liga staging -> deploy -> E2E -> (Playwright, futuro) -> tag e status staging-ok.
- deploy-prod (push na main): testes unitários -> exige imagem com staging-ok pelo hash da árvore -> aprovação no environment production -> deploy sem rebuild -> smoke test -> tag vx.y.z.
- Push em release/* ou hotfix/* deploya em staging: o agente só faz push com autorização explícita do administrador.
- Nunca aplicar Terraform ou publicar imagem fora da esteira.

## Harness (leia HARNESS.md)
- Estado do roteiro: docs/features.json (status só avança com evidência).
- Memória entre sessões: docs/progress.md; planos em docs/plans/active/; decisões em docs/adr/.
- Comece trabalho novo com /plan-phase e termine a sessão com /handoff.
- `make check` = fmt, vet, golangci-lint, scripts/archcheck.sh, `make harness-test` e testes com -race.
- Mudou um hook ou o archcheck: acrescente o caso em scripts/test-harness.sh (ou em scripts/testdata/) no mesmo PR.
- Revisores do projeto (.claude/agents/): architecture-guardian (código) e persona-reviewer (texto para a cliente). Para o resto, /code-review e /security-review. Manutenção dos docs: /garden.
- Hooks bloqueiam terraform apply/destroy, docker push, AWS CLI que altera recursos, commit/merge em main/develop, push fora de feature/*, release/x.y.z e hotfix/x.y.z, e gh pr merge.
- Quando errar por falta de regra, proponha a regra (doc, lint, archcheck ou hook), não só a correção.

## Como trabalhar
- Antes de codar, diga o plano em poucas linhas e quais arquivos vai tocar.
- Escreva testes (go test, table-driven) junto com o código; rode `make test` e `make lint` antes de concluir.
- Mudanças de infra: rode `terraform fmt` e `terraform validate`, e mostre o `plan`.
- Prefira mudanças pequenas e incrementais, seguindo a fase atual do roteiro.
- Código e comentários em inglês; mensagens para a cliente em português do Brasil.
