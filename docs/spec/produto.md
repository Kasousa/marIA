# Produto: tom e fluxos

O que a marIA faz e como ela fala. Leia antes de mexer em prompt, templates ou qualquer texto para a cliente.

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
