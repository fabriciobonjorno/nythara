# API REST — Fase 3

Base local: `http://localhost:8080`. Respostas e erros usam JSON; rotas
protegidas recebem `Authorization: Bearer <access_token>`.

## Identidade

- `POST /v1/auth/register` — `email`, `password` (12–256 caracteres) e
  `display_name`. Cria o grant competitivo completo do Alpha no servidor.
- `POST /v1/auth/login` — `email`, `password`.
- `POST /v1/auth/refresh` — `refresh_token`; cada uso rotaciona o token.
- `POST /v1/auth/logout` — `refresh_token`.
- `GET /v1/me` — principal autenticado.

## Catálogo e progressão

- `GET /v1/catalog/cards`
- `GET /v1/catalog/champions`
- `GET /v1/rulesets/current`
- `GET /v1/seasons/current`
- `GET /v1/collection`
- `GET /v1/rewards`

## Decks

- `GET /v1/decks`
- `POST /v1/decks`
- `GET /v1/decks/{id}`
- `PUT /v1/decks/{id}` — requer `version` atual no corpo.
- `DELETE /v1/decks/{id}?version=N`

Criação, alteração e exclusão exigem `Idempotency-Key` (8–128 caracteres).
Cartas são enviadas agregadas como `{"card_id":"VR-001","quantity":2}`.
O servidor valida regras e posse; uma constraint diferida repete a validação
no banco antes do commit. Atualizações concorrentes usam a versão do deck.

## Administração

- `POST /v1/admin/rewards/grant` — somente `admin`; exige
  `Idempotency-Key`. O corpo informa `user_id`, `source`, `quantity` e
  exatamente um entre `card_id` e `champion_id`.

Contas públicas nascem como `player`; elevação para `admin` é uma operação
administrativa fora da API pública.

## Matchmaking e batalha realtime

- `GET /v1/matchmaking` — estado `idle`, `queued` ou `matched`.
- `POST /v1/matchmaking` — corpo `{"deck_id":"..."}`; fila FIFO 1v1.
- `DELETE /v1/matchmaking` — sai da fila enquanto ainda não pareado.
- `POST /v1/battles/{id}/tickets` — cria ticket WebSocket de uso único por
  60 segundos. `mode` é `player`; `spectator` exige admin.
- `GET /v1/battles/{id}/ws?ticket=...&after_event=N` — upgrade WebSocket.

Mensagens aceitas do cliente:

```json
{"type":"ready"}
{"type":"command","client_sequence":1,"command":{"kind":"mulligan","cards":[]}}
```

`command` aceita somente `kind`, `card`, `cards`, `stance` e `decision_id`.
O slot do jogador vem do ticket. Campos como `player`, `damage`, `draw`,
`winner`, `reason` ou resultados calculados são rejeitados. Alvos que exigem
escolha usam o comando tipado `choose`; a engine atual não recebe `target_id`.

O servidor envia `sync`, `ready`, `events`, `match_cancelled` e `error`.
`sync` contém a visão atual redigida e os eventos posteriores a `after_event`.
Para jogadores, `sync.client_sequence` informa a última sequência confirmada
pelo servidor, permitindo retomar com segurança após queda entre envio e ACK.
Cada jogador tem sequência própria, iniciando em 1. Repetição da mesma intenção
canônica é idempotente; salto ou reutilização com outro conteúdo é erro de protocolo.

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
