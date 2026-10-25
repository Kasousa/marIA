# e2e/: suíte ponta a ponta de staging

Complementa o CLAUDE.md da raiz. Roda só na esteira (`deploy-staging`), nunca no `make test`.

- Todo arquivo tem a build tag `e2e` (`//go:build e2e`).
- Configuração por variável de ambiente: URL do webhook (API Gateway de staging), app secret, nomes de tabelas, filas e DLQs. Sem valores fixos no código.
- Os testes assinam o webhook com HMAC-SHA256 (`X-Hub-Signature-256`), como a Meta, simulando o número simulador.
- Cada cenário confere o efeito: itens no DynamoDB, schedules no EventBridge Scheduler e DLQs vazias ao final.
- Cenários usam ids únicos por execução e limpam o que criaram; nunca dependem da ordem de outro teste.
- Use os parâmetros encurtados de staging (APPOINTMENT_LEAD_TIME de 2 minutos, janela de água curta) e espere com polling e timeout, nunca com sleep fixo.
- Nada de telefone ou texto real da cliente; só dados do simulador.
