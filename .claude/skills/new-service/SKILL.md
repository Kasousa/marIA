---
name: new-service
description: Cria o esqueleto de um microserviço da marIA (gateway, conversation, reminders, outbound, relay) seguindo as convenções do monorepo - cmd, internal, config por env, slog mascarado, healthz, consumidor SQS idempotente, testes e módulo ecs-service no Terraform. Use quando for começar um serviço novo.
---

# Novo serviço

Entrada: nome do serviço (um de `gateway`, `conversation`, `reminders`, `outbound`, `relay`). Leia na `docs/spec.md` a linha dele na tabela de serviços (publica / consome) e a tabela DynamoDB de que ele é dono.

Crie, nesta ordem, com testes junto:

1. `internal/<svc>/config.go`: struct `Config` carregada de variáveis de ambiente (`ENV`, `AWS_REGION`, nomes de fila, tabela, tópico e parâmetros da spec). Falha no boot se faltar variável obrigatória. Teste table-driven do carregamento.
2. `internal/<svc>/service.go`: a lógica do serviço, dependendo só de interfaces pequenas (repositório, publisher, clock). Nada de SDK da AWS aqui.
3. `internal/<svc>/aws.go`: adaptadores reais (DynamoDB, SNS, SQS...) que implementam as interfaces.
4. Se consumir eventos: `internal/<svc>/consumer.go` usando `pkg/events` para decodificar e `pkg/idempotency` antes de qualquer efeito colateral. Mensagem que falha volta para a fila (a DLQ cuida do resto); nunca engula erro.
5. `cmd/<svc>/main.go`: só monta dependências, sobe `GET /healthz` em chi e o consumidor, e trata SIGTERM com shutdown gracioso (o ECS manda SIGTERM no deploy).
6. Logs com `log/slog` JSON via `pkg/logging`; textos e telefones sempre com `logging.Mask()`.
7. Terraform: instância do módulo `ecs-service` em `infra/envs/staging/compute` e `infra/envs/prod`, task role com acesso só à própria tabela e filas, fila SQS + DLQ + assinatura no SNS em `staging/base` e `prod`.
8. Rode `make check` e `make tf-check`. Peça revisão ao subagente `architecture-guardian` e, se tocou infra, rode `/security-review` conferindo a seção "Infra" do CLAUDE.md.
9. Atualize a feature correspondente em `docs/features.json` para `done_local` com a evidência.

Não copie código de outro serviço por importação de `internal/<outro>`: se algo é compartilhado, vai para `pkg/`.
