# Infraestrutura

Ambientes, Terraform, custos, liga/desliga do staging e segurança.

## Infraestrutura e ambientes

Tudo roda em containers Docker no ECS Fargate, provisionado com Terraform e implantado pela esteira do GitHub Actions, em dois ambientes idênticos: staging e prod. A mesma imagem passa pelos dois, e só a configuração muda.

Os dois ambientes ficam na mesma conta AWS (a conta pessoal do administrador), na região `us-east-1`, isolados por nome (sufixo `-staging` ou `-prod`), por tag `env` e por IAM: a role de cada ambiente só alcança recursos do próprio ambiente. A decisão está no ADR 0003.

**Stack**

- Serviços em Go (versão estável mais recente), com `net/http` + `chi`, `aws-sdk-go-v2` e `log/slog`.
- Imagens Docker multi-stage `linux/arm64`, finais em `distroless/static`, publicadas no ECR com a tag do hash da árvore do Git.
- Mensageria: SNS (tópico `maria-events`) com filas SQS por consumidor, cada uma com DLQ.
- Segredos (token da Meta, app secret, allowlist) no Secrets Manager.
- Logs básicos em JSON no CloudWatch Logs, com retenção de 14 dias e sem texto de mensagem em claro. Observabilidade completa com Datadog fica para a v2.
- Sem ambiente local de integração: localmente só rodam testes unitários com fakes (`go test`); toda integração é testada em staging.

**Ambientes**

|  | staging | prod |
| --- | --- | --- |
| Orquestração | ECS Fargate, 1 task por serviço | ECS Fargate, 1 task por serviço, deployment circuit breaker com rollback |
| Tamanho das tasks | 0,25 vCPU / 0,5 GB, ARM64, Fargate Spot | 0,25 vCPU / 0,5 GB, ARM64, Fargate Spot (ADR 0004) |
| Número da marIA | Teste | Oficial |
| Usuária | Número simulador | Cliente |
| Disponibilidade | Liga/desliga, desligado por padrão | Ligado das 05:50 às 00:00; desligado das 00:00 às 05:50 por dois schedules que zeram e restauram o `desiredCount` (ADR 0004) |
| Entrada do webhook | API Gateway HTTP API → VPC Link → Cloud Map → task do gateway | Igual a staging |
| Rede | Subnets públicas (tasks com IP público para sair à internet), security groups restritos, sem NAT Gateway; só o VPC Link alcança o gateway | Igual a staging |

**Estrutura do Terraform**

- `modules/`: `network`, `api-gateway`, `ecs-cluster`, `ecs-service`, `messaging`, `dynamodb`, `scheduler`, `media-bucket`, `github-oidc`.
- `bootstrap/`: bucket de state e budget alarm; aplicado uma vez à mão pelo administrador.
- `envs/shared`: recursos da conta usados pelos dois ambientes (repositórios ECR, provedor OIDC e roles da esteira).
- `envs/staging/base`, `envs/staging/compute` e `envs/prod`, cada um com state remoto próprio no S3 (lock nativo do S3).
- O módulo `github-oidc` cria o provedor OIDC e uma role por ambiente, para a esteira acessar a AWS sem chaves de acesso.

**Custos**

- Orçamento de R$150/mês para os dois ambientes juntos. Staging fica desligado por padrão e só é ligado para testar.
- Budget alarm da AWS em 80% e 100% desde o primeiro dia. O e-mail de destino é passado na linha de comando do bootstrap e nunca vai para o repositório, que é público.
- Estimativa (preços de lista, a confirmar): com 5 tasks em Spot ARM e prod desligado de madrugada, ~US$ 32/mês (~R$ 175), um pouco acima do orçamento; o risco foi aceito (ADR 0004). Com Spot, o IPv4 público (~US$ 3,65 por task por mês) é o maior custo por task.
- Sem load balancer: a entrada é o API Gateway HTTP API, cobrado por requisição, e o VPC Link e o Cloud Map não têm custo fixo relevante. Isso evita o custo por hora de um ALB em cada ambiente.
- Valores de Fargate, API Gateway, Bedrock e WhatsApp em `us-east-1` a confirmar nas páginas de preço antes do primeiro deploy.

## Liga/desliga do staging

O staging fica desligado por padrão e o administrador liga só quando vai testar. Para isso, o Terraform do staging é dividido em duas camadas: a base fica sempre de pé e guarda os dados; a compute é criada e destruída pelo liga/desliga.

| Camada | O que tem | Quando o staging está desligado |
| --- | --- | --- |
| `staging/base` (sempre ligada) | DynamoDB, SNS/SQS, bucket S3, Secrets Manager, KMS, API Gateway, VPC Link e namespace do Cloud Map (os repositórios ECR ficam em `envs/shared`) | Continua existindo; custo mínimo (on-demand sem tráfego, mais o valor fixo de segredos e chaves); a URL do webhook não muda |
| `staging/compute` (liga/desliga) | Serviços ECS, schedules do EventBridge | Destruída: sem custo de Fargate |

**Como operar**

- Um workflow do GitHub Actions, `staging-power`, com acionamento manual e opção `on` ou `off`. Dá para disparar pelo app do GitHub no celular.
- `on` aplica a camada compute e, ao terminar, roda um health check dos serviços. `off` destrói a camada compute.
- Desligamento automático todo dia às 23h, para o staging não ficar ligado por esquecimento.
- A esteira de deploy liga o staging sozinha quando precisa testar uma nova versão.

**Enquanto desligado**

- Mensagens enviadas ao número de staging da marIA não são processadas: o API Gateway continua no ar, mas sem task registrada no Cloud Map ele responde erro. É aceitável, porque só o número simulador e o administrador falam com ele. Se a Meta reenviar o webhook depois que o staging religar, a idempotência pelo `wamid` evita duplicidade.
- Nenhum lembrete de staging dispara, porque os schedules fazem parte da camada compute. Ao religar, o reminders-service recria os schedules a partir dos itens ativos no DynamoDB e refaz o planejamento de água do dia.
- Dados de teste (consultas, remédios, conversas) ficam preservados na camada base.

## Segurança

Na v1 não há OAuth 2: não existe front-end, e a única chamada síncrona entre serviços fica dentro da VPC. A proteção vem de rede, IAM e validação de origem. OAuth 2 com Client Credentials entre serviços fica no backlog.

- **Origem das mensagens:** assinatura `X-Hub-Signature-256` validada em todo webhook e allowlist de números por ambiente.
- **Rede:** só o gateway recebe tráfego de fora, e apenas pelo API Gateway com HTTPS (o security group dele aceita só o VPC Link); os outros serviços não aceitam conexões de entrada. A API interna do reminders só aceita tráfego do security group do conversation, via ECS Service Connect.
- **IAM de menor privilégio:** uma task role por serviço, com acesso só à própria tabela, às próprias filas e, no caso do outbound, ao bucket de imagens.
- **Criptografia:** KMS em DynamoDB, SQS, SNS e S3; TLS em todo o tráfego externo.
- **Segredos:** só no Secrets Manager, injetados na task pelo ECS; nada em variáveis em texto no repositório.
- **Logs:** texto de mensagem e telefones são mascarados no próprio serviço antes de ir para o CloudWatch Logs.
