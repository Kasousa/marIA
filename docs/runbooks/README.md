# Runbooks

Passo a passo de operação, escritos para o administrador (e para o agente entender o que ele não pode fazer sozinho).

Previstos, conforme o roteiro:

- `bootstrap.md` (Fase 1): bucket de state e primeira aplicação do OIDC, a única aplicação manual de Terraform.
- `branch-protection.md` (Fase 1): proteção de `develop` e `main`.
- `staging-power.md` (Fase 1): ligar e desligar o staging pelo GitHub.
- `meta-templates.md` (Fase 3): cadastro e aprovação de templates nas duas contas.
- `dlq.md` (Fase 3): como inspecionar e reprocessar mensagens em DLQ.
- `snoopy-images.md` (Fase 6): curadoria das imagens no bucket.
