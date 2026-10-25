# infra/: regras de Terraform

Complementa o CLAUDE.md da raiz. Detalhes em docs/spec/infra.md e no ADR 0003.

## Estrutura
- modules/: network, api-gateway, ecs-cluster, ecs-service, messaging, dynamodb, scheduler, media-bucket, github-oidc.
- envs/staging/base, envs/staging/compute e envs/prod, cada um com state remoto próprio no S3 (lock nativo).
- bootstrap/: bucket de state e primeira aplicação do OIDC; único Terraform aplicado à mão, pelo administrador (docs/runbooks/bootstrap.md).

## Regras
- Região us-east-1, uma conta para os dois ambientes: todo recurso leva o ambiente no nome (-staging/-prod) e a tag env.
- Uma task role por serviço, só com a própria tabela e as próprias filas (outbound também com o bucket de imagens); nada de `*` sem justificativa.
- OIDC: role de staging só pela develop, release/* e hotfix/*; role de prod só pela main no environment production; cada role só alcança recursos do próprio ambiente.
- staging/base guarda os dados e a entrada (API Gateway, VPC Link, namespace do Cloud Map), para a URL do webhook não mudar; staging/compute (serviços ECS, schedules) é destruída todo dia, então nada com dados nela.
- Sem load balancer: API Gateway HTTP API -> VPC Link -> Cloud Map (registros SRV) -> task do gateway; integração proxy, sem transformar o corpo.
- KMS gerenciada pelo cliente em DynamoDB, SQS, SNS e S3; TTL nas tabelas que a spec pede.
- Custo (orçamento R$150/mês): Fargate Spot em staging, 0,25 vCPU / 0,5 GB, sem NAT, logs com 14 dias; todo recurso novo com custo fixo vem com estimativa mensal.
- Segredos: só ARNs no código; nenhum valor de segredo em .tf ou .tfvars.

## Como trabalhar
- Rode `make tf-check` (fmt -check e validate sem backend) antes de concluir. `plan` e `apply` só na esteira.
- Mudança de IAM ou rede: rode /security-review e mostre o diff das políticas.
