# marIA: especificação

Oct 24, 2026 · @Kaique

Esta é a fonte da verdade do produto. Cada arquivo cobre uma área; abra só o que a tarefa precisa.

| Arquivo | Conteúdo |
| --- | --- |
| [produto.md](produto.md) | Personalidade, tom, exemplos de mensagens e fluxos F1 a F5 |
| [arquitetura.md](arquitetura.md) | Serviços, comunicação, eventos, modelo de dados (DynamoDB) e agendamentos (EventBridge Scheduler) |
| [whatsapp.md](whatsapp.md) | WhatsApp Cloud API: números, janela de 24h, imagens, templates, webhook |
| [bedrock.md](bedrock.md) | Modelo, system prompt, ferramentas e guardrails |
| [infra.md](infra.md) | Stack, ambientes, Terraform, custos, liga/desliga do staging e segurança |
| [esteira.md](esteira.md) | Git Flow, proteção de branches, workflows e steps |

Decisões posteriores à spec ficam em [`docs/adr/`](../adr/README.md). O CLAUDE.md vigente fica na raiz do repositório e é a única cópia; o funcionamento do harness está em `HARNESS.md`.

## Visão geral

A marIA v1 é uma assistente 100% via WhatsApp que lembra a cliente de tomar remédios, de ir às consultas e de beber água, em tom leve, com humor e imagens do Snoopy. Ela também serve de projeto de estudo para o administrador praticar arquitetura de microserviços orientada a eventos, Go, Docker, ECS, Terraform, GitHub Actions e Bedrock com uma usuária real.

**No escopo da v1**

- Lembrete diário de remédios que a cliente pedir, por padrão às 13h.
- Registro de consultas informadas por mensagem (dia e hora) e lembrete 1h antes.
- Lembretes de água em horários aleatórios ao longo do dia, sempre com contexto do Snoopy.
- Conversa amena quando ela responder, sem exigir confirmação.
- Imagens e memes do Snoopy, escolhidos de um bucket S3 curado pelo administrador, enviados junto com parte das mensagens.
- Encaminhamento das conversas para o WhatsApp do administrador, com aviso por template quando a janela de 24h dele estiver fechada.
- Dois ambientes no ECS, staging e prod, com deploy pela esteira do GitHub Actions.

**Fora do escopo da v1**

| Item | Quando |
| --- | --- |
| Observabilidade com Datadog (APM, logs, dashboards, monitores) | v2 |
| Testes funcionais com Playwright na esteira | Futuro (o step já fica previsto) |
| OAuth 2 entre serviços | Backlog |
| EKS | Backlog |
| Histórico de exames, áudio e imagens enviadas pela cliente | Backlog |
| Front-end ou painel web | Backlog |

**Usuários**

| Papel | O que faz | Canal |
| --- | --- | --- |
| Cliente | Usuária principal em prod: recebe lembretes e conversa com a marIA | WhatsApp |
| Administrador | Recebe o encaminhamento das conversas de staging e prod, cura as imagens e opera a esteira | WhatsApp e GitHub |
| Número simulador | Faz o papel da cliente em staging | WhatsApp |

A cliente é informada pessoalmente pelo administrador de que ele acompanha as conversas. A marIA não toca no assunto, mas confirma se ela perguntar.

## Glossário

| Termo | Significado |
| --- | --- |
| Cliente | Usuária principal em prod, que recebe os lembretes |
| Administrador | Quem opera o projeto, recebe o relay e aprova deploys |
| Número simulador | WhatsApp que faz o papel da cliente em staging |
| `wamid` | Id de uma mensagem no WhatsApp; chave de idempotência das mensagens |
| `event_id` | Id de um evento no barramento `maria-events`; chave de idempotência dos eventos |
| Janela de 24h | Período depois da última mensagem do contato em que se pode mandar texto livre; fora dela, só template |
| Template | Mensagem pré-aprovada pela Meta, com variáveis, usada fora da janela de 24h |
| Mensagem proativa | Enviada pela marIA sem a cliente ter escrito antes (lembretes) |
| Relay | Encaminhamento das conversas para o WhatsApp do administrador |
| Aviso (notice) | Template `maria_nova_conversa` que avisa o administrador de conversa nova |
| Lead time | Antecedência do lembrete de consulta (`APPOINTMENT_LEAD_TIME`) |
| `ReminderDue` | Evento do EventBridge Scheduler que diz que um lembrete venceu |
| `staging-ok` | Tag da imagem no ECR e status do commit que marcam que aquele código passou em staging |
| Hash da árvore | `git rev-parse HEAD^{tree}`: identifica o conteúdo dos arquivos, usado como tag da imagem |
| Camada base / compute | As duas partes do Terraform do staging: a que fica sempre ligada (dados e entrada) e a que liga e desliga (ECS e schedules) |
| Back-merge | PR da `main` para a `develop` depois de uma release ou hotfix |

## Roteiro de implementação

A fundação vem primeiro (infra e esteira), depois as funcionalidades em fatias de ponta a ponta. Cada fase passa por staging antes de prod, e a cliente começa a usar ao fim da fase 4.

1. **Fundação.** Monorepo Go, Terraform de staging (base e compute) e prod, OIDC do GitHub, Git Flow com proteção de `develop` e `main`, workflows `ci`, `deploy-staging`, `deploy-prod` e `staging-power`, com o portão de testes unitários e o step do Playwright já previsto (desligado).
   - Pronto quando: um serviço `hello` com testes sobe em staging por um merge na `develop`, uma `release/0.1.0` validada em staging chega em prod pelo merge na `main` e volta para a `develop` pelo back-merge, e um teste quebrado de propósito bloqueia o deploy.
2. **Conversa básica.** gateway, conversation e outbound: a marIA responde mensagens no tom do Snoopy, com Bedrock e a regra da janela de 24h.
   - Pronto quando: em staging, o número simulador conversa com a marIA, e a suíte E2E cobre uma resposta livre.
3. **Lembretes de consulta e remédio.** reminders com API interna, DynamoDB e EventBridge Scheduler; ferramentas do Bedrock; templates aprovados.
   - Pronto quando: a suíte E2E cria consulta e remédio, e os lembretes chegam no horário (consulta com antecedência de 2 minutos em staging).
4. **Relay para o administrador e go-live.** relay com fila de pendentes, aviso com botão e prefixo de ambiente; primeira promoção para prod.
   - Pronto quando: o administrador recebe as conversas de staging e prod em chats separados, e a cliente recebe o primeiro lembrete real.
5. **Lembretes de água.** Planejamento diário com horários aleatórios, templates de água, pausa e retomada pela conversa.
   - Pronto quando: em staging, com janela encurtada, saem os disparos sorteados respeitando o intervalo mínimo, e a pausa cancela os pendentes do dia.
6. **Imagens do Snoopy.** Bucket S3, upload pela Media API, templates com cabeçalho de imagem, regra de não repetir em 14 dias.
   - Pronto quando: cerca de 1 em cada 3 mensagens proativas leva imagem, sem repetição em 14 dias.
7. **Endurecimento.** Testes de falha (Meta fora, Bedrock lento, mensagem duplicada) na suíte E2E e revisão de custos.
   - Pronto quando: cada cenário de falha termina sem mensagem perdida ou duplicada.

**v2**

- Observabilidade com Datadog: APM com traces atravessando as filas, logs, LLM Observability, dashboards, monitores e SLO.
- Testes funcionais com Playwright ativados na esteira.

**Backlog**

- OAuth 2 com Client Credentials entre serviços.
- Migração ou ambiente de estudo em EKS.
- Histórico e leitura de exames.
- Entender áudio e imagens enviados pela cliente.
- Painel web, se um dia fizer sentido.
- Domínio próprio para o webhook: domínio no registro.br, zona no Route 53, certificado no ACM e custom domain no API Gateway, para a URL não depender do id do API.
