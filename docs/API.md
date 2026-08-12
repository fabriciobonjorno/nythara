# API REST — Fase 3

Base local: `http://localhost:18080`. Respostas e erros usam JSON; rotas
protegidas recebem `Authorization: Bearer <access_token>`.

## Identidade

- `POST /v1/auth/register` — `email`, `password` (12–256 caracteres) e
  `username` (2–32; somente `A–Z`, `a–z`, `0–9`, `_` e `-`). E-mail e nome
  de usuário são únicos sem diferenciar maiúsculas/minúsculas. O alias legado
  `display_name` ainda é aceito durante o Alpha. Cria o direito-base do
  catálogo Alpha e a progressão global em 0 XP (nível 1); Lendárias continuam
  sujeitas aos marcos de nível.
- `POST /v1/auth/login` — `email`, `password`.
- `POST /v1/auth/refresh` — `refresh_token`; cada uso rotaciona o token.
- `POST /v1/auth/logout` — `refresh_token`.
- `POST /v1/auth/forgot-password` — `email`, `locale` (`pt-BR`, `es` ou `en`).
  Responde sempre `202`, inclusive para conta ausente ou falha transitória de
  entrega, para não enumerar cadastros. Quando aplicável, envia pelo Resend um
  link de uso único válido por 30 minutos.
- `POST /v1/auth/reset-password` — `token`, `password` (12–256 caracteres).
  Consome o token, troca a senha e revoga todas as sessões da conta na mesma
  transação. Link inválido, expirado ou já usado retorna
  `400 reset_token_invalid`.
- `GET /v1/auth/providers` — disponibilidade pública dos provedores sociais
  (`{"google":true|false}`); não expõe client ID nem segredo.
- `GET /v1/auth/google/start` — inicia Authorization Code + PKCE, com `state`
  e verifier em cookies HttpOnly de cinco minutos. Existe somente quando o
  provedor foi configurado.
- `GET /v1/auth/google/callback` — retorno cadastrado no Google. O backend
  troca o código e retorna ao SPA com um passe opaco de uso único, nunca com
  tokens Google ou tokens de sessão na URL.
- `POST /v1/auth/oauth/exchange` — `ticket`; consome o passe em até dois
  minutos e entrega o mesmo envelope `user` + `tokens` do login local.
- `POST /v1/webhooks/resend` — webhook de eventos operacionais de e-mail. A
  rota só existe quando `RESEND_WEBHOOK_SECRET` está configurado e exige o
  corpo original com `svix-id`, `svix-timestamp` e `svix-signature`. Eventos
  autênticos retornam `200`; retries com o mesmo ID são idempotentes. Somente
  ID da mensagem, tipo e horários são persistidos — nunca destinatário,
  assunto ou conteúdo.
- `GET /v1/me` — principal autenticado, incluindo `display_name`, emblema de
  perfil e se a conta já possui senha local.
- `PUT /v1/me/profile` — `avatar_id`; escolhe um dos dez emblemas originais
  do ruleset ativo para o perfil, sem alterar o Avatar do baralho.
- `PUT /v1/me/password` — `current_password`, `new_password`. Exige a senha
  atual quando ela existe; conta social pode criar a primeira. Revoga todas as
  sessões na mesma transação e retorna `204`.

## Catálogo e progressão

- `GET /v1/catalog/cards` — Lendárias incluem `unlock_level`.
- `GET /v1/catalog/champions`
- `GET /v1/rulesets/current` — versão e parâmetros de apresentação do ruleset
  ativo (`starting_vitality`, pressão, teto de vazamento, tamanho e mínimos de
  composição do baralho). O cliente não repete esses números como autoridade.
- `GET /v1/seasons/current`
- `GET /v1/collection` — projeta somente cartas disponíveis no nível atual;
  aquisições Lendárias ainda bloqueadas não são tratadas como posse jogável.
- `GET /v1/rewards`
- `GET /v1/progress` — rituais do dia (sorteio determinístico por
  `sha256(dia|usuário)`, materializado na primeira consulta), carteira de
  Fragmentos do Véu, nível global (`account`: XP total, nível 1–50 e progresso
  dentro do nível), maestria cosmética por Avatar e rating da temporada ativa.
- `GET /v1/ranked/leaderboard?limit=n` — topo da temporada (`entries`, máx.
  100) e a posição do solicitante (`me`). O bot nunca aparece. Cada entrada e
  o `me` carregam a patente derivada do rating (`tier`; faixas em
  `domain/ranks.go` — de Errante do Crepúsculo a Voz do Eclipse).
- `GET /v1/matches?limit=n` — histórico pessoal de partidas encerradas
  (oponente, campeões, resultado, motivo do fim).
- `GET /v1/matches/{id}/replay` — crônica pós-partida: cabeçalho + log
  completo de eventos authoritative. Somente participantes (ou admin) e
  somente com a partida encerrada — o log revela informação oculta.
- `GET /v1/decks/{id}/code` — exporta o deck como código compartilhável
  (`VR1.` + base64url; determinístico por lista).
- `POST /v1/decks/import` — corpo `{"code":"VR1.…","name":"…"}`; exige
  `Idempotency-Key`. O código nunca é confiado: o deck reconstruído passa
  pelo mesmo funil de validação e posse da criação normal.

A progressão é gravada exclusivamente pelo battle server ao fim da partida, a
partir dos eventos authoritative — o cliente não envia progresso. A gravação é
idempotente por partida (`match_progress_log`); fragmentos deixam trilha em
`economy_transactions`. Treinos, tutoriais e lutas contra bot rendem 0 XP de
conta e 0 XP de maestria; podem render apenas rituais não-PvP. Somente PvP com
dois humanos rende 15 XP (+15 por vitória). XP global tem teto físico de
28.420 e nível derivado com teto 50. Rating Elo (K=32) também só muda nesse PvP
real.

## Decks

No ruleset competitivo ativo, cada conta mantém um único baralho de 30 cartas
(mínimo 8 Assaltos, 10 Guardas e 4 Ritos), sem restrição de facção.
Após alteração manual, `locked_until` protege a lista por 24 horas; o baralho
inicial fornecido pelo sistema não nasce travado. `champion_id` permanece no
contrato como ID do Avatar cosmético e não altera a engine.

- `GET /v1/catalog/precons` — os 10 decks preconstruídos oficiais (públicos).
- `POST /v1/decks/precon` — copia um precon para os seus decks; corpo
  `{"champion_id": "CH-…"}`; exige `Idempotency-Key`.


- `GET /v1/decks`
- `POST /v1/decks`
- `GET /v1/decks/{id}`
- `PUT /v1/decks/{id}` — requer `version` atual no corpo.
- `DELETE /v1/decks/{id}?version=N`

Criação, alteração e exclusão exigem `Idempotency-Key` (8–128 caracteres).
Cartas são enviadas agregadas como `{"card_id":"VR-001","quantity":2}`.
O servidor valida regras, posse e nível de desbloqueio; uma constraint diferida
repete a validação estrutural no banco antes do commit. Atualizações concorrentes
usam a versão do deck. Um deck histórico com Lendária ainda bloqueada não entra
em treino/PvP e fica editável para substituição.

## Administração

- `POST /v1/admin/rewards/grant` — somente `admin`; exige
  `Idempotency-Key`. O corpo informa `user_id`, `source`, `quantity` e
  exatamente um entre `card_id` e `champion_id`.

Contas públicas nascem como `player`; elevação para `admin` é uma operação
administrativa fora da API pública.

### LiveOps (Fase 7) — todas exigem `admin`; mutações são idempotentes por
construção e auditadas em `admin_audit` na mesma transação.

- `GET /v1/admin/rulesets` — versões publicadas e o ponteiro ativo.
- `POST /v1/admin/rulesets/{version}/activate` — ativa a versão (rollback é
  ativar uma versão anterior). Reponta o matchmaking em tempo real.
- `POST /v1/admin/rulesets/{version}/rotate` — concede a coleção da versão a
  todos os jogadores e clona os decks válidos da versão ativa (idempotente).
  Ordem operacional: publicar → rotacionar → ativar.
- `GET|POST /v1/admin/drafts`, `GET|PUT /v1/admin/drafts/{id}` — drafts de
  carta (`card` = CardDef, `effects` = CardFx da DSL) sobre a versão ativa.
- `POST /v1/admin/drafts/{id}/validate` — schema + validador da DSL +
  compilação do catálogo candidato; resultado persistido no draft.
- `POST /v1/admin/drafts/{id}/simulate` — corpo `{"games": n}` (máx. 2000);
  simulação headless com verificação de replay numa versão efêmera.
- `POST /v1/admin/drafts/{id}/publish` — corpo `{"version": "alpha-x.y.z"}`;
  exige draft validado; cria versão imutável (inativa) e a registra na engine.
- `GET|POST /v1/admin/bans`, `POST /v1/admin/bans/{card}/lift` — desativação
  emergencial no competitivo, sem apagar histórico/coleções.
- `POST /v1/admin/seasons` — abre temporada (encerra a aberta no início da
  nova). Ao encerrar, concede Fragmentos pela patente final aos ranqueados da
  temporada fechada, na mesma transação (trilha em `economy_transactions` e
  resumo `season:rewards` na auditoria; idempotente pela transição — ADR-034).
- `GET /v1/admin/telemetry` — agregados das partidas persistidas (win rate por
  Campeão e ritmo humano PvP: duração/rodadas média, p50, p95 e partidas acima
  de 30 minutos); relatórios de simulação ficam nos artifacts da CI.
- `GET /v1/admin/audit?limit=n` — trilha de auditoria administrativa.

## Matchmaking e batalha realtime

- `GET /v1/matchmaking` — estado `idle`, `queued` ou `matched`.
- `POST /v1/matchmaking` — corpo `{"deck_id":"..."}`; fila 1v1 que preserva
  FIFO entre contas compatíveis e só pareia níveis com diferença absoluta ≤5.
- `DELETE /v1/matchmaking` — sai da fila enquanto ainda não pareado.
- `POST /v1/practice` — corpo `{"deck_id":"...", "bot_champion_id":"CH-…"}`
  (campeão opcional; sem ele o servidor sorteia um precon do bot). Cria
  partida de treino contra o Treinador do Véu no mesmo pipeline authoritative
  (persistência, replay, reconexão); não passa por bans nem conta para PvP.
- `POST /v1/battles/{id}/tickets` — cria ticket WebSocket de uso único por
  60 segundos. `mode` é `player`; `spectator` exige admin.
- `GET /v1/battles/{id}/ws?ticket=...&after_event=N` — upgrade WebSocket.

Mensagens aceitas do cliente:

```json
{"type":"ready"}
{"type":"command","client_sequence":1,"command":{"kind":"play","card":"p0-c03"}}
{"type":"command","client_sequence":2,"command":{"kind":"pass"}}
```

No Modo Confronto, `command.kind` aceita `play`, `pass` e `concede`; campos
legados continuam no schema somente para replays/versionamento ≤0.8.3.
O envelope aceita somente `kind`, `card`, `cards`, `stance` e `decision_id`.
O slot do jogador vem do ticket. Campos como `player`, `damage`, `draw`,
`winner`, `reason` ou resultados calculados são rejeitados. Alvos que exigem
escolha usam o comando tipado `choose`; a engine atual não recebe `target_id`.

O servidor envia `sync`, `ready`, `events`, `match_cancelled` e `error`.
`sync` contém a visão atual redigida e os eventos posteriores a `after_event`.
Para jogadores, `sync.client_sequence` informa a última sequência confirmada
daquele assento pelo servidor, permitindo retomar com segurança após queda
entre envio e ACK. Erros de transporte distinguem `connection_closed` (conexão
já encerrada) e `subscriber_too_slow` (cliente deixou de consumir o fluxo) de
`spectator_read_only`, reservado ao espectador conectado que tentou escrever.
Cada jogador tem sequência própria, iniciando em 1. Repetição da mesma intenção
canônica é idempotente; salto ou reutilização com outro conteúdo é erro de protocolo.

A visão de jogador também inclui `state.playable`, lista de IDs de instância
da própria mão que a engine aceita na janela atual. Ela já considera fase,
custo de Vitalidade, sacrifício, alvos e requisitos da carta; o cliente deve
usar essa lista para habilitar cartas, sem duplicar regras. Espectadores não
recebem `playable` e continuam sem acesso às mãos privadas (ADR-046).

Prazos atuais: 30 segundos para ambos ficarem prontos e 45 segundos por ação.
Timeout de ação é persistido como concessão server-side. O espectador recebe
estado público e não pode enviar `ready` ou comandos.

## Erros

Formato estável:

```json
{"error":{"code":"invalid_request","message":"..."}}
```

Códigos HTTP relevantes: `400` validação, `401` autenticação, `403` RBAC,
`404` recurso, `409` conflito de versão/idempotência e `500` falha interna.
