# ADR 0003: us-east-1, conta única e entrada por API Gateway

- Data: 2026-10-24
- Status: aceita

## Contexto
A spec deixava em aberto a região, a organização de contas e como expor o webhook com HTTPS (ela previa um ALB). O orçamento é de R$150/mês para os dois ambientes, e o webhook da Meta exige HTTPS com certificado válido.

## Decisão
1. **Região `us-east-1`.**
   - A cliente e o administrador nunca falam direto com a AWS: as mensagens passam pelos servidores da Meta. Por isso, a região não muda a latência percebida.
   - `us-east-1` é mais barata que `sa-east-1`.
   - O Claude Haiku 4.5 está disponível na própria região. Em `sa-east-1`, ele só seria acessível pelo perfil de inferência global, processado fora do Brasil.
2. **Uma conta AWS (a pessoal do administrador)** para staging e prod.
   - Os ambientes se isolam por nome (`-staging`/`-prod`), tag `env` e IAM: a role da esteira de cada ambiente só alcança recursos do próprio ambiente.
3. **Entrada por API Gateway HTTP API**, ligado à task do gateway por VPC Link e Cloud Map, sem load balancer.
   - O webhook usa a URL padrão `execute-api`, que já tem certificado válido.
   - No staging, o API Gateway, o VPC Link e o namespace do Cloud Map ficam na camada `staging/base`, para a URL não mudar quando a compute for destruída.
4. **Domínio próprio fica em melhorias futuras** (backlog da spec).

## Alternativas consideradas
- **`sa-east-1`:** dados em repouso no Brasil, mas custo maior e Bedrock só pelo perfil global. A preferência inicial era só "estou no Brasil", o que não traz ganho nesta arquitetura.
- **ALB com HTTPS:** custo por hora em cada ambiente (prod sempre ligado) e exigiria um domínio e um certificado ACM desde o primeiro deploy.
- **Duas contas AWS (staging e prod):** isolamento mais forte, mas mais configuração (Organizations, OIDC e budget por conta) para um projeto de estudo.

## Consequências
- **Dados fora do Brasil:** dados de saúde em repouso (DynamoDB, S3) ficam nos EUA. As mensagens já trafegam pela Meta fora do Brasil.
- **Permissões por nome:** as políticas IAM precisam ser escritas com cuidado. A seção Infra do CLAUDE.md e o `/security-review` cobrem a regra "role de um ambiente só alcança recursos do próprio ambiente".
- **Integração do API Gateway:** precisa ser do tipo proxy, sem transformar o corpo, para validar o `X-Hub-Signature-256`. O serviço ECS do gateway registra SRV no Cloud Map, o único tipo de registro que o API Gateway consulta.
- **Webhook do staging:** com o staging desligado, o API Gateway responde erro. Reenvios da Meta são tratados pela idempotência por `wamid`.
- **URL ligada ao id do API:** se o API Gateway for recriado, a URL do webhook muda e precisa ser atualizada na Meta. O domínio próprio, quando vier, resolve isso.
