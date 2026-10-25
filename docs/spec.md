# marIA — Especificação Técnica

Oct 24, 2026 · @Kaique

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

## Personalidade e tom

A marIA soa como uma amiga carinhosa e bem-humorada, nunca como um alarme ou um sistema médico. O Snoopy aparece como tempero das mensagens, no texto e nas imagens.

**Regras de tom**

- Sempre em português do Brasil, frases curtas, no máximo 3 por mensagem.
- Uma referência ao Snoopy ou à turma (Woodstock, Charlie Brown, a casinha vermelha) por mensagem proativa, variando para não repetir.
- Nunca cobra, nunca culpa, nunca insiste. Se a cliente não responder, tudo bem.
- Respostas a "tomei", "bebi", "obrigada" ou similares: curtas, calorosas, com um toque de humor.
- Não dá conselhos médicos, não comenta dosagens e não interpreta sintomas. Em dúvida de saúde, sugere falar com o médico.
- Emojis com moderação: no máximo um por mensagem.

**Imagens do Snoopy**

- O administrador sobe as imagens e memes num bucket S3 por ambiente (`maria-media-<env>`), no prefixo `snoopy/`.
- O outbound-service anexa uma imagem a cerca de 1 em cada 3 mensagens proativas, escolhida ao acaso entre as que não foram enviadas nos últimos 14 dias. A frequência fica em configuração.
- Respostas curtas ("tomei!") também podem levar imagem, com a mesma regra de não repetir.
- Formatos aceitos pelo WhatsApp: JPEG ou PNG até 5 MB. Uma imagem fora do padrão é ignorada e registrada em log.

**Exemplos de mensagens**

| Situação | Exemplo |
| --- | --- |
| Lembrete de remédio | Oi! Hora da vitamina D. O Snoopy já tomou a dele e voltou a dormir em cima da casinha. |
| Lembrete de consulta | Daqui a 1 hora tem dermatologista às 15h. O Woodstock já está esperando no carro. |
| Lembrete de água | O Snoopy acabou de encher a tigela de água dele. Bora acompanhar com um copo? |
| Lembrete de água | Até o Woodstock fez uma pausa pra beber água. Sua vez! |
| Resposta a "tomei!" | Arrasou! O Snoopy fez a dança da felicidade aqui. |
| Consulta registrada | Anotado: dermatologista dia 12 às 15h. Te aviso 1 hora antes. |
| Falta a hora da consulta | Que horas vai ser? Quero te avisar no momento certo. |

## Fluxos funcionais

São cinco fluxos: registrar consulta, cadastrar remédio, lembrar de beber água, responder mensagens livres e encaminhar tudo para o administrador. Nenhum deles depende de confirmação da cliente.

### F1. Registrar consulta

1. A cliente escreve algo como "marquei o dermato dia 12 às 15h".
2. O conversation-service chama o Bedrock com a ferramenta `register_appointment`, que extrai especialidade, data, hora e local (opcional).
3. Se faltar data ou hora, a marIA pergunta. Datas relativas ("quinta que vem") são resolvidas no fuso America/Sao\_Paulo.
4. O conversation-service chama a API interna do reminders-service, que grava o item no DynamoDB.
5. Se a consulta for daqui a mais de 1h, cria um agendamento único para 1h antes. Se for daqui a menos de 1h, só confirma, sem lembrete.
6. A marIA confirma o registro.

Remarcação e cancelamento usam as ferramentas `reschedule_appointment` e `cancel_appointment`, que atualizam o item e o agendamento.

### F2. Cadastrar remédio

1. A cliente pede "me lembra de tomar a vitamina D".
2. O Bedrock chama `register_medication` com nome, horário (padrão 13h) e prazo (opcional, como "por 10 dias").
3. O reminders-service grava o item e garante um agendamento recorrente diário para aquele horário. Com prazo, o remédio recebe data de fim.
4. Remédios no mesmo horário saem juntos numa mensagem só.
5. Ela pode pedir para parar (`stop_medication`) ou mudar o horário (`update_medication`).

### F3. Lembrar de beber água

1. Todo dia às 06:00, um agendamento recorrente dispara o planejamento do dia no reminders-service.
2. O serviço sorteia os horários do dia: por padrão 3 lembretes entre 09:00 e 21:00, com pelo menos 2h entre um e outro. Quantidade, janela e intervalo mínimo ficam em configuração.
3. Para cada horário sorteado, cria um agendamento único que se apaga sozinho depois de disparar.
4. No horário, a marIA envia uma mensagem curta com contexto do Snoopy, sorteada entre as variações disponíveis, e às vezes com imagem.
5. Se um lembrete de água cair a menos de 30 minutos de outro lembrete (remédio ou consulta), ele é pulado naquele dia para não acumular mensagens.
6. A cliente pode pausar ("para de me lembrar de água") ou retomar; o Bedrock chama `pause_water_reminders` ou `resume_water_reminders`.

Os lembretes de água começam ativos para a cliente e nunca esperam resposta.

### F4. Mensagens livres e respostas

- Qualquer outra mensagem ("tomei!", "bebi", "obrigada", um papo qualquer) recebe resposta curta no tom da marIA, às vezes com imagem do Snoopy.
- Nenhuma resposta altera agendamentos.
- O histórico recente da conversa (últimas 20 mensagens) vai como contexto para o Bedrock.
- Áudio ou imagem enviados por ela recebem uma resposta carinhosa dizendo que, por enquanto, a marIA só entende texto.

### F5. Encaminhamento para o administrador

1. Toda mensagem enviada ou recebida no número da cliente gera um evento.
2. O relay-service grava a mensagem como pendente para o administrador.
3. Se a janela de 24h do administrador estiver aberta, encaminha na hora como texto livre.
4. Se estiver fechada e não houver aviso pendente, envia o template `maria_nova_conversa` com o botão "Ver agora".
5. Quando o administrador toca no botão, a janela abre e o relay envia todas as pendentes em ordem, marcando como entregues.
6. Enquanto houver aviso não aberto, novas mensagens só entram na fila: um template por lote, não por mensagem.

O administrador recebe o encaminhamento dos dois ambientes. Cada ambiente tem seu próprio número da marIA, então as conversas chegam em chats separados, e cada encaminhamento começa com o ambiente: `[PROD 13:00] marIA → cliente: texto` ou `[STAGING 13:07] simulador → marIA: texto`.

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
- Retry do Scheduler: 3 tentativas, com DLQ `reminder-due-dlq` e alarme quando ela tiver mensagens.

A antecedência do lembrete de consulta e o horário padrão dos remédios ficam em variáveis de ambiente (`APPOINTMENT_LEAD_TIME`, `MEDICATION_DEFAULT_TIME`). Em prod valem 1h e 13:00; em staging a antecedência pode cair para 2 minutos, para testar o fluxo inteiro sem esperar.

Os lembretes de água usam `WATER_REMINDERS_PER_DAY` (padrão 3), `WATER_WINDOW` (padrão 09:00-21:00) e `WATER_MIN_GAP` (padrão 2h). Em staging, a janela e o intervalo podem ser encurtados para testar vários disparos em poucos minutos. O payload do evento ganha o tipo `"water"`; se a água estiver pausada quando o evento chegar, ele é descartado.

## WhatsApp Cloud API

A marIA usa a WhatsApp Cloud API oficial da Meta, com um número de marIA por ambiente. A regra que mais molda o desenho é a janela de 24h: só dá para mandar texto livre a quem escreveu nas últimas 24h; fora disso, apenas templates aprovados, cobrados por mensagem.

**Números por ambiente**

| Ambiente | Número da marIA | Usuária principal | Relay |
| --- | --- | --- | --- |
| staging | Número de teste do app da Meta (ou um chip dedicado) | Número simulador | Administrador |
| prod | Número oficial, verificado | Cliente | Administrador |

A allowlist de cada ambiente fica no Secrets Manager. Qualquer número fora dela é ignorado.

**Consequências no desenho**

- Lembretes para a cliente quase sempre caem fora da janela, então são templates com variáveis.
- Respostas a mensagens dela são texto livre, porque a janela acabou de abrir.
- O outbound-service decide sozinho entre template e texto livre, com base em `last_inbound_at` de cada contato.
- O aviso para o administrador usa o template com botão; o toque no botão abre a janela dele.

**Imagens**

- Dentro da janela: mensagem do tipo imagem com legenda, usando o texto da marIA como legenda.
- Fora da janela: templates com cabeçalho de imagem (header de mídia), uma variante com imagem para cada template de lembrete.
- A imagem é enviada para a Meta pela Media API (upload que devolve um `media_id`), sem expor o bucket publicamente.

**Templates a cadastrar (categoria Utility, pt\_BR)**

| Nome | Para | Texto | Variáveis |
| --- | --- | --- | --- |
| `maria_remedio_1` a `maria_remedio_7` | Cliente | Uma variação por dia da semana, ex.: "Oi! Hora do {{1}}. O Snoopy já tomou o dele e voltou pra casinha." | {{1}} = remédio(s) |
| `maria_remedio_img_1` a `_7` | Cliente | Mesmos textos, com cabeçalho de imagem | imagem + {{1}} |
| `maria_consulta_1` a `maria_consulta_3` | Cliente | Ex.: "Daqui a 1 hora tem {{1}} às {{2}}. O Woodstock já está te esperando." | {{1}} = especialidade, {{2}} = hora |
| `maria_consulta_img_1` a `_3` | Cliente | Mesmos textos, com cabeçalho de imagem | imagem + {{1}}, {{2}} |
| `maria_agua_1` a `maria_agua_10` | Cliente | Variações curtas com o Snoopy e a turma, ex.: "O Snoopy acabou de encher a tigela de água dele. Bora acompanhar com um copo?" | nenhuma |
| `maria_agua_img_1` a `_10` | Cliente | Mesmos textos, com cabeçalho de imagem | imagem |
| `maria_nova_conversa` | Administrador | "Teve conversa nova com a marIA. Quer ver?" + botão de resposta rápida "Ver agora" | nenhuma |

Os templates precisam ser aprovados nas duas contas (staging e prod). Dentro da janela de 24h, os lembretes de água saem como texto livre gerado com as mesmas regras de tom, o que aumenta a variedade.

**Outros cuidados**

- Validar a assinatura `X-Hub-Signature-256` de todo webhook antes de processar.
- Responder 200 ao webhook em menos de 5s e processar de forma assíncrona (fila).
- Rate limit e retry com backoff exponencial no outbound-service.

## Bedrock: modelo, prompt e ferramentas

O conversation-service usa Claude Haiku 4.5 no Bedrock, pela API Converse com tool use. O modelo nunca grava nada direto: ele escolhe uma ferramenta, e o serviço valida e chama o reminders-service.

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

## Infraestrutura e ambientes

Tudo roda em containers Docker no ECS Fargate, provisionado com Terraform e implantado pela esteira do GitHub Actions, em dois ambientes idênticos: staging e prod. A mesma imagem passa pelos dois, e só a configuração muda.

**Stack**

- Serviços em Go (versão estável mais recente), com `net/http` + `chi`, `aws-sdk-go-v2` e `log/slog`.
- Imagens Docker multi-stage, finais em `distroless/static`, publicadas no ECR com a tag do commit.
- Mensageria: SNS (tópico `maria-events`) com filas SQS por consumidor, cada uma com DLQ.
- Segredos (token da Meta, app secret, allowlist) no Secrets Manager.
- Logs básicos em JSON no CloudWatch Logs, com retenção de 14 dias e sem texto de mensagem em claro. Observabilidade completa com Datadog fica para a v2.
- Sem ambiente local de integração: localmente só rodam testes unitários com fakes (`go test`); toda integração é testada em staging.

**Ambientes**

|  | staging | prod |
| --- | --- | --- |
| Orquestração | ECS Fargate, 1 task por serviço | ECS Fargate, 1 task por serviço, deployment circuit breaker com rollback |
| Tamanho das tasks | 0,25 vCPU / 0,5 GB, Fargate Spot | 0,25 vCPU / 0,5 GB |
| Número da marIA | Teste | Oficial |
| Usuária | Número simulador | Cliente |
| Disponibilidade | Liga/desliga, desligado por padrão | Sempre ligado |
| Rede | Subnets públicas com security groups restritos, sem NAT Gateway | Igual a staging |

**Estrutura do Terraform**

- `modules/`: `network`, `ecs-cluster`, `ecs-service`, `messaging`, `dynamodb`, `scheduler`, `media-bucket`, `github-oidc`.
- `envs/staging/base`, `envs/staging/compute` e `envs/prod`, cada um com state remoto próprio no S3 (lock nativo do S3).
- O módulo `github-oidc` cria o provedor OIDC e uma role por ambiente, para a esteira acessar a AWS sem chaves de acesso.

**Custos**

- Orçamento de R$150/mês para os dois ambientes juntos. Staging fica desligado por padrão e só é ligado para testar.
- Budget alarm da AWS em 80% e 100% desde o primeiro dia.
- Valores de Fargate, ALB e WhatsApp a confirmar nas páginas de preço antes do primeiro deploy.

## Liga/desliga do staging

O staging fica desligado por padrão e o administrador liga só quando vai testar. Para isso, o Terraform do staging é dividido em duas camadas: a base fica sempre de pé e guarda os dados; a compute é criada e destruída pelo liga/desliga.

| Camada | O que tem | Quando o staging está desligado |
| --- | --- | --- |
| `staging/base` (sempre ligada) | DynamoDB, SNS/SQS, bucket S3, Secrets Manager, KMS, repositórios ECR | Continua existindo; custo mínimo (on-demand sem tráfego, mais o valor fixo de segredos e chaves) |
| `staging/compute` (liga/desliga) | ALB, serviços ECS, schedules do EventBridge | Destruída: sem custo de ALB nem de Fargate |

**Como operar**

- Um workflow do GitHub Actions, `staging-power`, com acionamento manual e opção `on` ou `off`. Dá para disparar pelo app do GitHub no celular.
- `on` aplica a camada compute e, ao terminar, roda um health check dos serviços. `off` destrói a camada compute.
- Desligamento automático todo dia às 23h, para o staging não ficar ligado por esquecimento.
- A esteira de deploy liga o staging sozinha quando precisa testar uma nova versão.

**Enquanto desligado**

- Mensagens enviadas ao número de staging da marIA não são processadas. É aceitável, porque só o número simulador e o administrador falam com ele.
- Nenhum lembrete de staging dispara, porque os schedules fazem parte da camada compute. Ao religar, o reminders-service recria os schedules a partir dos itens ativos no DynamoDB e refaz o planejamento de água do dia.
- Dados de teste (consultas, remédios, conversas) ficam preservados na camada base.

## Esteira de CI/CD (GitHub Actions + Git Flow)

O repositório segue o Git Flow padrão: a branch `develop` integra o trabalho e entrega em staging, e a branch `main` contém só o que está em prod. Todo deploy na AWS passa pelo GitHub Actions; ninguém aplica Terraform nem publica imagem da própria máquina. A esteira para no primeiro step que falhar. A decisão está registrada no ADR 0002.

**Branches**

| Branch | Sai de | Entra em | O que acontece |
| --- | --- | --- | --- |
| `feature/*` | `develop` | `develop` (PR) | Workflow `ci` no PR; o merge dispara o deploy em staging |
| `develop` | — | — | Cada push faz deploy em staging |
| `release/x.y.z` | `develop` | `main` (PR) e depois `develop` (back-merge) | Congela o escopo da versão; cada push faz deploy em staging; só entram correções, versão e changelog |
| `hotfix/x.y.z` | `main` | `main` (PR) e depois `develop` (back-merge) | Correção urgente; cada push faz deploy em staging antes de ir para prod |
| `main` | — | — | Cada push faz deploy em prod e cria a tag `vx.y.z` |

**Ciclo de uma versão**

1. Features entram na `develop` por PR, com o `ci` verde. Cada merge vai para staging.
2. Quando o escopo da versão está pronto, o administrador cria `release/x.y.z` a partir da `develop`. A esteira faz o deploy da release em staging e roda o E2E. Correções encontradas em staging são commitadas na própria release.
3. O administrador abre o PR da `release/x.y.z` para a `main`. Com o `ci` verde e o status `staging-ok` no último commit da release, faz o merge (merge commit, sem fast-forward).
4. O merge na `main` dispara o `deploy-prod`, que pede aprovação no environment `production`, entrega a mesma imagem validada em staging e cria a tag `vx.y.z`, com o número tirado do nome da branch.
5. **Back-merge:** o administrador abre o PR da `main` para a `develop`, para que as correções feitas na release ou na hotfix voltem para a `develop`. A release ou hotfix é apagada depois disso.

Uma hotfix segue os passos 2 a 5, mas sai da `main` em vez da `develop`.

**Proteção das branches**

- `develop` e `main` não aceitam push direto: só por pull request, com o workflow `ci` verde.
- A `main` só aceita PR vindo de `release/*` ou `hotfix/*` com nome no formato `x.y.z`. Um check do `ci` valida a branch de origem, o status `staging-ok` no commit de origem e se a branch de origem contém a `main` atual (`git merge-base --is-ancestor`).
- Merge na `main` sem fast-forward (merge commit), como manda o Git Flow.
- Como o projeto tem um único administrador e o GitHub não deixa ninguém aprovar o próprio PR, os PRs não exigem aprovação de revisor. A aprovação humana obrigatória fica no environment `production`, antes do deploy em prod.

**Por que o hash da árvore encontra a imagem certa:** a branch de origem contém a `main` atual (o `ci` verifica). Então o merge commit na `main` tem exatamente a mesma árvore de arquivos do último commit da release ou hotfix, e esse commit foi o que passou em staging.

**Workflows**

| Workflow | Quando roda | O que faz |
| --- | --- | --- |
| `ci` | Todo PR para `develop` ou `main` | Lint, testes unitários (portão), build da imagem e `terraform plan` do ambiente de destino; nos PRs para a `main`, também as checagens de origem |
| `deploy-staging` | Push na `develop`, em `release/**` ou em `hotfix/**` | Esteira de staging |
| `deploy-prod` | Push na `main` | Esteira de prod |
| `staging-power` | Manual e todo dia às 23h | Liga ou desliga a camada compute do staging |

Staging é um ambiente só: o `deploy-staging` usa `concurrency` para que um deploy da `develop` e um da release não rodem ao mesmo tempo. Enquanto uma release está em validação, convém não mesclar features na `develop`, para o staging ficar com a versão da release.

**Steps do `deploy-staging`**

1. **Lint e análise estática:** `go vet`, `golangci-lint`, `scripts/archcheck.sh`, `terraform fmt -check` e `terraform validate`.
2. **Testes unitários (portão obrigatório):** `go test ./... -race` em todos os serviços e pacotes. Se qualquer teste falhar, a esteira para aqui e nenhum step seguinte roda.
3. **Build e push:** imagem Docker de cada serviço, com a tag do hash da árvore do Git (`git rev-parse HEAD^{tree}`), push para o ECR.
4. **Liga staging:** se a camada compute estiver desligada, aplica o Terraform dela.
5. **Deploy em staging:** `terraform apply` de `staging/base` e `staging/compute` com a nova tag; espera os serviços ECS ficarem estáveis.
6. **Testes E2E em staging:** suíte em Go que envia webhooks assinados simulando o número simulador (consulta, remédio, água, resposta livre, mídia não suportada) e confere o resultado no DynamoDB, nos agendamentos e nas DLQs vazias.
7. **Testes funcionais com Playwright (futuro):** step já previsto, desligado na v1. Quando ativo, também é portão.
8. **Registro da validação:** a imagem recebe a tag adicional `staging-ok` no ECR, e o commit recebe o status `staging-ok`.

**Steps do `deploy-prod` (branch `main`)**

1. **Testes unitários (portão obrigatório):** rodam de novo sobre o merge commit; se falharem, nada segue.
2. **Localiza a imagem validada:** pelo hash da árvore do merge commit; a esteira exige a tag `staging-ok` e, sem ela, para.
3. **Aprovação manual:** o environment `production` do GitHub exige aprovação do administrador.
4. **Deploy em prod:** a mesma imagem é implantada em prod, sem rebuild. Se o health check falhar, o circuit breaker do ECS faz rollback.
5. **Smoke test em prod:** health check de cada serviço e verificação de que as DLQs continuam vazias.
6. **Tag da versão:** cria a tag Git `vx.y.z` (do nome da `release/*` ou `hotfix/*` mesclada) e uma release no GitHub com as mudanças.
7. **Lembrete de back-merge:** o resumo da run traz o link para abrir o PR da `main` para a `develop`.

**Acesso à AWS**

- Autenticação por OIDC do GitHub, com uma role por ambiente. A role de staging só pode ser assumida a partir da `develop`, de `release/*` e de `hotfix/*`; a de prod, só por jobs da `main` no environment `production`.
- Segredos da Meta ficam no Secrets Manager; o GitHub guarda só o ARN das roles.

O administrador recebe encaminhamentos dos dois ambientes ao mesmo tempo. Como cada ambiente tem seu número da marIA, as conversas chegam em chats separados no WhatsApp dele, além do prefixo `[STAGING]` ou `[PROD]` no texto.

## Segurança

Na v1 não há OAuth 2: não existe front-end, e a única chamada síncrona entre serviços fica dentro da VPC. A proteção vem de rede, IAM e validação de origem. OAuth 2 com Client Credentials entre serviços fica no backlog.

- **Origem das mensagens:** assinatura `X-Hub-Signature-256` validada em todo webhook e allowlist de números por ambiente.
- **Rede:** só o gateway é exposto, atrás de um ALB com HTTPS; os outros serviços não têm porta pública. A API interna do reminders só aceita tráfego do security group do conversation, via ECS Service Connect.
- **IAM de menor privilégio:** uma task role por serviço, com acesso só à própria tabela, às próprias filas e, no caso do outbound, ao bucket de imagens.
- **Criptografia:** KMS em DynamoDB, SQS, SNS e S3; TLS em todo o tráfego externo.
- **Segredos:** só no Secrets Manager, injetados na task pelo ECS; nada em variáveis em texto no repositório.
- **Logs:** texto de mensagem e telefones são mascarados no próprio serviço antes de ir para o CloudWatch Logs.

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

## CLAUDE.md

O CLAUDE.md vigente fica na raiz do repositório e é a única cópia; o funcionamento do harness está em `HARNESS.md`.
