# .github/: regras dos workflows

Complementa o CLAUDE.md da raiz. Detalhes em docs/spec/esteira.md e no ADR 0002.

- Workflows: ci (PRs para develop e main), deploy-staging (push na develop, release/** e hotfix/**), deploy-prod (push na main), staging-power (manual e 23h).
- `permissions:` mínimas por job; `id-token: write` só nos jobs que assumem role da AWS.
- Actions de terceiros fixadas por SHA completo, com a versão num comentário.
- `go test ./... -race` roda antes de qualquer build ou deploy; a esteira para no primeiro step que falhar.
- Jobs de feature/* nunca assumem role da AWS; terraform validate roda sem backend.
- Tag da imagem = `git rev-parse HEAD^{tree}`; prod nunca faz rebuild e exige staging-ok.
- `concurrency` de staging para nunca rodar dois deploys de staging ao mesmo tempo.
- O ci também roda `make harness-test`, para os guarda-corpos do harness não quebrarem sem ninguém perceber.
- Build de imagem sempre com `--platform linux/arm64` (ADR 0004); o Dockerfile não fixa a arquitetura.
