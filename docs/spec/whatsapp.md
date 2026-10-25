# WhatsApp Cloud API

Regras da Meta que moldam o desenho: janela de 24h, templates, imagens e webhook.

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
- O webhook usa a URL padrão do API Gateway de cada ambiente (`https://<api-id>.execute-api.us-east-1.amazonaws.com/webhook`), que já tem certificado válido da AWS. A integração é proxy, sem transformar o corpo, para a validação da assinatura funcionar. Domínio próprio fica em melhorias futuras.
- Rate limit e retry com backoff exponencial no outbound-service.
