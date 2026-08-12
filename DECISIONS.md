# Registro de Decisões — Nythara

Decisões tomadas onde o GDD era ambíguo, no formato ADR curto. Toda mudança de
regra passa por aqui e regenera os goldens (`make golden`) com revisão.

## ADR-001 — Deslocamento intrínseco de Eclipse por carta

Toda carta desloca o Medidor pelo seu `eclipse_shift` ao resolver (Vhal puxa
para a Noite, Solara para a Aurora, etc.), além de efeitos explícitos de texto.
Exceção: cartas cujo texto descreve o próprio movimento (VR-068, VR-069) têm
`ownShift` — o texto executa o deslocamento e o intrínseco é suprimido, para
não dobrar o movimento.

## ADR-002 — Medidor e estados totais

- Atingir ±3 dispara o estado total (Eclipse Noturno / Aurora Total), **uma vez
  por rodada**; o estado persiste até o fim da rodada mesmo que o medidor recue.
- No Crepúsculo de uma rodada com estado total, o medidor **retorna a 0**.
- Sem estado total, o medidor **persiste** entre rodadas (é a "segunda partida"
  de cabo de guerra do GDD §7).

## ADR-003 — Ressonância: trilha própria + linha do tempo global

- Cada jogador tem sua Trilha (limite 5/rodada; Rauk em Eclipse Noturno: 6).
- Condições `X→Y` avaliam a trilha **própria**, imediatamente **antes** de a
  carta atual emitir seu Sigilo (senão VR-005 nunca dispararia).
- Existe também uma linha do tempo **global** da rodada; referências como
  "o último Sigilo desta rodada" (VR-025) leem a global — inclusive sigilos do
  oponente. Isso torna a carta reativa a Solara/Vhal, coerente com o flavor.
- Com a trilha no limite, nada é emitido (nem gatilhos derivados).

## ADR-004 — Ordem de resolução do combate

1. Assalto pago e anunciado (dano base + condicionais calculados; bônus de
   Predação/Campeão aplicados **antes** da janela de Guarda — o defensor reage
   ao poder declarado).
2. Janela de Guarda (máx. 1 Guarda). A Guarda resolve como reação: emite Sigilo
   e desloca o Eclipse **antes** do dano.
3. Dano por instância: Exposto (+2, pré-prevenção, consumido na 1ª instância
   mesmo se prevenida) → prevenção → Ward → Vitalidade.
4. Efeito posterior da Guarda (sabe se preveniu tudo). Efeitos com decisão
   (VR-062) rodam aqui, após o Assalto, para não pausar o combate.
5. Sigilo e deslocamento intrínseco do Assalto → efeitos posteriores → gatilhos
   de permanentes.

"Ignora X de Ward" (VR-020): o Ward efetivo é reduzido em X naquele Assalto;
o Ward não consumido permanece.

## ADR-005 — Ward só absorve dano de Assalto

Fadiga, Sangramento e Maldição são perda direta de Vitalidade. Coerente com a
forma de Eclipse de Kaedor ("não pode ser prevenido por Ward" só faz sentido se
Ward age sobre Assaltos).

## ADR-006 — Sangramento e Maldição disparam no Crepúsculo da rodada seguinte

"No Crepúsculo da próxima rodada do alvo" = rodada R+1 para efeito aplicado na
rodada R (rodadas são compartilhadas). Ordem de resolução no Crepúsculo:
jogador com iniciativa primeiro.

## ADR-007 — Estrutura de rodada e janelas

- Compra em **toda** rodada, inclusive a 1ª (mão sobe a 6 antes das Posturas).
- Janelas sequenciais: iniciativa joga quantas cartas quiser e passa; depois o
  oponente; sem retorno. Rito/Relíquia/Manifestação só na fase de Rito;
  Assalto só no Confronto.
- Limite de mão (7) aplicado no Crepúsculo via decisões de descarte (iniciativa
  primeiro). A compra da rodada seguinte pode levar a mão a 8 até o próximo
  Crepúsculo.
- Vitória simultânea (duplo nocaute no mesmo passo): vence quem tiver mais
  Vitalidade; persistindo o empate, o jogador **sem** iniciativa.

## ADR-008 — Custos

- Essência temporária é gasta primeiro (expira no fim da rodada).
- Sacrifício de Vitalidade exige sobreviver (Vitalidade > custo).
- Postura Arcano: desconto só para Ritos **sem dano direto** (flag por carta).
- Todos os modificadores aplicáveis se acumulam; piso 0.
- Cura limitada à Vitalidade inicial do Campeão.

## ADR-009 — Terceira Relíquia/Manifestação

No alpha, jogar a 3ª é rejeitado (sem substituição). Substituição com destruição
explícita entra em iteração futura, junto com remoção (VR-046).

## ADR-010 — Replay e snapshots

Partida reproduzível por `ruleset + seed + decks + command_log` (replay do
zero). Decisões pendentes são tipadas e serializáveis (sem closures), então
snapshot mid-partida é viável, mas restaurar-e-continuar ainda não é suportado
— entra na Fase 4 (reconexão usa snapshot + eventos posteriores).

## ADR-011 — DSL de efeitos (Fase 2)

Cartas são definidas em `backend/internal/engine/data/effects_alpha.json`
(versionado; `version` acompanha `RulesetVersion`). Um validador roda no boot e
rejeita: op/condição desconhecida, seção incompatível com o tipo da carta,
valores fora de intervalo, três decisões encadeadas, gatilho de Sigilo que
emite Sigilo sem `once_per_round` (anti-loop), cobertura incompleta (toda
carta do catálogo está em `cards` OU em `unsupported` com motivo). O
compilador transforma as definições em closures sobre o intérprete — nenhum
`switch` por carta. Handlers especializados em Go seguem permitidos como
exceção registrada em `impls.go`. Decisões pendentes carregam a continuação
(`then`) como dados — serializável, sem closures no estado.

## ADR-012 — Interpretações da Fase 2

- **VR-068/VR-069/VR-023**: o `eclipse_shift` do catálogo descreve o movimento
  do próprio texto (`own_shift`) — não soma duas vezes.
- **Véu em efeitos condicionais** (VR-039): cartas cujo único efeito alveja o
  oponente são rejeitadas ao jogar (`targets_opponent`); ramos condicionais
  que alvejariam um jogador sob Véu fracassam na aplicação (evento
  `status_fizzled`). VR-067 remove Véu por design e o ignora.
- **VR-021**: "até sua próxima rodada" = Manifestações adversárias suprimidas
  até o fim da rodada seguinte.
- **VR-074**: prioridade determinística — remove a própria Maldição; senão a
  do oponente (e compra 1).
- **VR-008**: "perca Vitalidade" é perda direta (não é sacrifício; não aciona
  passiva de Seris).
- **VR-059**: exige 2+ cartas no descarte (recupera sempre 2, descarta 2).
- **VR-078**: encerra a janela de Assalto do atacante atual (marca passe).
- **VR-046**: destruição de Relíquia não é "alvo em jogador" — Véu não bloqueia.
- **VR-056**: a sobretaxa +1 usa modificador de custo por instância.

## ADR-013 — Comandos activate e ultimate

Habilidades ativadas de permanentes (VR-058, VR-072) e as ultimates de
Campeão são comandos próprios, legais na janela de Rito do dono (ultimates
também no próprio Confronto — Seris precisa ter causado dano). `CanActivate`/
`CanUltimate` são a fonte única de validação para engine, bots e UI. A Recusa
da Morte de Kaedor é automática (dispara no funil central de perda de
Vitalidade), não um comando.

## ADR-014 — Janela de reação a Ritos (VR-035)

Ritos direcionados (alvejam o oponente ou revelam a mão dele) abrem uma
janela de reação **apenas quando o defensor tem um counter jogável na mão**.
Sempre abrir atrasaria todo Rito direcionado; abrir condicionalmente vaza a
informação de que o defensor segura um counter — tradeoff aceito no alpha e
revisitado na Fase 4 (timer de auto-passe permite sempre abrir). O Rito
anulado vai ao descarte sem resolver; o conjurador recupera metade do custo
pago como Essência temporária.

## ADR-015 — Sistema de cópias

Cópias (VR-036, ultimate de Nyra) são virtuais: sem instância física, sem
custos adicionais, sem zonas. Emitem o Sigilo da carta copiada. Anti-loop:
Lendárias jamais são copiadas; cópias não entram no registro de jogadas da
rodada (não podem ser recopiadas); profundidade máxima 2. Cópias de Assalto
abrem janela de Guarda normalmente. VR-032 espelha os gatilhos de uma
Relíquia adversária até o fim da rodada (instância virtual com contador
próprio de "uma vez por rodada"). VR-030 emite também o Sigilo da carta
jogada imediatamente antes dele (rastreada entre rodadas).

## ADR-016 — Interpretações dos Campeões

- **Seris**: "efeitos de Dreno" = curas que ela recebe de efeitos; +1 na
  Noite. Ultimate cura metade do dano líquido do seu último Assalto.
- **Ilyan**: a anulação armada persiste até consumir um deslocamento causado
  por carta adversária (inclusive intrínseco).
- **Oren**: reordena o topo antes do mulligan (decisão pré-jogo); em Eclipse
  Total, a primeira compra além da compra normal fica 1 mais barata.
- **Nyra**: scry ao atingir exatamente 3 Sigilos na rodada; em Eclipse Total
  as cópias emitem um Sigilo extra.
- **Edda**: marcadores de Cinza contam descartes causados pelas próprias
  cartas (custos e efeitos; limite de mão não conta).
- **Saela**: na Noite, o Sigilo Garra de Assaltos ecoa (emissão dupla).
- **VR-076**: empate de Vitalidade → o controlador da Relíquia escolhe a direção.

## ADR-017 — Fronteira de persistência e autenticação da Fase 3

- PostgreSQL é a fonte de verdade para identidade, coleção, decks, temporadas,
  recompensas e idempotência. O catálogo jogável continua embutido e validado
  pela engine, mas é sincronizado em tabelas versionadas para referências,
  consultas e histórico; Redis não participa de decisões de economia.
- Tokens de acesso e refresh são opacos, aleatórios e armazenados apenas como
  SHA-256. Acesso é curto (15 minutos); refresh dura 30 dias e é rotacionado de
  forma transacional, revogando a sessão se um token antigo for reutilizado.
  Senhas usam PBKDF2-HMAC-SHA-256 com salt aleatório e parâmetros gravados no
  próprio hash.
- No Alpha, criar conta concede, por transação server-side auditada, os 10
  Campeões e a coleção competitiva completa (2 cópias de cartas não-Lendárias,
  1 de Lendárias). Isso materializa a diretriz de PvP justo sem inventar ainda
  uma economia de aquisição; temporadas futuras podem trocar o grant inicial.
- Escritas REST relevantes exigem `Idempotency-Key`, vinculado a usuário,
  operação e hash do corpo. Reutilizar a chave com conteúdo diferente é erro.
- Legalidade de deck tem defesa em profundidade: `engine.ValidateDeck` na
  aplicação e constraint trigger diferida no PostgreSQL, incluindo posse,
  quantidade, facção, tamanho e raridade. Assim, nenhum caminho de escrita
  consegue confirmar um deck ilegal.

## ADR-018 — Protocolo authoritative e salas da Fase 4

- Cada sala é um actor single-threaded: somente sua goroutine chama
  `engine.Apply`. O cliente envia uma intenção sem `player`, dano, compra ou
  vencedor; a identidade autenticada determina o slot e a engine determina o
  resultado.
- `client_sequence` é estritamente crescente por jogador. Repetir a mesma
  sequência com o mesmo payload devolve o resultado persistido sem reaplicar;
  payload diferente ou salto de sequência é rejeitado como violação de
  protocolo. Comando aceito, eventos e snapshot são uma transação PostgreSQL.
- O matchmaking inicial é FIFO e local ao monólito. Uma entrada de fila pode
  se perder em restart, mas uma partida formada nunca: configuração, jogadores,
  comandos, eventos e snapshots ficam no PostgreSQL. Redis entra quando houver
  múltiplas instâncias e não será fonte de verdade da partida.
- Ambos confirmam `ready` antes da engine iniciar. Há 30 s para ready e 45 s
  por ação; timeout durante a partida vira concessão server-side do jogador que
  deve agir. Em Mulligan/Postura, a ordem de timeout é iniciativa e depois o
  oponente, embora a engine aceite submissão antecipada de ambos.
- Snapshot é persistido no início, a cada 10 comandos aceitos e no fim. Restart
  restaura o snapshot e reaplica o catch-up persistido, verificando que os
  eventos gerados coincidem. Reconexão recebe visão atual + eventos posteriores
  ao cursor do cliente.
- Visões são redigidas por destinatário: mão/deck e decisões privadas só vão ao
  dono; eventos de compra e revelação só expõem carta a quem pode vê-la.
  Tickets WebSocket são aleatórios, de uso único e expiram em 60 s. Espectador
  é somente leitura e, no Alpha, tickets de espectador exigem papel admin.

## ADR-019 — Cliente Web/PWA da Fase 5

- O cliente é uma SPA React + TypeScript. React Router define URLs estáveis;
  TanStack Query cuida do estado remoto; Zustand contém apenas sessão e estado
  efêmero de batalha. O cliente nunca replica a resolução da engine: cartas
  potencialmente jogáveis são exibidas, mas toda legalidade continua sendo
  confirmada pelo servidor authoritative.
- A mesa combina uma camada PixiJS estritamente visual (atmosfera, Eclipse e
  movimento) com controles HTML semânticos. Assim teclado, leitores de tela,
  foco, zoom textual e redução de movimento não dependem do canvas.
- Tokens ficam em `sessionStorage`, pois a API Alpha entrega tokens no corpo e
  ainda não oferece cookie HttpOnly. Preferências não sensíveis ficam em
  `localStorage`. Fechar a sessão do navegador encerra a autenticação; migrar o
  refresh para cookie seguro fica para a fase de hardening.
- O PWA usa manifesto e service worker próprios, com cache apenas do shell e de
  respostas públicas do catálogo. APIs autenticadas e WebSocket nunca são
  persistidos pelo service worker. Durante batalha, perda de conexão cria novo
  ticket de uso único e retoma com `after_event`; `sync.client_sequence` traz a
  última sequência confirmada e a intenção pendente é reenviada idempotentemente.
- Replay, ranking histórico e estatísticas ainda não possuem leitura REST na
  superfície atual. Na Fase 5, replay reproduz a linha do tempo recebida pela
  sessão e o perfil mostra temporada/identidade sem inventar MMR ou resultados.
  Persistência e consulta histórica entram quando os módulos `replay` e
  `ranked` forem expostos pelo backend.

## ADR-020 — Simulação e métricas de balanceamento da Fase 6

- O runner headless usa uma matriz ordenada 10×10 de Campeões, repetida em
  blocos. Dentro de cada confronto, a iniciativa alterna por repetição e as
  seeds da partida e dos bots derivam apenas de `base_seed + índice`. Workers
  podem executar em qualquer ordem; o relatório agrega inteiros e ordena todas
  as saídas, portanto o mesmo input produz bytes equivalentes nas métricas.
- Cada Campeão recebe um deck preconstruído determinístico de sua facção e de
  Errantes, validado por `ValidateDeck`. O baseline não mistura facção aliada:
  isso reduz variáveis na matriz inicial; decks gerados/fuzz e matchups de
  arquétipo entram como cenários adicionais, não no baseline de Campeões.
- `loop` é uma partida que alcança o teto configurado de comandos sem terminar.
  `estado morto` é uma partida não encerrada em que o bot não encontra comando.
  Comando rejeitado, panic recuperado e divergência entre replay e snapshot/log
  são falhas separadas e fazem o comando do simulador retornar erro.
- Carta dominante é um sinal, não um veredito: o relatório ordena por
  `played_win_rate` apenas cartas jogadas em pelo menos `max(20, 0,5% das
  partidas)` e marca alerta acima de 65%. `dead_in_hand_rate` é a fração das
  aparições em mãos finais; não afirma que a carta era literalmente injogável.
  Dano, prevenção, deslocamento, rodada média, compra e mulligan são atribuídos
  pelos eventos/instâncias authoritative, nunca por estimativa do cliente.
- Pull requests rodam uma amostra curta. O job noturno/manual executa 100 mil
  partidas com replay integral e publica JSON + Markdown como artifact de CI.
  Os relatórios não levam timestamp nem duração de CPU para permanecerem
  reproduzíveis.

## ADR-021 — Decisões suspendem a finalização de Ritos

- Se o texto de um Rito abrir uma decisão, a engine serializa uma continuação
  em `pending_rites` e pausa a resolução intrínseca da carta. Toda a cadeia de
  escolhas e continuações da DSL termina antes da emissão do Sigilo, do
  deslocamento do Eclipse e da ida ao descarte.
- A ordem evita que gatilhos do Sigilo comprem, descartem ou movam cartas que
  ainda são opções de uma decisão anterior. A continuação faz parte do snapshot
  e sobrevive a restart/replay; não usa closure nem estado fora da engine.
- A correção altera a ordem authoritative de eventos e replays. Por isso o
  ruleset sobe para `alpha-0.4.0`; o catálogo abre a temporada `Alpha 0.4` com
  novo ID e encerra a temporada anterior na data de início da nova, preservando
  decks, partidas e definições históricas sob seus rulesets originais.
- Eventos `eclipse_shifted` passam a carregar em `p` o jogador que causou o
  movimento (`-1` apenas para o sistema), permitindo atribuição direta e
  reproduzível das métricas por carta sem inferir pelo evento anterior.
- Decisões enfileiradas ainda não foram apresentadas ao jogador. Ao serem
  promovidas, opções dependentes de mão/deck/descarte/permanentes são
  revalidadas contra o estado resultante das escolhas anteriores; opções
  obsoletas são removidas e uma decisão sem opções é pulada preservando sua
  continuação declarativa.
- Efeitos de compra que abrem escolha filtram novamente as cartas após os
  gatilhos da compra. Isso cobre efeitos substitutivos como a Recusa da Morte
  de Kaedor, que pode esvaziar a mão antes de a escolha ser apresentada.
- Cópias virtuais de Ritos usam a mesma pilha de continuações. A pilha é LIFO:
  uma cópia aninhada conclui escolha/Sigilo/deslocamento antes de o Rito físico
  externo retomar, sem criar instância ou zona para a cópia.

## ADR-022 — Rulesets versionados e LiveOps (Fase 7)

- **Engine**: `Ruleset` é o conjunto imutável {catálogo, Campeões, efeitos
  compilados} de uma versão. `CompileRuleset` é o único caminho de construção
  (o embutido passa por ele) e um registro por versão resolve `NewGame`/replay.
  Registrar a mesma versão com conteúdo diferente é rejeitado. Os globais
  (`engine.Cards` etc.) viram visões do embutido. Passivas/ultimates de
  Campeão continuam em Go: mudar comportamento de Campeão exige release do
  binário + bump de versão (apenas atributos são versionados por dados).
- **Persistência**: `ruleset_payloads` guarda o snapshot compilável (3
  documentos JSON) de cada versão; `SyncCatalog` também grava o do embutido.
  O boot do servidor compila e registra todas as versões persistidas —
  partidas históricas seguem reproduzíveis para sempre.
- **Fluxo de conteúdo**: draft de carta (novo ou alteração) → `validate`
  (schema + DSL + compilação completa do catálogo candidato) → `simulate`
  (versão efêmera `draft:<id>:sim`, headless com verificação de replay,
  descartada ao final) → `publish` (nova versão imutável, inativa) →
  `activate` (ponteiro competitivo; **rollback é ativar a versão anterior**).
  O matchmaking segue o ponteiro em tempo real; partidas em andamento não são
  afetadas.
- **Ban emergencial**: `ranked_card_bans` remove a carta da fila competitiva
  sem tocar em histórico, coleções ou decks; o lift é imediato. Um ban ativo
  por carta (índice parcial).
- **Auditoria**: toda mutação administrativa grava `admin_audit` na mesma
  transação do efeito — sem auditoria, sem mudança. Mutações são idempotentes
  por construção (publicar versão repetida/banir carta banida conflitam;
  ativar a ativa é no-op), então não exigem `Idempotency-Key`.
- **Operação de virada de versão**: ativar uma versão nova NÃO migra
  coleções/decks dos jogadores (fila passa a exigir decks da nova versão). A
  ordem operacional é publicar → conceder coleção/rotacionar temporada →
  ativar. A rotação existe desde a Fase 9: `POST /v1/admin/rulesets/{v}/rotate` (idempotente; clona apenas decks que seguem legais na nova versão).
- **Painel**: a Fase 7 entrega o plano de controle completo via API admin
  (RBAC + auditoria); a página web do painel fica para o ciclo de UI.

## ADR-023 — Postura de segurança da Fase 8

- Rate limiting em três faixas (geral por IP 240/min; autenticação 10/min por
  IP; comandos WS 120/min por conexão) via token bucket próprio com poda —
  sem dependência externa. `X-Forwarded-For` NÃO é confiado; atrás de proxy,
  o proxy reescreve `RemoteAddr`.
- API JSON com CSP `default-src 'none'`; a PWA carrega por meta as diretivas de
  recursos (`style-src 'unsafe-inline'` é exigido por React/Pixi no alpha) e
  entrega `frame-ancestors 'none'` como cabeçalho HTTP, pois navegadores
  ignoram essa diretiva em meta. Endurecer com nonces quando o pipeline de
  build ganhar suporte.
- `govulncheck` bloqueante na CI; base zerada com toolchain go1.25.12 +
  grpc 1.82.1 + x/text 0.39.0 (30 findings alcançáveis corrigidos em
  2026-08-10). Resíduo aceito: 1 aviso em módulo requerido sem chamada.
- Backup/restore é código executável com prova (`ops/backup-restore-test.sh`),
  não documentação: o processo falha ruidosamente se a restauração perder dados.
- Métricas mínimas via expvar em `/internal/metrics`; rotas `/internal/*`
  nunca expostas no edge. Alertas e runbooks em `ops/observability.md`.
- Fora do alpha (registrado): 2FA/admin hardening no provedor, detecção
  comportamental de botting, recuperação de conta.

## ADR-024 — alpha-0.5.0: modo treino, ritmo e 1ª rodada de balanceamento

**Modo treino vs bot**: partidas `mode=practice` no mesmo pipeline
authoritative (persistência, snapshots, replay, reconexão); o bot heurístico
ocupa o assento 1 com origem `bot` e RNG derivado da seed da partida; conta e
decks oficiais do bot são semeados (a conta jamais autentica). Treino ignora
bans (não é competitivo) e fica fora da telemetria PvP. A implementação achou
e corrigiu um bug real de reentrância: morte por Fadiga durante emissão de
Sigilo numa Guarda corrompia a resolução (endMatch não limpa mais s.Guard;
resolveAssault tolera contexto nulo; teste de regressão dedicado).

**Ritmo**: Fadiga em passos de 2 (2, 4, 6…) e Essência máxima 10.
Medido: média 35,2→33,0 rodadas; p95 75→59. Meta de p95 ≤ 40 segue aberta.

**Balanceamento (4 lotes, 10–20 mil partidas por iteração)**:
- Nerfs Solara: VR-014 previne 3 e custa 2; VR-017 custa 2; VR-019 custa 4;
  VR-022 previne 6 (era tudo) e custa 4; VR-013 exige Eclipse ≤ -1; VR-023
  move 2 (não seta -2); passiva de Mara vira desconto (era +1 prevenção);
  Ilyan ganha Ward 1 (era 2).
- Buffs Mirr/Cinéreo: Nyra compra 1 ao completar 3 Sigilos (era scry) e tem
  30 de Vitalidade; Oren tem 32 e o desconto de compra extra vale sempre;
  VR-025 base 3; VR-031 base 4/instead 6; VR-027 previne 3; VR-030 compra 1
  ao entrar; VR-033 previne 4; VR-034/VR-036/VR-060 custam menos; VR-050 +2
  com exílio; VR-053 Maldição 3; VR-055 causa 5.
- Resultado (100 mil, heurístico): fundo levantou de 20–30% para 24–42%
  (Nyra 20,7→32,8; Cinéreo ~30→41); Solara resiste (Mara 91,6→83,8). Duas
  hipóteses estruturais foram testadas e rejeitadas com dados: quantidade de
  prevenção (lote 4: 84,0→84,0) e velocidade de Eclipse (VR-023: 83,8).
  Conclusão honesta: a dominância Solara sob ESTE bot é sistêmica (kit
  consistente + o heurístico joga defesa melhor que combo) e exige sessão de
  design dedicada + bot melhor + dados humanos — não mais botões numéricos.

O catálogo runtime (backend/internal/engine/data/) é a fonte de verdade;
docs/design/ preserva o pacote original do alpha como referência histórica.

## ADR-025 — P1: progressão (rituais, maestria, ranked, carteira)

**Contexto.** Retenção pedia motivo para voltar (rituais diários), senso de
evolução (maestria por Campeão), stakes (rating) e economia futura
(Fragmentos do Véu), sem abrir nenhum vetor de trapaça.

**Decisão.**
- **Fonte de verdade**: progresso nasce só no battle server, derivado dos
  eventos authoritative da partida (`StatsFromEvents`), nunca do cliente. O
  recorder roda fora do goroutine da sala (cópia do estado + timeout próprio)
  e falha de progressão jamais bloqueia o fim da partida.
- **Idempotência**: `match_progress_log` (PK match_id) trava a transação —
  rituais, fragmentos (trilha em `economy_transactions`, kind
  `fragment_grant`), maestria e rating no mesmo commit; replay do recorder é
  no-op.
- **Rituais**: 3 por dia, sorteio determinístico `sha256(diaUTC|user)` sobre
  pool de 8 (verificável, sem RNG de servidor); progresso `LEAST(p+delta,
  target)`; recompensa creditada uma única vez (transição de completed
  observada na mesma query).
- **Maestria**: XP 10 base +5 PvP +15 vitória; nível custa 100 +20/nível,
  teto 50. Treino progride maestria e rituais não-PvP — quem só treina também
  sente evolução.
- **Ranked**: Elo K=32 (piso 0, delta nunca 0), `FOR UPDATE` nos dois
  ratings na transação; só PvP com dois humanos pontua; bot excluído do
  leaderboard e reservado fora de progressão.
- **Superfície**: `GET /v1/progress` (materializa o dia na primeira leitura)
  e `GET /v1/ranked/leaderboard`; Home mostra rituais/carteira/maestria/
  ranked reais; fila ganhou o botão de treino.

**Consequências.** Migração 000005 (5 tabelas); temporada "Alpha 0.5" semeada
para dar season ativa ao rating. Loop diário completo: jogar → ritual →
fragmento auditado → maestria/rating na Home. Fragmentos ainda não têm loja
(sink) — decisão futura de economia; a trilha de auditoria já nasce pronta.

## ADR-026 — Game feel como projeção dos eventos authoritative

- A camada de apresentação da batalha deriva animações e áudio somente do log
  redigido recebido por WebSocket. Ela não antecipa dano, legalidade, Sigilos,
  Posturas ou vencedor e nunca altera o estado da partida.
- O primeiro lote de eventos observado é baseline: entrada e reconexão não
  reanimam o histórico. Apenas sequências novas geram números flutuantes,
  pulsos de Ressonância, revelação de Posturas, banners, impacto e a
  cinematografia do Eclipse Total.
- PixiJS mantém um único palco por partida e interpola o Eclipse; partículas
  são decorativas e limitadas. WebAudio sintetiza efeitos curtos sem assets e
  respeita a preferência local de som.
- Toda camada visual usa `pointer-events: none`. Redução de movimento desliga
  tremor, rotação, interpolação e partículas, mas preserva mensagens estáticas
  por tempo curto. A ação continua em HTML semântico e o servidor permanece a
  única autoridade.

## ADR-027 — Ambiente local isolado por portas e variáveis próprias

- O desenvolvimento local do Nythara usa portas dedicadas: API `18080`,
  PostgreSQL `55432`, Redis `56379` e cliente `5173`. Produção continua
  configurável por ambiente.
- `make run` e as migrações locais injetam `VEURUBRO_DEV_DATABASE_URL` em vez
  de herdar silenciosamente um `DATABASE_URL` genérico do shell. Isso impede
  que um servidor do jogo leia ou migre o banco de outro projeto aberto na
  mesma máquina.
- O proxy do Vite aponta por padrão para a API dedicada. `VITE_API_TARGET`,
  `VEURUBRO_DEV_API_PORT` e as portas do Compose continuam disponíveis como
  overrides explícitos.
- Catálogo, coleção ou Campeões indisponíveis devem produzir erro visível com
  ação de reconexão; a interface nunca representa falha de infraestrutura como
  uma coleção vazia.

## ADR-028 — Identidade pública única no Alpha

- O `player_profiles.display_name` existente passa a ser o nome de usuário
  canônico. A coluna não é renomeada neste ciclo para preservar sessões,
  respostas e clientes Alpha; o cadastro recebe `username` e mantém
  `display_name` apenas como alias de compatibilidade.
- Nomes de usuário têm 2–32 caracteres e aceitam somente letras ASCII,
  números, hífen (`-`) e sublinhado (`_`). Espaços, acentos, pontuação e
  demais caracteres são recusados antes de qualquer escrita.
- Unicidade de nome é global e case-insensitive no PostgreSQL. A unicidade de
  e-mail já existe desde a migração 000001 e continua case-insensitive por
  normalização obrigatória para minúsculas.
- A migração 000006 aplica a unicidade e valida o formato para novas escritas.
  O `CHECK NOT VALID` preserva perfis Alpha antigos com nomes fora do novo
  formato, sem permitir criar outros; esses perfis podem ser renomeados numa
  futura tela de conta.

## ADR-029 — alpha-0.6.0: rodada 2 de balanceamento (gate de competitividade)

**Contexto.** Gate reprovado com Mara 85% / Oren 20% (heurístico, 20k). O
ADR-024 mandava: melhorar o instrumento antes de mais nerfs numéricos.

**Diagnóstico (medido, 20k por lote).**
1. A matriz tinha matchups degenerados (Mara 100%/99,5% contra Mirr/Varka).
2. Partidas de ~34 rodadas são corrida de Fadiga: passivas recorrentes por
   rodada compõem (desconto de Guarda da Mara ≈ 25+ Essência/partida).
3. Solara deslocava o Eclipse ~26/partida contra ~13 de Varka — todas as
   cartas Solara tinham shift intrínseco; o cabo de guerra era invencível,
   mantendo as passivas/bônus deles sempre ligados e os kits de Noite sempre
   desligados.
4. O arquétipo de compra (Oren) esvazia o próprio baralho primeiro e perdia a
   corrida de atrito que o próprio plano alonga; Nyra e Oren jogam o MESMO
   deck algorítmico — o gap entre eles era 100% kit.

**Instrumento (bot heurístico v2).** Scoring minera a DSL declarativa:
dano esperado com bônus avaliados no estado real (`evalCond` com contexto
mínimo — condições de combate leem falso, conservador), valor de Sigilo com
proximidade da tríade, plano de Eclipse por polo do kit + mão, economia de
Guarda (não desperdiça prevenção grande em arranhão), Ultimate com timing,
compra com valor decrescente conforme o baralho encolhe, mulligan por curva.
Introspecção dos championImpls (on3Sigils/onExtraDraw) no lugar de tabela por
ID. Sozinho, o bot não moveu o spread (85→85) — provou que a distorção era de
conteúdo, não só de instrumento.

**Mudanças de conteúdo/regra (todas medidas).**
- *Ritmo*: Fadiga em passos de 4 (era 2); campeões modulam o passo.
- *Eclipse*: VR-015/VR-018 sem shift intrínseco; VR-022 −1 (era −2); VR-023
  move −1/−1 (era −2/−2). Solara cai de ~26 para ~15 de deslocamento.
- *Taxas Solara*: VR-022 previne 5 (era 6); VR-020 causa 3 e ignora 1 (era
  4/2); VR-013 bônus exige Eclipse ≤ −2 (era ≤ −1).
- *Varka*: VR-044 vira 2×3 por custo 4 (era 1×3 por 3 — com 2×3 a custo 3
  Varka foi a 69%, rejeitado); VR-047 mantém 4.
- *Mirr (cartas)*: VR-031 instead 7; VR-025 bônus +2; VR-027 previne 4;
  VR-028 custa 1; VR-030 custa 2; VR-036 custa 3.
- *Kits*: Mara — desconto de Guarda só na Aurora Total, full-prevent cura 1
  (era compra; heal 2 regrediu para 73,8%, rejeitado), ultimate drena 1
  (era 2). Ilyan — Ward não acumula. Nyra — tríade compra 1 + 1 Essência
  temporária + cura 1; cópias sempre emitem Sigilo. Oren — Fadiga cresce
  2 a menos (passos 2,4,6), desconto de compra extra −2, compra extra cura 1
  (34 de Vitalidade; Nyra idem).

**Resultado (gate de 100 mil, heurístico v2, replay verificado).** Spread
[19,7–85,1] → **[38,8–66,1]**; saúde perfeita (0 crashes/loops/divergências);
iniciativa 49,81%; p50 30 rodadas. Dos seis matchups 99-100% restou um
polarizado (Mara×Nyra 96/4). Mara 66,1 fica 1,1 ponto acima do teto [35,65]:
aceito como marginal e registrado — o próximo lever é composição de precon,
não mais knobs numéricos.

**Dívidas honestas.** (1) p95 subiu para ~70 rodadas: a rodada de sustain
esticou a cauda; a meta p95 ≤ 40 segue aberta e pede lever de ritmo que não
seja Fadiga (tamanho de deck, teto de cura). (2) Números calibrados sob bot —
telemetria humana pode reordenar o topo; a régua correta continua sendo
rating (P1). (3) Precon algorítmico acopla deck ao campeão: separar força de
deck e de kit exige variedade de construção (Set 2).

## ADR-030 — Arena: patentes, códigos de deck, histórico e crônica

**Contexto.** Retenção de quem ama duelo (gênero TCG): escada com identidade,
moeda social de listas, memória das próprias partidas.

**Decisão.**
- *Patentes e títulos são derivação pura* (rating→patente, nível→título) em
  `domain/ranks.go` — zero migração; rebalancear faixas é ajuste de
  apresentação. Nomes 100% originais do universo.
- *Códigos de deck* (`VR1.` + base64url de JSON ordenado): determinísticos
  por lista; importar revalida tudo no funil do SaveDeck — o código carrega
  intenção, nunca autoridade.
- *Crônica*: a engine é determinística e o log de eventos É a partida; o
  endpoint expõe o log bruto e o cliente o transforma em história (curvas de
  Vitalidade/Eclipse, momentos). Autorização estrita: participante (ou
  admin) + partida encerrada — o log revela mão/baralho, então partida viva
  jamais vaza (testado).
- *Histórico* lê as tabelas de batalha existentes; sem estado novo.

**Consequências.** Nenhuma migração; nenhuma escrita nova. Futuro natural:
recompensas de fim de temporada por patente e replay palco-a-palco na mesa
(o log já sustenta os dois).

## ADR-031 — alpha-0.6.1: cirurgia de precon (Mara) e swaps por Campeão

**Contexto.** Após a rodada 2 (ADR-029), Mara fechou o gate de 100 mil em
66,1% — 1,1 ponto acima do teto [35,65] — com o próximo lever apontado:
composição de precon, não knobs numéricos. Nyra e Oren provaram que gêmeos de
facção jogam o MESMO deck algorítmico; a diferença Mara×Ilyan (66×54) era
kit+deck acoplados.

**Decisão.** `precon_swaps` nos dados do Campeão (`champions_alpha.json`):
mapa carta→carta aplicado cópia a cópia pelo `PreconstructedDeck`, com o
`ValidateDeck` integral como único juiz do resultado. Dados versionados no
ruleset — sem `if championID` em código, funciona com rotação e com decks de
bot semeados.

**Medições (20k por lote).**
- *4 slots (VR-022→VR-077 e VR-020→VR-076)*: Mara 66→22,7% — rejeitado. A
  força dela estava CONCENTRADA nesses slots; cortar muralha E lança colapsa
  o arquétipo (o kit já fora recalibrado para baixo na rodada 2).
- *2 slots (só VR-020→VR-076, Relógio do Eclipse)*: Mara 46,4% e o campo
  inteiro em [39,4–60,7] — aceito.

**Resultado (gate de 100 mil, replay verificado).** Todos os 10 Campeões em
**[38,9–60,4]** — gate [35,65] APROVADO com folga; iniciativa 49,94%; p50 30
rodadas; saúde perfeita (0 crashes/loops/divergências). Resta 1 matchup
polarizado (Mara×Nyra 95/5 — muralha versus combo; mesmo par de antes).
Bump para alpha-0.6.1: composição de precon é produto competitivo; goldens
regenerados (só a string de versão nos eventos muda). Relatórios em
artifacts/balance-alpha-0.6.1-100k.*.

**Consequências.** Mara e Ilyan agora são decks distintos de fato — precedente
para precons curados por Campeão no Set 2. Mara em ~46% fica abaixo do meio:
aceitável no gate; se incomodar, o ajuste fino é um alvo de swap mais brando
(ex.: VR-071), medido no mesmo harness.

## ADR-032 — alpha-0.7.0: Set 2 (50 cartas) sem tocar no gate aprovado

**Contexto.** "Mais cartas, mais combinações" com um gate de competitividade
recém-aprovado que não podia ser re-rolado.

**Decisão.**
- *Campo `set` no catálogo* (1 = núcleo, 2 = expansão, 0 = drafts do admin).
  Precons só usam o Set 1 — o gate de balanceamento continua medindo o mesmo
  produto (prova: sim de precon pós-Set 2 é BIT-IDÊNTICO ao gate alpha-0.6.1,
  delta 0.0000 em todos os campeões). O Set 2 vive no deckbuilding e na
  facção aliada (regra que já existia no ValidateDeck e agora tem cartas que
  valem o splash).
- *50 cartas 100% na DSL existente* (zero ops novas; 130/130 compiladas,
  unsupported vazio): 8 por facção + 10 Errantes, minerando counters
  (VR-092/121), ativáveis (VR-088/116/126), janelas extras (VR-106),
  reemissão/curinga de Sigilos (VR-097/101/104), indefensáveis condicionais
  (VR-086/128), retaliação (VR-120) e recorrência (VR-103). 1 Lendária:
  O Peso de Nythara.
- *Gate de saúde de expansão*: modo `-decks varied` no simulador — decks
  legais sorteados por partida sobre o catálogo inteiro (com facção aliada).
  20 mil partidas: zero crashes/loops/estados inválidos/divergências; as 50
  cartas entraram em jogo (win rate jogada 31–62%).

**Bug real encontrado e corrigido pelo gate variado** (3 jogos em 20k): uma
cadeia que compra carta DEPOIS de abrir uma decisão de topo (Guarda VR-091
abre reordenação; retaliação do VR-120 compra uma das opções) corrompia
zonas — o refresh de opções só cobria decisões na fila. Correção dupla no
engine: o Apply refresca a decisão pendente ao fim de toda cadeia, e a
resolução de reordenação ignora cartas que saíram do baralho (regressão
dedicada). O caso "loop" era um jogo finito de 168 rodadas — o gate variado
usa teto de 4000 comandos (o de 2000 é calibrado para precons).

**Consequências.** Bump alpha-0.7.0; goldens regenerados. Arte: manifesto em
docs/design/SET2_ART_MANIFEST.md no modelo do Set 1 (fallback procedural
cobre até os webp chegarem). Balanceamento fino do Set 2 virá de telemetria
humana (ranked) e do harness variado — números de estreia deliberadamente
honestos, sem staple óbvio.

## ADR-033 — Painel LiveOps web (fecha a dívida da Fase 10)

**Decisão.** Página `/salao` (admin-only; link condicional no nav) como casca
fina sobre a API da Fase 7: rulesets (ativar/rotacionar com confirmação),
bans de emergência (banir/derrubar), temporadas, telemetria, trilha de
auditoria e a Forja de drafts com editor JSON e o pipeline
validar→simular→publicar. Nenhuma lógica no cliente: o servidor segue sendo
o único juiz, e toda mutação continua auditada em transação.

**Bug real encontrado pelo primeiro consumidor.** A query de telemetria
juntava `match_players.champion_id` — coluna que nunca existiu (o campeão de
um assento vem do deck). Os testes de handler usavam fake e nenhuma
integração cobria `MatchTelemetry`; o painel quebrou na primeira carga.
Corrigido com JOIN em `decks` + teste de integração de regressão. Moral
registrada: toda query nova ganha ao menos um smoke de integração.

**De quebra.** O helper de integração foi alinhado à identidade (ADR-028):
usernames sem espaço e únicos por execução.

## ADR-034 — Recompensas de fim de temporada por patente

**Decisão.** Fechar uma temporada (efeito colateral de `POST
/v1/admin/seasons`) concede Fragmentos do Véu pela patente FINAL de cada
ranqueado da temporada fechada (Errante 20 → Voz do Eclipse 250; tabela em
`domain/ranks.go`, derivada como as patentes — ajustar é economia, não
migração). Bot fora; exige ao menos 1 partida ranqueada.

**Robustez.** Tudo na MESMA transação do fechamento, e a concessão só roda
para temporadas que TRANSICIONARAM nesta chamada (`UPDATE … RETURNING id`) —
idempotente por construção: repetir a virada não reconcede (testado). Cada
grant deixa trilha em `economy_transactions` (`source=season_reward`, payload
com temporada/patente/rating) e o fechamento registra um resumo
(`season:rewards`) na auditoria admin.

**Consequências.** O ciclo competitivo fecha: ranqueada → patente →
virada → Fragmentos. O sink de Fragmentos continua sendo a próxima decisão
de economia (loja/cosméticos), agora com mais uma torneira auditada.

## ADR-035 — Grant alpha completo acompanha novos rulesets

**Contexto.** O Alpha promete coleção competitiva completa no registro. A
chegada do Set 2 criou um ruleset com 130 cartas, mas o seed de boot só
concedia a coleção nova ao bot; contas existentes mantinham o direito
`alpha_complete` auditado e, ainda assim, viam as 50 cartas como 0/0.

**Decisão.** `SyncCatalog`, depois de persistir as definições do ruleset,
propaga cartas e Campeões ausentes para toda conta com uma transação
`collection_grant` cujo payload declare `grant=alpha_complete`. A inserção
usa as quantidades normais (2; Lendária 1) e `ON CONFLICT DO NOTHING`, na
mesma transação da sincronização. O grant original é a autorização contínua;
por isso a propagação não cria uma segunda transação econômica.

**Limite.** Contas sem esse grant não recebem itens — a futura aquisição por
fragmentos continua intacta. Quando o produto deixar o Alpha completo, a
mudança exige outro tipo de grant/ADR em vez de reinterpretar o histórico.

**Consequências.** Contas antigas e novas observam a mesma coleção completa;
reiniciar/sincronizar é idempotente, coberto por integração que remove uma
carta e um Campeão, sincroniza duas vezes e verifica restauração sem grant
duplicado.

## Achados de design para revisão de balanceamento

1. **VR-033 (Véu de Prata)**: com fases estritas, o Véu ganho na Guarda expira
   antes de qualquer Rito adversário poder ser bloqueado (Ritos precedem o
   Confronto). Sugestão: duração "até o fim da **próxima** rodada" ou mover
   proteção para outro gatilho.
2. **VR-005 (Fome Educada)**: a sequência Presa→Coroa exige um emissor de Coroa
   pós-Presa na mesma janela; no set, só VR-061/VR-002+ordem viabilizam. Ok com
   VR-061 implementada, mas a densidade de enablers é baixa em mono-Vhal.
3. Baseline alpha-0.5.0 (100 mil, heurístico): Mara 83,8 / Ilyan 77,7 no
   topo; Oren 24,0 no fundo. Próximos passos registrados no ADR-024:
   heurístico com linha combo/eclipse, sessão de design Solara/Mirr e
   telemetria humana antes de novos nerfs numéricos.
4. **Baseline alpha-0.4.0 (100 mil, heuristic × heuristic)**: iniciativa ficou
   neutra (50,038%); média 35,23 rodadas, p95 75 e máximo 155, sem loops. A
   matriz precon encontrou forte delta de arquétipo: Mara 91,61%, Ilyan 83,75%,
   enquanto Nyra 20,66% e Oren 23,96%. Os alertas de played-win-rate se
   concentram nas cartas Solara porque o precon liga inclusão ao arquétipo; não
   justificam nerf isolado sem decks variados/search bot. O próximo passe de
   balanceamento deve separar força do Campeão, composição do precon e efeito
   individual antes de alterar conteúdo.

## ADR-036 — alpha-0.8.0: catálogo fiel e Véu interativo entre rodadas

**Contexto.** A DSL já continha os números de balanceamento aprovados nos
ADRs 024 e 029, mas o texto público de 12 cartas ainda descrevia valores de
versões anteriores. Isso fazia a Arena prometer um efeito e a engine resolver
outro. Além disso, o Véu concedido por VR-033 na Guarda expirava no mesmo
Crepúsculo; como Ritos são jogados antes do Confronto, VR-067 nunca encontrava
um Véu adversário em uma partida normal e o componente anti-Rito de VR-033 não
tinha janela útil.

**Decisão.** O catálogo passa a reproduzir exatamente os números da DSL para
VR-013, VR-014, VR-020, VR-022, VR-023, VR-025, VR-027, VR-031, VR-044,
VR-050, VR-053 e VR-055. VR-033 mantém prevenção 4 e passa a dizer e conceder Véu
"até o fim da próxima rodada": o status fica ativo na rodada em que nasceu e
na seguinte, quando pode bloquear Ritos ou ser removido por VR-067, expirando
no Crepúsculo dessa rodada seguinte. A engine centraliza a consulta em um
único predicado para impedir divergência entre ações legais, resolução de
Ritos, cópias e aplicação de status.

**Versionamento e validação.** Bump para `alpha-0.8.0`. Replays conservam a
versão registrada na configuração. Testes cobrem persistência entre rodadas,
bloqueio de Rito, remoção por VR-067 e expiração. O heurístico usado pelo gate
também passa a avaliar continuações de decisões e habilidades ativadas sem um
bônus fixo artificial; isso altera somente o instrumento de medição, nunca as
regras ou o resultado autoritativo.

**Consequências.** Não existe mais carta-tech morta por ordem de fases, e o
texto que o jogador lê volta a ser contrato verificável da engine. Os números
de balanceamento posteriores a este ADR não são comparáveis diretamente aos
relatórios gerados com o heurístico anterior.

## ADR-037 — Gate competitivo v2 mede causa do término e extremos

**Contexto.** Média de duração e win rate agregado escondem os defeitos que
mais expulsam jogadores: caudas de partidas por Fadiga, Campeões sem escolha
competitiva e matchups praticamente decididos antes da primeira mão. O
relatório v1 também registrava todo nocaute apenas como `vitalidade`, embora o
log determinístico já preserve a causa real do último dano.

**Decisão.** `balance-report.v2` inclui causas de término ordenadas, com
quantidade, participação, média e p95 de rodadas. A classificação é derivada
dos eventos autoritativos (Assalto, Fadiga, Sangramento, Maldição, efeito de
carta ou habilidade), sem alterar estado, protocolo nem replay.

O fechamento competitivo exige, em 100 mil partidas por modo: saúde e replay
sem falhas; primeiro jogador entre 48–52%; p95 de duração até 40 rodadas;
Campeões entre 40–60% no campo variado; matchups variados entre 25–75%; e
nenhuma carta com amostra mínima acima de 65% ou abaixo de 35% de win rate ao
ser comprada. Precons são produto de aprendizado, portanto cada Campeão deve
ficar entre 40–60% e seus matchups entre 20–80%. Alertas de `played win rate`
continuam diagnósticos, pois jogar uma resposta naturalmente correlaciona com
o estado da partida e não prova causalidade.

**Cobertura.** Além da simulação, o gate exige teste direto das 130 cartas,
propriedades de conservação de zonas/recursos, fuzz de comandos e escolhas,
golden/replay, corrida, integração do servidor de batalha e E2E real de PvP e
treino. Toda exceção precisa aparecer no relatório final; não existe teste
silenciosamente pulado.

**Consequências.** Uma rodada de balanceamento só fecha quando os extremos e
a experiência ponta a ponta passam juntos. Alterações de heurístico ou modo de
deck ficam gravadas no relatório e invalidam comparação ingênua com baselines
anteriores.

## ADR-038 — Passe competitivo alpha-0.8.0: ritmo, Oren e Cinéreo

**Contexto.** Com o heurístico corrigido, 20 mil decks variados produziram
p95 56, 33,03% de finais por Fadiga, Oren em 62,88% e os Campeões Cinéreos em
35,05%/38,35%. Nos precons, Fadiga chegou a 50,98% e os dois Cinéreos ficaram
em 22,68%/28,10%. Oren acumulava três bônus não descritos no catálogo a cada
compra extra (desconto 2, cura e Fadiga reduzida em 2); já o pacote Cinéreo
pagava cartas, descarte e exílio por retornos tardios.

**Decisão.** A faixa de Vitalidade inicial cai para 27–30, preservando a ordem
relativa dos Campeões. Oren mantém sua identidade de leitura, mas o desconto
da primeira compra extra e a redução do passo de Fadiga passam a 1; a cura é
removida sem substituição recorrente. Nyra deixa de curar ao completar a tríade (o texto
público nunca prometeu essa cura). O passo base de Fadiga passa de 4 para 6
(6, 12, 18…), encerrando o
terceiro ciclo de um deck vazio mesmo sem dano anterior. Voren ganha Ward 1 no
primeiro autoexílio de cada rodada. Edda limita
Cinzas a 2 e reescreve uma página com 2 marcadores.

No pacote Cinéreo, VR-058 custa 3 e exila 1 para gerar 2 Essências; VR-059
custa 2; VR-060 custa 3 e concede uma fórmula para cada 3 cartas exiladas
(máximo 3). Os textos públicos dos Campeões e cartas acompanham exatamente
essas regras. São mudanças originais de ritmo/economia interna, não cópias de
outro jogo.

**Validação.** Os valores são candidatos submetidos aos gates do ADR-037; se
algum limite falhar, o mesmo ADR registra o valor final antes do fechamento.
Goldens são regenerados somente depois da última iteração aprovada.

Os precons passam a ter cirurgias por Campeão: Seris recebe seis slots de
sacrifício/pressão do Set 2; Oren recebe dois slots de seleção/compra que ligam
sua passiva e dois slots de Guarda contra a segunda janela de Assalto. As listas continuam
legais e declarativas em `precon_swaps`.

## ADR-039 — Ruptura do Véu encerra partidas de controle com clímax

**Contexto.** Vitalidade 27–30 e Fadiga 6 reduziram o p95 variado de 56 para
45 rodadas, mas os espelhos Solara/Mirr ainda chegaram a p95 50–56. Aumentar
mais a Fadiga pune compra e transforma ainda mais finais em esvaziamento de
baralho; reduzir prevenção isolada destrói a identidade desses arquétipos.

**Decisão.** A partir da rodada 25 ocorre a **Ruptura do Véu** na Preparação:
ambos perdem Vitalidade simultaneamente, ignorando Ward, em escala 1, 2, 3…
por rodada. A perda passa pelo mesmo funil de escudos e Recusa da Morte, mas a
vitória só é conferida depois de aplicar o valor aos dois lados. Em nocaute
duplo valem os critérios determinísticos já publicados no ADR-013; portanto a
ordem de emissão de eventos não concede iniciativa escondida.

**Consequências.** Controle continua podendo vencer por recursos, mas precisa
converter a vantagem antes que Nythara se rompa. O novo término é distinguido
como `ruptura_do_veu` pelo relatório v2. O início e a progressão ficam em
constantes testadas e exigem novo ruleset/ADR para mudar.

## ADR-041 — Calibração fina do RC alpha-0.8.0 (pré-commit)

**Contexto.** O RC passou todos os gates do ADR-037, mas os relatórios de 100
mil mostravam margens: Edda 40,2% colada no piso do precon, pares Mara×Varka
em 77–79 (borda do 20–80), VR-060 como pior carta dos DOIS modos mesmo após o
ADR-038, VR-044 no teto de compra do variado (62%) e cartas com alta taxa de
"morta na mão" (Relógio do Eclipse 63% — justamente o swap da Mara).

**Lotes medidos (20 mil por modo).**
- *A (rejeitado)*: Edda com VR-115+VR-116 explodiu +20,8 (86% contra Seris);
  Mara com VR-071 foi a 61; VR-044 a custo 5 só transferiu dor ao precon de
  Varka (−8) sem mover o variado — revertido.
- *B (aprendizado)*: só VR-115 ainda mantinha Edda em 61,9 — maldição barata
  e repetível é combustível demais para o kit de reescrita dela. Mara a 59
  com qualquer carta viva no slot: o 48,2 do baseline era imposto de carta
  morta, não identidade.
- *C (lição de ecossistema)*: Mara equilibrada (51,7) trocando também um muro
  (VR-014→VR-123), mas o muro dela era estrutural para o meta — a agressão
  liberada atropelou a Edda lenta (25,5, extremos 87). Precon é sistema
  acoplado; mudar um deck re-rola a mesa.
- *D (aprovado)*: Edda com VR-116 (Urna dos Nomes — motor exílio→compra, sem
  tempo de maldição) fecha o conjunto.

**Mudanças finais sobre o RC.**
- Precon Mara: `{VR-020→VR-075, VR-014→VR-123}` (slots vivos; um muro a
  menos). Precon Edda: `{VR-057→VR-116}`.
- VR-060 custa 2 (segundo ajuste; seguia a pior carta dos dois modos).
- VR-099 previne 3 (66% morta na mão; está no precon do Oren) — texto
  público acompanha (ADR-036).
- VR-044 permanece em custo 4 (hipótese de nerf testada e rejeitada).

**Resultado (gates de 100 mil por modo, replay verificado, saúde limpa).**
Precon **[44,1–54,6]** (era [40,2–57,2]) com pior par 22/78; variado
**[42,8–55,6]** com pior par 36/62. Nenhuma carta amostrada fora de 44–62% de
win rate ao comprar (limites do gate: 35/65). p95 29 e iniciativa 49,8/50,1
preservados. Relatórios oficiais em
backend/artifacts/balance-alpha-0.8.1-*-100k.*. A guarda de imutabilidade do
catálogo ("carta mudou sem novo RulesetVersion") exigiu — corretamente — o
bump para **alpha-0.8.1** (temporada semeada Alpha 0.8.1, ID …06); os gates
rodaram com regras idênticas às publicadas, apenas o rótulo de versão nos
eventos difere.

**Observação registrada, sem mudança.** Só ~24% dos duelos terminam por
Assalto (35% Ruptura, 21–29% Fadiga): dois terços dos fins são por
esgotamento. Meta de design futura: término por Assalto ≥ 35%, com lever
próprio (ex.: Ruptura amplificando dano em vez de perda passiva) — grande
demais para calibração e recém-medido; decisão pede dor humana no ranked.

## ADR-042 — Auditoria do fluxo completo (alpha-0.8.2)

**Escopo auditado com evidência executada + leitura de código.** Engine
(fases, combate ADR-004, Ruptura, duplo nocaute, Fadiga com delta por
Campeão), redação de informação oculta (mão, postura pré-reveal, deck,
opções de decisão do rival), contabilização (Elo, maestria, rituais,
Fragmentos, recompensas de temporada, patentes) e superfícies (recorder,
histórico, crônica, códigos de deck). Suíte integral verde, fuzz de
atomicidade com 1,28M execuções, DSL 130/130, gates de 100 mil × 2 modos
com saúde limpa.

**Achado 1 (contabilização, corrigido).** `StatsFromEvents` creditava TODO
`damage_dealt` ao oponente da vítima — inclusive Fadiga, Ruptura do Véu e
auto-dano de carta própria (Trono de Espinhos, Pacto do Limiar). Na era da
Ruptura (35% dos fins), o ritual "Cause 25 de dano" progredia com dano sem
autor. Corrigido: perdas de sistema não creditam; dano de carta usa o dono
determinístico da instância (`pN-…`); Sangramento/Maldição seguem creditando
o autor real. Regressão dedicada.

**Achado 2 (contrato ADR-036, corrigido).** Texto público da Nyra prometia
menos que a engine entrega (tríade também dá 1 Essência temporária; cópias
emitem Sigilo sempre, não só em Total). Como todo o balanceamento 0.8.x foi
medido com a engine assim, o TEXTO foi corrigido — zero mudança de regra. A
guarda de imutabilidade exigiu, corretamente, o bump alpha-0.8.2 (temporada
semeada …07); os relatórios alpha-0.8.1 permanecem válidos (regras
idênticas).

**Recomendações registradas (sem mudança).** (1) Desacoplar temporada de
versão de ruleset — patch de texto não deveria ciclar temporada; (2)
matchmaking é FIFO (rating não pareia; adequado ao alpha, registrar para o
beta); (3) progressão de conta = maestria + patentes (não há "nível de
conta" — decisão de produto, não bug); (4) meta de término por Assalto ≥35%
segue no ADR-041.

## ADR-043 — Precons curados: 10 Campeões, 10 decks distintos (alpha-0.8.3)

**Contexto.** Gêmeos de facção compartilhavam ~90% do precon (o construtor
algorítmico + facção define o pool); o jogador via CH-CI-01 e CH-CI-02 como
"o mesmo deck". A pedido do usuário, todo Campeão ganhou cirurgia
declarativa (`precon_swaps`) alinhada ao kit, com teste de diferenciação
(cada par de gêmeos difere em ≥6 cópias, listas sempre legais).

**Identidades finais.**
- Seris: sacrifício/pressão (082/084/085 — ADR-038). Kaedor: lastro para o
  kit de desespero (083 Muro, 084 Cripta, 087 Pacto).
- Mara: muralha viva (075 Ponte, 071 Escudo). Ilyan: Ward/aurora (095 Selo,
  090 Édito).
- Nyra: cadeias de Sigilo (097 Reflexo, 098 Estilhaço). Oren: leitura/defesa
  (099 Vidro, 078 Armadilha — ADR-038).
- Rauk: alcateia total (109 Salto, 106 Chamado). Saela: escaramuça com pele
  (108 Matilha, 107 Pele Fechada).
- Voren: exílio/limpeza (118 Escriba, 117 Rasura). Edda: motor de cinzas
  (116 Urna, 124 Peso).

**Lotes medidos (20k cada; rejeições documentadas).** A: re-rolagem geral —
Kaedor 73,7 com pares 99–100 (indefensável+bleed barato) e Edda 65,3;
B–E: qualquer slot JOGÁVEL levava Kaedor a 64–72 — o poder estava no kit,
mascarado por duas cartas fracas (mesma lição da Mara no ADR-041); F: lastro
puro deixou os pares vs controle em 86–87; G (aprovado): remover a fonte de
Sangramento (VR-009 — dano que muralha não previne) foi a alavanca exata de
despolarização.

**Resultado (gate de 100 mil, saúde limpa).** Todos os 10 em
**[45,1–58,5]**; iniciativa 50,13; p50 24, p95 29. Pares: 89 de 90 dentro de
20–80; resta **Rauk×Nyra em 81/19 — 1 ponto acima do limite**, aceito como
limítrofe (IC ±2,4pp na amostra por par) após quatro tentativas medidas de
correção falharem: Armadilha na Nyra (60,5 geral + novo par com Oren),
Mordida no Rauk (66,0; 4 violações), tech-por-poder na Nyra (38,2) e Rauk
com swap único (66,5 — remover a Pele Grossa era estrutural). Registro: a
correção limpa é kit/carta com dados humanos, não mais troca de slot. Decks
variados não usam precon — gate variado alpha-0.8.1 permanece válido.

**Consequências.** Bump alpha-0.8.3 (guarda de campeão), temporada semeada
…08. O construtor "MONTAR DECK EQUILIBRADO" e o treino herdam as listas
curadas automaticamente (fonte única `PreconstructedDeck`).

## ADR-040 — Seed operacional acompanha alpha-0.8.0

**Contexto.** A guarda de imutabilidade recusou corretamente sincronizar o
novo ruleset sobre o ID fixo da temporada Alpha 0.7. Os decks do bot também
usavam nomes iguais entre versões; a unicidade por `(user_id, name)` fazia o
seed ignorar silenciosamente os 10 decks de treino do ruleset novo.

**Decisão.** A temporada Alpha 0.8 recebe o ID fixo novo
`00000000-0000-4000-8000-000000000005`. Temporadas anteriores abertas são
encerradas antes da inserção, como já definido. Decks semeados do bot incluem
o `RulesetVersion` no nome, preservando históricos e garantindo uma lista
completa para cada versão.

**Consequências.** `SyncCatalog` continua idempotente, não reinterpreta
temporadas/decks antigos e fornece treino no mesmo ruleset do deck humano.
Integração em banco real é gate obrigatório de todo bump futuro.

## ADR-044 — Modo Confronto: pivô para duelo direto (alpha-0.9.0)

**Contexto.** Validação humana da arena alpha-0.8.x: a partida real ficou
ilegível — o jogador não entende "sua janela está aberta", Posturas
simultâneas, Eclipse e Ressonância competindo por atenção, e a técnica de
Campeões (passiva + ultimate + forma) confunde antes da primeira partida
terminar. Veredito do usuário: "totalmente não funcional… a dinâmica ficou
muito confusa". A referência de sensação pedida é o card game brasileiro de
navegador dos anos 2000 (Heróis e Vampiros / Masters of Cards — usados apenas
como contexto de gênero, IP 100% própria): energia única que é vida e custo,
turno curto de ataque → defesa reativa → magia, selos de proibição, e o
confronto resolvido visualmente no centro da mesa com a carta perdedora se
destruindo. Converge com a meta já registrada no ADR-041 (término por Assalto
≥ 35%: hoje só ~24%; dois terços dos fins são por esgotamento).

**Decisão.** Novo ruleset **alpha-0.9.0 ("Modo Confronto")** passa a ser o
modo servido (treino, casual e ranked). O ruleset legado (≤ 0.8.3) permanece
no motor exclusivamente para replay de partidas históricas — a guarda de
imutabilidade continua valendo por versão.

O fluxo de regras é metadado imutável do snapshot (`mode=confront`), não uma
comparação com o texto `alpha-0.9.0`. Versões posteriores de balanceamento,
drafts simulados e rotações herdam o modo no payload e no Postgres; assim não
são reinterpretadas acidentalmente como o motor legado de 36 cartas.

Regras do Confronto:

- **Sem Campeões como mecânica.** Viram **avatares cosméticos** (retrato e
  título na arena; zero efeito de regra). Sem Essência, Posturas, Eclipse,
  Ressonância ativa, Relíquias, Manifestações, mulligan e Ruptura do Véu.
- **Vitalidade 30 dupla função**: é a vida e o recurso. Jogar carta custa
  Vitalidade (sacrifício); custo só é pagável se deixar o jogador com ≥ 1.
  Vence quem zerar a Vitalidade inimiga (duplo nocaute segue ADR-013).
- **Deck único de 30 cartas** por usuário, apenas Assalto/Guarda/Rito do pool
  legal do modo; sem restrição de facção; cópias seguem a regra vigente
  (máx. 2, Lendária 1), com composição mínima **8 Assaltos / 8 Guardas / 4
  Ritos** e 10 slots livres. Um deck ativo por conta, com **trava de edição
  de 24h** após salvar (constante com override por env; o deck inicial gerado
  pelo sistema não trava).
- **Turno alternado**: Compra (1 inclusive no primeiro turno; mão
  limite 7 — compra excedente queima para o descarte; deck vazio = Fadiga
  crescente 2, 4, 6… sem reembaralhar) → **Assalto** (até 1 carta; se
  defensável, abre a única janela reativa do modo: o defensor joga até 1
  Guarda ou passa) → **Rito** (até 1 carta) → fim do turno (Sangramento
  resolve no início do turno do alvo; durações expiram no fim do turno do
  dono).
- **Escala e iniciativa**: cada Assalto recebe +4 de Poder sobre a base
  compilada do catálogo; o primeiro Assalto da partida recebe −2. Assim os
  dois lados agem pela primeira vez com 6 cartas, mas a iniciativa não vira
  um golpe completo grátis.
- **Resolução do confronto**: dano final = poder do Assalto − prevenção da
  Guarda (piso 0), passando pelo funil existente (Ward, Exposto, Recusa da
  Morte). O motor emite eventos dedicados de confronto — abertura, resposta,
  resolução com o lado **estilhaçado** (dano 0 → Assalto perde; dano > 0 →
  Guarda perde; sem resposta → acerto direto) — para o cliente animar o
  centro da arena. Ambas as cartas vão ao descarte normalmente.
- **Antitrava visível**: desde o turno 20 o cliente avisa que a névoa vai se
  fechar; do turno 25 em diante ambos perdem 2, 4, 6… de Vitalidade no início
  do turno. Queda simultânea empatada usa o vencedor do último confronto;
  sem confronto, conserva o desempate geral. Essa pressão substitui a cauda
  de Fadiga, não o jogo normal.
- **Pool legal por whitelist de ops da DSL**: cartas cujo efeito depende de
  sistema removido (Eclipse, Ressonância, Postura, Relíquia/Manifestação)
  ficam fora do modo via `ImplementationReport` — exclusão explícita, nunca
  silenciosa. Assaltos e Guardas cuja estatística primária é independente
  continuam legais com **adaptação explícita**: preservam Poder/Prevenção e
  retiram apenas o efeito secundário legado; catálogo, API e relatório marcam
  `adapted=true` e substituem o texto, portanto o cliente nunca promete um
  efeito que o motor não executa. Ritos não recebem fallback: ou todas as ops
  estão na whitelist ou ficam fora. Sigilos permanecem como identidade visual
  (afinidade), sem regra ativa.

**Balanceamento publicado como alpha-0.9.1.** A escala, compensação de
iniciativa, composição e antitrava alteram regra; a guarda de imutabilidade
recusou corretamente gravá-las sobre 0.9.0. Esses parâmetros agora viajam no
snapshot `confront_rules`: replays 0.9.0 mantêm Poder/compra/ritmo originais e
versões derivadas de 0.9.1 herdam a configuração nova.

**Consequências e gate fechado.** Arena reconstruída em torno da zona central
(voo, choque e estilhaço), builder com composição/trava visíveis, matchmaking
e temporadas preservados. Dois gates oficiais de **100 mil partidas**, ambos
com replay integral e saúde 0/0/0/0/0/0, foram aprovados:

- precon: iniciativa **47,98%**, média **12,40**, p95 **22**, Assalto
  **82,86%**, Avatares **49,40–50,39%**, matchups **46,40–53,20%**;
- baralhos variados legais: iniciativa **49,86%**, média **8,63**, p95 **25**,
  Assalto **83,13%**, Avatares **49,59–50,36%**, matchups **46,80–53,90%**.

O gate de iniciativa é simétrico em 47,5–52,5%; duração exige p95 ≤30 e
término por Assalto ≥35%. Played-WR sinaliza quatro Assaltos em decks variados
por viés de seleção, mas nenhum item amostrado viola o gate causal de
drawn-WR 35–65% (máximo 62,99%); esses quatro ficam marcados para telemetria
humana, não para nerf sem evidência. Relatórios em
`backend/artifacts/confront-approved-{precon,varied}-100k.{json,md}`.

## ADR-045 — Migração integral da experiência e dos rituais para o Confronto

**Contexto.** O motor e a Arena já serviam `alpha-0.9.1`, mas a validação
visual ponta a ponta encontrou superfícies ainda ensinando o ruleset legado:
landing 0.8, onboarding/tutorial de 36 cartas, Campeões com Passiva/Ultimate,
resultado/Crônica de Eclipse e três rituais diários impossíveis no modo atual
(tríade de Sigilos, Eclipse Total e permanentes). Um jogador novo recebia
instruções contraditórias antes mesmo de entrar na mesa.

**Decisão.** Toda superfície ativa passa a usar um único vocabulário:
**Avatar cosmético**, **baralho único de 30**, **Assalto → Guarda → Rito**,
Vitalidade como custo e zona central como resolução. O catálogo distingue o
pool competitivo do arquivo; afinidades/Sigilos continuam apenas como marca
visual. Resultado, replay e Crônica mostram confrontos, cartas estilhaçadas,
dano e Vitalidade, nunca métricas removidas. Importação/exportação de listas
é retirada da Arena enquanto existir a regra de um único baralho protegido,
evitando uma rota de UX que só terminaria na restrição do banco.

O pool diário preserva os cinco objetivos ainda possíveis e substitui os três
IDs legados por objetivos derivados exclusivamente de eventos authoritative:
`win_confronts_5` (vencer cinco confrontos centrais), `shatter_rival_4`
(estilhaçar quatro cartas rivais) e `play_assaults_6` (jogar seis Assaltos).
Sorteio continua determinístico por `(dia, usuário)`; estados antigos do mesmo
dia podem permanecer armazenados para auditoria, mas deixam de ser
materializados ou progredidos. Perdas da Pressão de Nythara, como Fadiga, não
contam como dano causado.

**Consequências.** Não há mudança na regra do duelo nem bump de ruleset: a
mudança corrige progressão e apresentação para a regra já publicada. Testes
de eventos cobrem atribuição de confronto/estilhaço/Assalto e garantem que
nenhum ritual ativo depende de Eclipse, Ressonância, Postura ou permanente.

## ADR-046 — Cartas iluminadas vêm da autoridade da engine

**Contexto.** No E2E visual do Modo Confronto, o cliente iluminou `VR-067`
(Pó de Vidro) durante a fase de Rito mesmo sem Véu ativo no rival. O clique
foi corretamente recusado pela engine com `requirement_not_met`, mas o
jogador já havia recebido uma promessa visual falsa. Reimplementar no cliente
custo, alvo e cada condição da DSL duplicaria regra e voltaria a divergir.

**Decisão.** `Game.LegalPlayIDs` passa a ser a única enumeração de instâncias
jogáveis na janela atual. Ela reutiliza exatamente o caminho puro já usado
pelos bots e inclui fase, turno, Vitalidade restante, alvo, Véu, sacrifício e
pré-requisitos. A visão WebSocket individual expõe `playable`; espectador não
recebe a lista e nenhuma carta da mão rival é revelada. O web ilumina e envia
somente IDs presentes nessa superfície; `Apply` permanece a validação final.

**Consequências.** UI, bots e servidor compartilham a mesma decisão sem
transferir autoridade ao cliente. Reconexão recalcula a lista a partir do
estado restaurado, requisitos inválidos aparecem desabilitados antes do
clique e a mudança não altera regra publicada, portanto não exige bump do
`alpha-0.9.1`.

## ADR-047 — Experiência “aprenda jogando” sem nova camada de regra

**Contexto.** O Modo Confronto passou nos gates de engine, balanceamento e
E2E, mas a auditoria de produto encontrou atrito entre regra simples e
apresentação ainda textual: onboarding longo, ação principal competindo com a
mesa, estatística decisiva ausente na mão, Avatar genérico, Vitalidade sem
proporção visual e controles limitados ao clique. O código de efeitos já
produzia um anúncio de turno, porém a Arena não o renderizava. Reabrir a regra
para “criar diversão” invalidaria um balanceamento aprovado sem atacar essas
causas.

**Decisão.** O Alpha 0.9.1 recebe um pacote exclusivamente de experiência:

- introdução curta em três decisões, com chamada direta ao treino;
- orientação contextual derivada do `BattleState` e de `state.playable`;
- inspeção de carta com custo, Vitalidade restante e previsão base do
  confronto — explicitamente informativa, nunca autoridade de resultado;
- controles equivalentes por mouse/toque e teclado (1–7, Espaço e Esc);
- anúncio de turno, emblema original do Avatar, barra de Vitalidade e estados
  visuais independentes de cor;
- som e resposta tátil opcionais, movimento reduzido e dicas de combate
  configuráveis;
- acesso direto ao treino e pós-partida com leitura das decisões registradas
  nos eventos authoritative.

O cliente continua enviando somente intenção. Toda enumeração de jogadas vem
da engine conforme ADR-046; previsão visual nunca substitui `Apply`, não
considera informação oculta e usa o rótulo “base” quando efeitos posteriores
podem alterar o valor. Não há alteração de schema, eventos ou regra, portanto
o ruleset permanece `alpha-0.9.1` e os relatórios de 200 mil partidas continuam
válidos.

**Consequências.** A complexidade fica disponível sob demanda: novato enxerga
uma ação, veterano usa atalhos e histórico. Preferências são locais e não
entram no replay. Critérios e fontes ficam em
`docs/design/PLAYER_EXPERIENCE_PLAN.md`; o gate exige build, race, E2E real e
partida humana completa no navegador.

## ADR-048 — Intenção tátil, paisagem sonora e leitura consultiva do baralho

**Contexto.** Depois do primeiro gate de experiência, a regra passou a ser
compreensível e a jornada completa ficou funcional, mas ainda havia atrito em
três retornos frequentes: jogar uma carta continuava parecendo um formulário,
o resultado devolvia o usuário à seleção de modo e o construtor dizia apenas
se a lista era legal. Esses pontos podem ganhar presença física e clareza sem
reabrir o ruleset balanceado.

**Decisão.** O cliente do `alpha-0.9.1` recebe uma segunda onda exclusivamente
de interação e apresentação:

- uma carta autorizada pode ser arrastada para a zona central; cruzar o limiar
  visual envia exatamente o mesmo comando `play` do clique e do atalho. O
  gesto nunca calcula legalidade, alvo, custo ou resultado;
- clique/toque direto, botão explícito, teclas 1–7, Espaço e tecnologias
  assistivas continuam caminhos equivalentes. Movimento reduzido encurta a
  apresentação, mas não remove nenhum controle;
- uma paisagem sonora sintetizada, discreta e opcional começa somente depois
  de interação do usuário e reage a turno, Guarda e perigo de Vitalidade. Ela
  é estado local de apresentação, não evento de partida nem dado de replay;
- o resultado oferece treino imediato com o mesmo baralho ativo e mantém a
  busca PvP como escolha distinta;
- o construtor mostra uma leitura consultiva e determinística da lista
  selecionada (composição, curva de custo, Poder, Prevenção e variedade de
  Ritos). O diagnóstico não completa o deck silenciosamente, não estima taxa
  de vitória e não substitui a validação do servidor.

**Consequências.** A engine e o WebSocket permanecem inalterados; todos os
comandos ainda passam por `LegalPlayIDs` e `Apply`. Falha de áudio, vibração ou
gesto nunca derruba nem bloqueia a mesa. A análise do baralho pode evoluir com
telemetria, mas qualquer recomendação que altere regra, pool ou parâmetro de
balanceamento exige novo ADR e novo gate oficial.

## ADR-049 — Treino guiado reativo e compartilhamento sem informação privada

**Contexto.** A introdução curta leva ao treino, porém termina no instante em
que a aprendizagem mais importante começa. A Arena já informa a ação aberta,
mas um estreante ainda precisa relacionar Assalto, Guarda, Rito e Vitalidade
durante uma partida real. No outro extremo da jornada, o resultado não oferece
um artefato social seguro: a Crônica completa é privada ao participante porque
seu log contém informação de mão e baralho.

**Decisão.** O cliente marca localmente uma partida de prática como **treino
guiado** quando ela nasce pelo onboarding ou pelo guia. Essa marca é o ID da
partida na sessão; não entra no banco, no comando, no replay nem na fila. Um
coach compacto deriva sua mensagem exclusivamente de `BattleState`,
`state.playable` e eventos já redigidos, acompanhando a janela atual sem exigir
uma carta específica ou bloquear passe, clique, arraste e teclado. Encerrar o
coach remove somente a marca local.

O resultado do treino guiado apresenta um checklist derivado de eventos
authoritative: declarar Assalto, responder com Guarda e usar Rito. Item não
realizado permanece visivelmente pendente; não há recompensa, progressão ou
fabricação de sucesso. O resultado também pode copiar um resumo social com
resultado, rodadas e agregados pessoais. O texto exclui ID de partida, nome do
oponente, cartas, mão, baralho, e-mail e link para a Crônica privada.

**Consequências.** A primeira partida vira tutorial hands-on usando exatamente
o mesmo bot, ruleset e servidor do treino comum. Nenhuma jogada é simulada no
cliente e nenhuma telemetria competitiva muda. Compartilhar a Crônica integral
continua proibido sem uma futura política explícita de consentimento/redação;
o resumo agregado é seguro para copiar fora do app.

## ADR-050 — Replay como projeção visual e ritmo local da Arena

**Contexto.** A tela chamada de Replay expunha somente o texto do evento atual,
uma barra e duas setas. Ela permitia percorrer o log, mas não reproduzia a
partida: não havia duelistas, Vitalidade, cartas, fases nem comparação visual.
Na Arena ao vivo, a apresentação do confronto também terminava antes que a
maioria dos jogadores pudesse ler Assalto, Guarda e resultado. Esses problemas
são de legibilidade e presença da mesa, não da engine balanceada.

**Decisão.** O replay passa a projetar, em ordem, estados visuais derivados
exclusivamente dos eventos redigidos já armazenados na sessão: rodada, fase,
Vitalidade, contadores conhecidos de mão/baralho, carta de Assalto, Guarda,
dano e estilhaço. Informação ausente no log permanece oculta; o cliente não
reexecuta efeitos nem tenta reconstruir segredos. Reprodução, pausa, avanço,
retrocesso, scrub e velocidade controlam apenas essa projeção local.

A Arena recebe um ritmo de apresentação configurável (`cinemático`, `normal`
ou `rápido`), com o modo cinematográfico como padrão. O ajuste muda duração de
voo, impacto, estilhaço e compra, mas nunca altera relógio do turno, ordem de
eventos, comando, estado da engine ou resultado. Movimento reduzido continua
prioritário. A composição da mesa pode ocupar melhor telas largas e explicitar
os dois espaços do confronto, preservando os mesmos controles e o encaixe em
viewport compacto.

**Consequências.** Um replay antigo continua tão completo quanto o seu log
redigido permite e não adquire autoridade nova. A animação pode ser pulada ou
acelerada sem mudar a partida. Não há mudança de schema, protocolo ou ruleset;
o `alpha-0.9.1` e seus gates de balanceamento permanecem válidos. O gate desta
mudança exige navegação visual do replay, reprodução temporizada, build, race,
E2E real e regressão da simulação.

## ADR-051 — Selos táticos, texto executável e comando direto na Arena

**Contexto.** O Confronto aprovado em `alpha-0.9.1` simplificou o duelo, mas a
auditoria de uso encontrou duas perdas de profundidade. Primeiro, cartas
adaptadas exibiam uma estatística genérica seguida de texto herdado de sistemas
removidos; em três Guardas de contra-Rito, isso produziu cartas legais com
Prevenção 0 e nenhum efeito. Segundo, a interface exigia selecionar uma carta,
ler uma prévia e clicar novamente em “Jogar agora”, interpondo um formulário
entre a decisão e o centro da mesa. O pedido de “proibido atacar/proibido
defender” exige regra pública e testável, não texto decorativo no cliente.

**Decisão.** O ruleset competitivo passa a `alpha-0.10.1` e introduz três
estados originais chamados **Selos de Fase**:

- **Selo do Assalto**: a próxima fase de Assalto do duelista afetado é pulada;
  ele ainda pode usar um Rito antes de entregar o turno;
- **Exposto / Selo da Guarda**: no próximo Assalto contra o duelista afetado,
  não se abre janela de Guarda. Exposto é consumido ao declarar esse Assalto e
  deixa de conceder o bônus legado de +2 de dano;
- **Selo do Rito**: a fase de Rito do atacante após o confronto atual é pulada.

As adaptações são semânticas, não tabelas silenciosas por ID. A operação
`reveal_lock_assault` torna-se Selo do Assalto no Confronto; Guardas marcadas
com `counter_rite` recebem Prevenção mínima 2 e aplicam Selo do Rito; qualquer
Rito que aplique `exposto` prepara o Selo da Guarda. Aplicação, consumo e salto
de janela usam eventos públicos e entram no replay. Os campos de duração ficam
no estado authoritative e a engine continua sendo a única fonte de jogadas
legais.

Cada carta competitiva recebe `tactical_text`, `role` e `keywords` derivados
da DSL efetivamente aceita. O texto descreve Poder/Prevenção, condições,
efeitos posteriores e Selos sem reaproveitar cláusulas removidas. Texto e regra
continuam viajando juntos no snapshot imutável do ruleset. `alpha-0.9.0` e
`alpha-0.9.1` permanecem registrados sem Selos e preservam o significado de
seus replays.

Na Arena, uma carta iluminada é jogada com um único clique/toque, arraste ou
tecla 1–7; todos enviam o mesmo comando `play`. A orientação de fase vira um
dock compacto e o passe permanece disponível como ação secundária. Quando a
engine informa zero cartas legais, o cliente avança a janela depois de uma
breve indicação, pois não existe decisão de carta a preservar. Concessão sai da
ação principal e fica no painel de ajuda. Nenhuma dessas conveniências calcula
regra no cliente.

O primeiro snapshot local `alpha-0.10.0` registrou a regra antes da última
lapidação gramatical. A guarda de imutabilidade recusou sua substituição; ele
permanece histórico e a publicação revisada usa `alpha-0.10.1`, sem apagar ou
reinterpretar o snapshot anterior.

**Consequências.** A mudança altera legalidade, timing e valor de cartas, logo
invalida o gate de balanceamento de `alpha-0.9.1`. Publicação exige testes
unitários dos três Selos, prova de imutabilidade histórica, replay determinista,
`make test-race`, E2E real, inspeção visual da Arena e novo gate oficial de 100
mil partidas. Descrições profissionais deixam de ser conteúdo editorial
solto: divergência entre texto e efeito passa a ser uma falha testável do
ruleset.

## ADR-052 — Exposto recupera pressão para neutralizar a iniciativa

**Contexto.** O primeiro gate oficial dos Selos (`alpha-0.10.1`, 100 mil
precons) foi saudável em duração, personagens, pareamentos, cartas e
determinismo, mas reprovou iniciativa: 45,85% para quem começou, abaixo da
faixa 47,5–52,5%. Alterar a penalidade global do primeiro Assalto para 1
inverteu demais o problema nos baralhos variados (56,72% em 20 mil). Dar Ward
ao jogador inicial teve o mesmo defeito (56,27%). O desvio estava concentrado
no precon, que contém duas cópias de Marca do Caçador; o pool variado não
precisava de uma compensação global.

**Decisão.** O ruleset competitivo passa a `alpha-0.10.2`. **Exposto** mantém
a proibição de Guarda definida no ADR-051 e também concede **+2 de Poder** ao
Assalto que o consome. O bônus fica em `ConfrontRulesConfig`, viaja no snapshot
e aparece no texto profissional da carta; não é uma exceção oculta por ID. O
`alpha-0.10.1` permanece registrado com bônus 0 e não é reinterpretado.

A candidata com +2 foi a única alavanca localizada que colocou simultaneamente
os dois cenários dentro da faixa em 20 mil partidas: 48,05% de iniciativa nos
precons e 49,15% nos decks variados, mantendo p95 de duração abaixo de 30 e
sem falhas de saúde. Ela também torna o ritual de preparação mais legível: o
jogador vê que o próximo golpe será mais forte e indefensável, enquanto ainda
pode responder antes de o turno voltar ao agressor.

**Consequências.** Marca do Caçador e todo efeito futuro que aplique Exposto
ganham valor; portanto os relatórios exploratórios não autorizam publicação
sozinhos. O aceite continua exigindo dois gates oficiais de 100 mil partidas,
testes de replay/imutabilidade, race, build e E2E real no `alpha-0.10.2`.

**Validação final.** Os dois gates oficiais foram aprovados. Nos precons,
iniciativa 47,87%, média 12,09 e p95 23; nos decks variados, iniciativa 49,54%,
média 8,84 e p95 25. As 200 mil partidas tiveram zero crash, loop, estado
morto/inválido, comando rejeitado ou divergência de replay. Nenhuma carta com
amostra mínima saiu da faixa 35–65% de drawn win rate. Race, vet, build, suíte
Postgres isolada, E2E PvP/treino e inspeções desktop/mobile também passaram.

## ADR-053 — Expiração no treino passa a janela, não concede a partida

**Contexto.** O relógio de 45 segundos foi criado no ADR-018 para proteger
partidas competitivas. O treino reutilizou a mesma sala authoritative e,
consequentemente, uma única janela expirada passou a conceder toda a partida ao
bot. Isso produz derrotas por tempo já nas primeiras rodadas, enquanto o
próprio cliente orienta que uma janela perdida apenas avance a fase. O
comportamento pune leitura, acessibilidade e aprendizado sem proteger um rival
humano ou rating.

**Decisão.** Em partidas `practice`, uma expiração durante o jogo envia um
comando authoritative de `pass` para o ator esperado. O comando usa a origem
persistida `timeout`, entra no replay e aciona o bot pelo mesmo pipeline de uma
ação normal. Se a engine histórica não aceitar `pass` naquela janela, a sala
mantém o fallback seguro de concessão existente. Ready timeout continua
cancelando a sala. Em PvP, os 45 segundos e a concessão server-side do ADR-018
permanecem inalterados.

**Consequências.** O treino pode continuar mesmo quando o jogador demora para
ler ou se afasta brevemente; nenhuma carta é escolhida em seu lugar. O bot
continua tomando apenas decisões do assento 1. Não há mudança de ruleset,
legalidade, cálculo, schema ou rating, mas persistência e replay passam a
conter passes com origem `timeout`. O gate exige teste de integração da sala,
race, E2E de treino e prova de que PvP ainda termina por timeout.

## ADR-054 — Duelo longo: a Guarda recebe bônus e a partida passa a durar

**Contexto.** A medição de 3.000 partidas do `alpha-0.10.2` mostrou uma partida
de **12,07 rodadas** e **28,5 comandos** — algo entre 4 e 6 minutos reais. A
causa não era a Vitalidade: era uma assimetria estrutural. Todo Assalto recebia
`PowerBonus = 4` sobre o dano da carta, e nenhuma Guarda recebia bônus algum.
Com Poder 6–7 contra Prevenção 2–4, defender nunca anulava um golpe, apenas o
adiava. Os dados confirmavam a distorção: os cinco maiores win rates do jogo
eram Assaltos (56–60%) e as Guardas ocupavam o fundo da tabela (36–46%), com
`Instinto de Recuo` e `Passo Calculado` morrendo na mão em 57% das partidas. O
espalhamento entre a melhor e a pior carta era de **24,4 pontos**. Além disso,
o jogador via cerca de 11 das 30 cartas do próprio baralho, o que esvaziava a
construção de deck.

**Decisão.** Publicar o ruleset `alpha-0.11.0`, sem tocar nas versões
anteriores, com quatro alavancas calibradas por varredura de simulação
(`cmd/calibrate`, grade de Vitalidade × bônus de Guarda × relógio de pressão ×
compensação de iniciativa, 800 a 6.000 partidas por combinação):

- `GuardBonus = 3` — espelho defensivo do `PowerBonus` existente. A Guarda
  contra-Rito é a exceção explícita: o valor dela está no Selo do Rito, e somar
  o bônus a tornaria estritamente superior às demais.
- `StartingVitality = 56` — sustenta a partida sem que o relógio a decida.
- `PressureStartTurn = 58` — a Pressão de Nythara vira rede de segurança, não
  árbitro: passou de 4,8% para 1,08% das partidas.
- `TargetMinP50Rounds = 30` e `TargetP95Rounds = 60` — o formato declara a
  faixa de duração que promete e o gate valida contra ela, com piso e teto.

A alavanca `SecondPlayerBonusDraw` existe, é testada e fica em zero: nesta
calibragem o primeiro jogador já fecha em 47,99% sem ela.

**Consequências.** Gate competitivo de **100.000 partidas aprovado** em todos os
critérios: primeiro jogador 47,99%; p50 42 rodadas; p95 55; campeões entre
49,39% e 50,57%; matchups entre 45,80% e 54,10%; nenhuma carta fora da faixa de
35–65%; 56,39% das partidas decididas por dano. A partida passou de 12,07 para
**41,53 rodadas** e de 28,5 para **93,9 comandos**. O espalhamento de win rate
entre cartas caiu de 24,4 para **3,0 pontos** e a média de cartas mortas na mão
caiu de 20,1% para 14,6%. Com ~24 turnos por jogador, o baralho de 30 cartas é
praticamente todo consumido: a construção de deck passa a importar e a fadiga
vira desfecho legítimo de 24% das partidas.

O gate de p95 deixou de ser um teto fixo de 30 rodadas e passou a ser a faixa
declarada por cada formato — a trava não foi removida, foi parametrizada, e
ganhou piso para impedir que o duelo longo encurte sem que alguém perceba.
Rulesets antigos permanecem intactos por teste, e replays históricos não mudam
de significado. Decks de jogador seguem o modelo de rotação já existente: o
`SyncCatalog` semeia um baralho legal da nova versão para cada conta.

**Limite conhecido.** O gate de decks sorteados entrega uma partida mais curta
que a dos precons. A investigação da causa e a correção estão no ADR-055.

## ADR-055 — A Guarda tem teto de vazamento e o baralho tem um terço de resposta

**Contexto.** O ADR-054 entregou 41,5 rodadas com os precons, mas o gate de
decks sorteados ficou em 23,8. A primeira hipótese registrada — densidade de
`Exposto` no pool — foi **medida e descartada**: só 3 das 63 cartas legais
aplicam o selo (4,8%) e ele é jogado mais nos precons (6,59 por partida) do que
em decks livres (1,40). A medição também mostrou que a Guarda é usada *mais* em
decks livres (0,76 por Assalto contra 0,62). Forçar Ritos não mudou nada
mensurável (28,0 contra 27,8 rodadas).

O número que explicou tudo foi o Poder médio ponderado pelas cartas realmente
jogadas: **6,40 nos precons contra 7,18 em decks livres**, com Prevenção média
de 6,0 e 5,8. O líquido por confronto defendido salta de **0,43 para 1,37** —
três vezes. Ou seja: 0,78 de Poder médio cortava a partida quase pela metade. A
causa estrutural é que um bônus **fixo** de Prevenção não acompanha uma curva de
Poder que vai de 6 a 10: sobra contra o Assalto fraco e falta contra o forte.
Com isso, a duração da partida passava a depender de quão agressivo era o
baralho do rival, e não das decisões tomadas na mesa.

**Decisão.** Duas alavancas no `alpha-0.11.0`, ambas medidas antes de adotadas:

- `GuardLeakCap = 1` — uma Guarda comprometida nunca deixa passar mais que 1 de
  dano. A regra na mesa continua sendo Poder menos Prevenção; o teto só corta a
  cauda que um bônus fixo não alcança. Quem **não** responde continua levando o
  golpe inteiro, então o Assalto segue relevante.
- `MinGuards = 10` — um terço do baralho responde, que é a forma 10/10/10 dos
  precons, isto é, a forma sobre a qual o formato foi calibrado. O mínimo
  anterior de 8 permitia baralhos que jogavam uma partida mais curta que a
  prometida.

Ritos ficaram livres: a varredura mostrou ganho dentro do ruído, e restringir a
construção de baralho sem ganho medido é custo para o jogador sem retorno.

**Consequências.** Os dois gates de 100.000 partidas aprovam. Precons vão de
41,53 para **41,94 rodadas** (94,78 comandos) com primeiro jogador em 48,27% e
campeões entre 49,46% e 50,63%. Decks sorteados vão de 23,77 para **30,10
rodadas** (70,98 comandos), primeiro jogador 51,31%, campeões entre 49,53% e
50,63%. A distância entre o baralho oficial e o baralho montado à mão cai de
1,75× para 1,39×, o que coloca a faixa realista do jogador em torno de 28 a 42
rodadas. Nenhuma carta foi alterada: identidade, texto e arte do catálogo
permanecem intactos, e rulesets anteriores continuam protegidos por teste.

## ADR-056 — Recado do Alpha: convite opcional depois do duelo

**Contexto.** O Alpha precisa de leitura de quem joga, mas a tela de resultado é
onde alguém acabou de perder uma partida de quarenta rodadas. Um formulário
obrigatório, um modal ou uma recompensa por responder transformariam ajuda
voluntária em pedágio, e enviesariam a resposta de quem só quer voltar à fila.

**Decisão.** Um bloco discreto no rodapé da coluna de próximos passos, depois
dos botões de jogar de novo. Começa fechado, em uma linha (“Deixar um recado”),
tem um × para dispensar, e o texto diz explicitamente que é opcional. Nada
bloqueia o caminho para a próxima partida. Quem responde ou dispensa não é
perguntado de novo sobre aquela partida — a marca fica em `localStorage`, por
ser preferência de quem joga e não dado de produto.

`POST /v1/feedback` grava `user_id`, `match_id` opcional, versão de regras,
mensagem e data. O identificador da partida vem do cliente, então o serviço
verifica que ela é mesmo do remetente antes de associar: sem isso, um recado
poderia ser pendurado na partida de outra pessoa. Limite de 2.000 caracteres,
resposta `204` sem corpo — sem prêmio, sem contador, sem gamificação.

**Consequências.** Nova tabela `feedback` (migração 000011) e um índice único
por `(user_id, match_id)` que faz um segundo envio virar atualização em vez de
erro na cara de quem quis ajudar. Nenhuma outra parte do produto lê essa tabela
para decidir algo: se ela ficar vazia, nada quebra. A versão de regras viaja
junto com o recado, então uma crítica de ritmo continua legível depois da
próxima rotação de ruleset.

## ADR-057 — Poderes de Avatar: construídos, medidos e ainda não publicados

**Contexto.** O Modo Confronto tinha dez Avatares puramente cosméticos. O motor
já trazia `championImpls`, `canUltimate` e `ultimate()`, mas nada disso era
usado: os poderes históricos dependiam de Essência, Eclipse e Ressonância,
sistemas que este modo removeu. Sobraram vestígios visíveis — o ícone de
Ultimate e um banner "ULTIMATE DESATADA" que nunca podia disparar — e a tela de
Avatares dizia ao jogador que a escolha não muda nada. Num jogo de duelo, o
Avatar é o gancho de identidade; dez idênticos é o maior desperdício do produto.

**Decisão.** O `alpha-0.12.0` implementa poder para cada Avatar usando apenas o
que o Confronto tem: custo em Vitalidade, janelas de Assalto e Guarda, Ward,
estilhaço e Fadiga. Os poderes são declarativos (gatilho + efeito + grandeza) e
vivem na configuração do ruleset, junto com um mecanismo novo de
`CardAdjustments` que reequilibra cartas **por versão** — antes disso, mexer
numa carta reescreveria o significado de todas as versões anteriores, já que
todos os rulesets compilam do mesmo catálogo. As duas Guardas que morriam na mão
em ~56% das partidas passam a custar zero nesta versão.

A calibragem trocou intuição por medição e derrubou três hipóteses:

- Um desconto que vale todo turno soma ~20 de Vitalidade numa partida de 44
  rodadas; o primeiro rascunho deu **82% de win rate** para um Avatar.
- **Comprar carta virou desvantagem.** O baralho é o relógio do duelo longo, e
  acelerar a compra aproxima a Fadiga: o Avatar que comprava ficou em 28%.
  Nenhum poder publicado compra cartas.
- **Cura vale pouco com a vida cheia**, então poder curativo só compensa quando
  dispara depois do dano.

**Consequências.** Os dois gates de 100.000 partidas **aprovam**. Precons:
campeões entre 46,12% e 52,28%, primeiro jogador 49,32%, 43,74 rodadas, 39,70%
das partidas decididas por dano. Decks sorteados: campeões entre 46,84% e
53,36%, primeiro jogador 47,70%, 29,63 rodadas, 67,58% por dano. O espalhamento
entre Avatares — o número que importa para a promessa de equidade — ficou em
**6,2 pontos no baralho oficial e 6,5 no baralho montado à mão**.

Chegar lá exigiu descobrir quais poderes são estáveis entre os dois modos. Só
dois gatilhos passaram: "quando seu Assalto conecta" e "quando a Vitalidade cai
até um limiar". Gatilhos ligados à frequência de Guarda ou de estilhaço variam
com a agressividade do baralho — um deles marcou 48,6% no precon e 60,2% em
deck livre, 11,6 pontos de diferença. O limiar de Vitalidade virou o dial fino
do formato: cada ponto de limiar vale cerca de meio ponto de win rate.

O preço dessa escolha é honesto: seis dos dez Avatares compartilham o arquétipo
de resiliência, variando só o limiar. Um jogo equilibrado com arquétipos
próximos vale mais que dez identidades distintas em que uma é vinte pontos mais
forte — mas ampliar esse vocabulário sem perder estabilidade é trabalho aberto.

A iniciativa exigiu um dial diferencial. Os poderes derrubaram o primeiro
jogador para 45,9% no precon e o elevaram a 52,8% em deck livre; nenhuma
penalidade de primeiro turno corrige os dois ao mesmo tempo, porque ela move
ambos na mesma direção. A carta extra para quem não abre resolve: no jogo curto
ela é ativo, no jogo longo é passivo, porque lá o baralho é o relógio. É a única
alavanca medida que empurra os dois modos em sentidos opostos.

**Nota sobre a curva de Poder.** A compressão do topo foi medida e descartada:
render +1,3 rodada custando a identidade de dez cartas é troca ruim. O teto de
vazamento da Guarda (ADR-055) já havia neutralizado a curva.

**Trabalho aberto: o pool de Ritos.** Só 9 das 63 cartas legais são Ritos, o
eixo mais fino do jogo. As 18 cartas de Rito bloqueadas foram mapeadas: seis
delas dependem apenas de escolha do jogador (`choose_discard`, `pick_top2` e
`recover_pick`, duas cartas cada) e o restante depende de Sigilos, Eclipse ou
permanentes — sistemas que este modo removeu. Destravar as seis exige levar a
máquina de decisão para o Confronto, que hoje tem executor próprio e recusa
qualquer comando fora de jogar e passar: é uma fatia vertical de engine, view,
interface, bot e replay, não um ajuste de dados.

## ADR-058 — O Modo Confronto passa a aceitar decisões do jogador

**Contexto.** O eixo mais fino do formato é o Rito: 9 cartas legais contra 28
Assaltos e 26 Guardas. O mapeamento das 18 cartas de Rito bloqueadas mostrou que
seis delas não dependem de nenhum sistema removido — dependem apenas de uma
escolha do jogador (`choose_discard`, `pick_top2` e `recover_pick`, duas cartas
cada). O motor já tinha a máquina de decisão completa e genérica: fila,
validação, eventos `decision_requested`/`decision_resolved` e resolução. O
Confronto simplesmente não a roteava — tinha executor próprio e recusava
qualquer comando fora de jogar e passar. O custo disso não é só de seis cartas:
enquanto o modo não aceitar escolha, **nenhuma carta futura pode pedir uma**.

**Decisão.** O Confronto passa a aceitar o comando de escolha, com caminho
próprio em vez de reaproveitar o executor legado — os dois fluxos permanecem
separados, como no resto do modo. Uma decisão aberta **trava a mesa**: qualquer
outra ação é recusada até ela ser respondida, porque a alternativa seria estado
ambíguo no replay. A validação exige carta oferecida, ainda na mão, sem
repetição; resposta inválida não muta estado nem emite evento. Mão vazia não
abre decisão — a continuação roda direto, senão a partida travaria esperando uma
resposta impossível.

A capacidade é versionada pela alavanca `Decisions`, e a primeira operação
aberta é `choose_discard`, que destrava `Dívida de Sangue` e `Ampola de
Fuligem` — duas cartas de lapidação de mão, exatamente o que falta ao eixo
tático do formato.

**Consequências.** Verificação em 1.500 partidas simuladas: zero crashes, zero
loops, zero estados mortos, zero comandos rejeitados e **zero divergências de
replay** — a escolha entra no log e a partida se reproduz idêntica. Testes
cobrem o efeito da escolha, a recusa de ação com decisão aberta, a recusa de
resposta inválida sem mutação e a reprodução do replay.

As duas cartas novas mudam a composição dos precons, que são montados a partir
do pool legal, e o ritmo cai de 43,74 para 31,75 rodadas. Por isso a capacidade
nasce em `alpha-0.13.0`, **registrada e verificada, mas não servida**: o produto
segue em `alpha-0.12.0`, cujos dois gates estão aprovados. Calibrar o formato
com o pool ampliado é o próximo trabalho, e agora ele é trabalho de números —
não de arquitetura.

O que ainda falta para `alpha-0.13.0` poder ser servida: a sala de batalha
precisa publicar a decisão pendente na visão do jogador e aceitar a resposta
pelo WebSocket, e a mesa precisa da interface de escolha. O bot já responde
decisões pelo caminho que o motor legado usava.

## ADR-059 — Decisões servidas: alpha-0.13.0 no ar

**Contexto.** O ADR-058 deixou a capacidade de decisão pronta e verificada na
engine, mas não servida: faltavam a sala em tempo real, a interface e a
calibragem com o pool ampliado. Sem isso, as cartas de escolha eram letra morta.

**Decisão.** O alpha-0.13.0 passa a ser a versão competitiva. Três peças:

*Sala.* O cano do WebSocket já era genérico e a visão já redigia as opções do
rival; o que faltava era o relógio: com decisão pendente, o timeout do treino
tentava passar, falhava e caía em concessão. Agora a expiração responde a
escolha com as primeiras N opções — determinística, persistida, reproduzível.
Quem perdeu a janela perdeu a escolha, não a partida. Em PvP, o timeout segue
concedendo (ADR-018). Coberto por teste de integração da sala que provoca uma
decisão de verdade e dispara a expiração, com detector de corrida.

*Interface.* A escolha é um sheet modal — o mesmo em telefone e desktop, porque
decisão trava a mesa por regra e a interface torna isso visível. Seleção exata
de N com numeração na ordem do toque, confirmação explícita (escolha
irreversível não se auto-envia), teclado e auto-passe travados enquanto a
pendência existe, e aviso no painel quando é o rival que decide.

*Calibragem.* Os dois motores de carta a custo 1 quebravam a iniciativa: o
first-player caía a 45,5% no baralho oficial. A varredura mostrou que nenhuma
alavanca global corrigia os dois modos ao mesmo tempo — a causa eram as
próprias cartas, então a correção foi nelas: custo 2 via ajuste por versão. A
penalidade de primeiro turno zera pelo mesmo motivo. Gates de 100.000: precons
**aprovado** (primeiro jogador 50,22%, p50 45, campeões 45,93%–52,70%, dano
41,97%); decks sorteados **aprovado** (47,59%, campeões 45,98%–53,21%).

**Consequências.** Publicar exigiu ensinar `choose_discard` ao gerador de texto
tático — VR-049 compilava com texto vazio e o pipeline de draft do admin
falhava por "sem texto de regras"; o guard de imutabilidade também obrigou o
motivo de exclusão histórico a permanecer byte a byte nas versões antigas.
Verificação de ponta a ponta no navegador com o servidor real: a carta abriu o
sheet com as 7 opções da mão, o clique confirmou, o descarte cresceu e a mesa
seguiu; no mesmo dia, o caminho de timeout resolveu uma decisão ao vivo sem
conceder. O treino guiado e os textos seguem válidos; a temporada renomeia
sozinha para "Alpha 0.13.0".

## ADR-060 — Diagnóstico de conexão, prova de decisões e ritmo humano

**Contexto.** O primeiro teste prolongado de uma decisão encontrou uma falha
enganosa: um consumidor que deixava de ler o fluxo da sala enchia o buffer e
era removido corretamente, mas o comando seguinte recebia
`spectator_read_only`. A causa era transporte lento; a mensagem culpava uma
permissão que o jogador possuía. A fatia de decisão também precisava provar as
bordas que mais quebram em produção — quantidade exata, repetição, reconexão,
restauração da sala, confirmação idempotente e timeout determinístico — e o
pipeline tratava toda verificação como se tivesse o mesmo custo. Por fim, a
simulação mede rodadas, mas o produto ainda não mostrava quanto uma partida
humana demora no relógio.

**Decisão.** A sala passa a guardar, durante sua própria vida, o motivo pelo
qual cada assinante saiu. Apenas uma conexão ativa em modo espectador recebe
`spectator_read_only`; uma conexão fechada recebe `connection_closed`, e um
assinante removido por não consumir atualizações recebe
`subscriber_too_slow`. Esses códigos são diagnóstico de transporte e não
alteram autoridade, sequência nem regras da partida.

A matriz de regressão cobre a decisão do pedido até o replay: quantidade
exata, IDs e opções inválidas sem mutação, duplicatas, privacidade das opções,
reconexão, restauração a partir do store, reenvio idempotente e a escolha das
primeiras N opções no timeout de treino. O sheet ganha teste de interação no
DOM e em Chromium real (ordem visual, limite, renumeração e confirmação); o
duelo contra servidor real continua no gate de ponta a ponta.

A CI fica dividida por custo: cada commit roda unidade/integração, corrida nos
pacotes authoritative, regressão web e simulação curta com replay; a agenda
noturna roda a suíte completa com detector de corrida e os dois gates de 100
mil; o despacho explícito de release roda o servidor e o duelo de ponta a
ponta, além das provas de restauração/replay. Relatórios de simulação são
artefatos, não alterações automáticas de regra.

O agregado administrativo de telemetria passa a incluir uma amostra de ritmo
PvP: duração média, p50 e p95 em segundos, rodadas média/p50/p95 e quantidade
acima de 30 minutos. A medida vem apenas de partidas finalizadas com início e
fim persistidos. Ela é observação: qualquer ajuste posterior de Vitalidade,
Fadiga, compra ou carta continua exigindo nova versão e novo ADR.

**Consequências.** Falhas operacionais deixam de parecer falhas de autorização,
e testes que não consomem o WebSocket precisam fazê-lo explicitamente. A nova
telemetria é aditiva, pode retornar amostra zero e não muda rating nem
matchmaking. A suíte rápida preserva retorno curto em pull requests; race
integral, gates caros e E2E real continuam obrigatórios no momento apropriado,
sem esconder seu custo dentro de todo commit.

## ADR-061 — Idioma é uma projeção local, nunca parte da regra

**Contexto.** O cliente nasceu em Português do Brasil e espalhou textos pela
interface, enquanto engine, snapshots e eventos persistidos usam termos e
identificadores estáveis. Traduzir esses valores authoritative faria filtros,
comandos e replays dependerem do idioma do navegador; deixar mensagens e
acessibilidade fora do catálogo produziria uma interface parcialmente
traduzida.

**Decisão.** O cliente passa a oferecer `pt-BR`, `es` e `en`. A primeira visita
respeita o idioma preferido do navegador entre os três suportados e cai em
`pt-BR`; a escolha explícita fica em `nythara-preferences`, pode ser alterada
na entrada, na navegação e em Configurações, e atualiza `html[lang]` na hora.
Português do Brasil é a fonte editorial e o fallback de qualquer texto ainda
desconhecido.

A localização é uma projeção exclusiva da apresentação. IDs de carta,
tipos/fases do protocolo, comandos, `command_log`, snapshots e eventos não são
traduzidos. A interface traduz os rótulos desses valores na borda, bem como
texto visível, alternativas, títulos, placeholders e nomes acessíveis. Nomes
próprios do universo (Nythara, Avatares e cartas) permanecem como foram
publicados; instruções mecânicas e vocabulário do jogo são localizados. Erros
conhecidos vindos da API/WebSocket são exibidos pelo `code`; mensagem de um
código desconhecido é preservada para não esconder diagnóstico.

O catálogo é local e embarcado no shell, sem serviço externo. Componentes que
apresentam conteúdo mecânico (cartas, poderes e decisões) traduzem seus dados
explicitamente; para a superfície textual já existente, uma projeção observa
novos nós e atributos do documento, guarda o texto-fonte por nó e reaplica o
idioma ativo. Isso permite migração incremental sem prender o produto a uma
biblioteca ou alterar o React state. Toda interpolação conserva os valores
authoritative, e datas/números usam o locale ativo. Testes verificam detecção,
fallback, interpolação, mudança persistida e as três versões do seletor.

**Consequências.** Trocar o idioma é imediato, funciona offline e não muda o
hash nem a reprodutibilidade de uma batalha. Novos textos entram pelo catálogo
de apresentação; novos conceitos de regra continuam exigindo versão/ADR como
antes. Localizar futuramente nomes próprios ou flavor editorial exige um
catálogo de conteúdo versionado separado, não alterações na engine.

## ADR-062 — A VPS usa edge único, redes privadas e segredos montados

**Contexto.** O repositório tinha Compose apenas para desenvolvimento, com
credencial conhecida e portas de PostgreSQL/Redis publicadas. O backend ainda
caía silenciosamente nessa URL de desenvolvimento quando `DATABASE_URL` não
existia. Atrás de proxy, o rate limiter via o endereço do proxy; confiar
livremente em `X-Forwarded-For`, por outro lado, permitiria falsificação do IP.

**Decisão.** Produção usa imagens multi-stage separadas para API e PWA, Caddy
como único edge público com HTTPS automático e PostgreSQL em rede interna sem
porta no host. Migrações são um job bloqueante antes da API. Credenciais entram
por arquivos `0600` montados como secrets; `DATABASE_URL`/`DATABASE_URL_FILE`
são mutuamente exclusivos e a ausência impede o boot. O IP encaminhado só é
aceito quando o peer TCP pertence ao CIDR exato do Caddy, e a cadeia é lida da
direita para a esquerda. `/internal/*` é bloqueado no edge.

**Consequências.** Desenvolvimento continua explícito pelo Makefile, mas nenhum
binário inicia por acidente contra credenciais de desenvolvimento. A topologia
de produção fica reproduzível em uma VPS e inclui healthchecks, limites de log,
filesystem somente leitura onde possível, backup e runbook. A primeira versão
é single-host: alta disponibilidade, PITR gerenciado, cofre externo de segredos
e coletor de telemetria continuam decisões de operação quando a carga exigir.

## ADR-063 — Mínimos de baralho são projeção do ruleset ativo

**Contexto.** O Alpha 0.11 elevou o mínimo de Guardas de 8 para 10, mas GDD,
tutorial e construtor ainda mostravam 8. O botão de salvar podia habilitar uma
composição que a engine rejeitava. Repetir o valor no cliente faria a próxima
calibragem produzir a mesma divergência.

**Decisão.** `GET /v1/rulesets/current` projeta tamanho e mínimos de Assalto,
Guarda e Rito da configuração já compilada. O construtor usa esses valores para
contador, validação e mensagem, com fallback local igual ao Alpha 0.13 apenas
para indisponibilidade transitória. GDD, tutorial e texto de Avatar são
sincronizados com 0.13; IDs, comandos e validação continuam authoritative no
servidor.

**Consequências.** A interface não oferece mais um baralho que o backend
recusa, e uma futura mudança de composição chega ao cliente junto da versão do
ruleset. Os campos são aditivos no contrato REST e não mudam engine, replay,
deck persistido ou regra publicada.

## ADR-064 — Compatibilidade de dados e atomicidade das decisões no Alpha 0.13

**Contexto.** A revisão anterior à entrada do Alpha 0.13 na branch protegida
encontrou três lacunas de compatibilidade. A migração de identidade criava a
unicidade case-insensitive sem tratar nomes Alpha já duplicados; um Rito do
Confronto avançava o turno mesmo quando seu efeito ainda esperava uma escolha;
e o recado de uma partida era etiquetado com a versão competitiva atual, não
com a versão registrada no replay. O evento de resolução do Confronto também
usava a quantidade escolhida em `n`, enquanto o contrato legado usa o ID da
decisão, e uma continuação podia causar Fadiga letal sem verificar vitória.

**Decisão.** Antes de criar o índice de nome público, a migração 000006 bloqueia
escritas concorrentes, preserva o perfil mais antigo de cada colisão por
`created_at` e `user_id` e renomeia os demais com um identificador ASCII
determinístico derivado do `user_id`. Se esse nome já estiver ocupado, um
sufixo numérico determinístico encontra o próximo livre. A operação é
deliberadamente unidirecional: o `down` remove as restrições, mas não tenta
adivinhar os nomes duplicados anteriores.

No Confronto, um Rito que abre decisão permanece em `clash` e ganha uma
continuação serializável própria. A fase, o jogador ativo e a rodada ficam
inalterados até toda a cadeia ser respondida; só então a carta é descartada, a
vitória é verificada e o próximo turno começa. Toda continuação verifica
nocaute, inclusive quando não veio de um Rito suspenso, e
`decision_resolved.n` volta a carregar o ID da decisão. O feedback associado a
uma partida herda `replay.ruleset_version`; somente recados sem partida usam a
versão competitiva corrente.

**Consequências.** Bancos Alpha com colisões passam pela migração sem perder o
perfil mais antigo e sem abrir janela para nova duplicata. Snapshots do
Confronto recebem um campo opcional para a continuação do Rito, preservando
replay e restauração durante a escolha. Testes cobrem colisão com o primeiro
nome de fallback já ocupado, turno congelado, descarte tardio, ID do evento,
Fadiga letal e atribuição histórica do feedback. Esta correção integra o mesmo
`alpha-0.13.0` porque a versão ainda não entrou em `main` nem foi publicada por
release; depois dessa entrada, qualquer mudança de comportamento continua
exigindo nova versão, conforme a política de replays imutáveis.

## ADR-065 — Recuperação de senha por token opaco e Resend

**Contexto.** O Alpha aceitava cadastro e login, mas não oferecia recuperação
de conta. Alterar senhas diretamente no banco ou enviar senha temporária em
texto puro criaria uma credencial reutilizável e não encerraria sessões
possivelmente comprometidas. A integração de e-mail também não pode alcançar
a engine determinística nem expor segredo ao cliente.

**Decisão.** `POST /v1/auth/forgot-password` responde sempre `202`, exista ou
não a conta, e permanece sob o limitador estrito de autenticação. Para uma
conta real, a aplicação gera 256 bits aleatórios, persiste somente SHA-256,
invalida pedidos anteriores e envia pelo endpoint HTTPS do Resend um link para
`PUBLIC_APP_URL/reset-password`. O link expira em 30 minutos e carrega a única
cópia do token em texto puro. `POST /v1/auth/reset-password` valida a nova
senha pelo mesmo PBKDF2-HMAC-SHA256 do cadastro e, numa transação, consome os
tokens ativos do usuário, troca o hash e revoga todas as sessões.

O Resend existe apenas em `backend/internal/mailer`; falha de envio é logada no
servidor sem mudar a resposta pública. Produção exige `RESEND_API_KEY`,
`RESEND_FROM_EMAIL`, `RESEND_WEBHOOK_SECRET` e `PUBLIC_APP_URL` pelo
`.env.production` ignorado pelo Git. Desenvolvimento pode iniciar sem o
provedor, com a recuperação explicitamente desativada. O domínio do remetente
precisa estar verificado no Resend.

O webhook `POST /v1/webhooks/resend` verifica o corpo bruto pelo envelope Svix
do Resend (`svix-id`, `svix-timestamp`, `svix-signature`), aceita somente uma
janela de cinco minutos e usa o ID assinado como chave idempotente. Persistimos
somente ID do evento, ID da mensagem, tipo e horários; destinatário, assunto e
corpo são descartados. Produção assina apenas eventos operacionais de envio,
entrega, atraso, bounce, complaint, falha e supressão.

**Consequências.** Uma URL roubada tem janela curta, uso único e não revela a
senha. Redefinir a credencial encerra inclusive sessões anteriores. A API não
confirma cadastros, o cliente nunca recebe a chave do provedor e a engine,
replays e regras de batalha permanecem intocados. Entrega de e-mail passa a
ser uma dependência operacional observável; indisponibilidade do provedor não
vira um oráculo de existência de conta. Retries do webhook não duplicam
registros e payloads adulterados ou antigos não alimentam a observabilidade.

## ADR-066 — Nível global, Lendárias por marco e pareamento por faixa

**Contexto.** O produto possuía maestria por Avatar e patentes sazonais, mas o
ADR-042 registrava explicitamente que não havia nível global. O pedido de
produto agora exige que toda conta comece no nível 1, desbloqueie Lendárias ao
progredir e não enfrente rivais muito distantes. Aceitar `level` ou XP do
cliente permitiria uma conta iniciante forjar nível 100; gravar o mesmo fim de
partida duas vezes produziria o mesmo abuso por retry. O grant `alpha_complete`
do ADR-035 também representa aquisição histórica, não autorização para ignorar
um novo gate de progressão.

**Decisão.** Esta ADR substitui a recomendação de “sem nível de conta” do
ADR-042. Toda conta, antiga ou nova, inicia com 0 XP, portanto nível 1. O banco
persiste apenas XP global; nível é uma projeção server-side com intervalo
fechado 1–50. A curva começa em 100 XP e o custo do próximo nível cresce 20;
28.420 é o teto físico de XP. Somente PvP finalizado com dois humanos concede
XP: 15 por participação e +15 por vitória, tanto na conta quanto na maestria
do Avatar. Treino, tutorial, bot e modo desconhecido concedem zero XP de conta
e zero XP de maestria; rituais não-PvP continuam independentes porque não
alteram nível. Esta regra substitui a concessão de maestria em treino definida
no ADR-025. O recorder já idempotente por `match_id` passa a creditar também a
coluna global na mesma transação. Entradas fora de 0–30 XP por jogador/partida
falham fechadas. Nenhuma rota aceita nível ou XP enviado pelo cliente.

As oito cartas já publicadas como Lendárias ganham marcos explícitos sem mudar
seus atributos de combate: VR-012/10, VR-024/15, VR-036/20, VR-048/25,
VR-060/30, VR-079/35, VR-080/40 e VR-130/50. O ledger `player_cards` preserva o
direito histórico `alpha_complete`, mas coleção, criação de deck e entrada em
partida projetam apenas itens disponíveis no nível atual. Assim não apagamos
aquisições nem reescrevemos economia passada. Deck antigo com Lendária
bloqueada perde a trava de edição na migração e é recusado em treino/PvP até o
dono substituí-la. ID Lendário futuro sem agenda explícita fica fechado no
nível 50.

O matchmaking recebe o nível derivado pela aplicação, rejeita valores fora do
intervalo e percorre a fila até o primeiro jogador do mesmo ruleset com
diferença absoluta de no máximo 5. Incompatíveis permanecem aguardando em sua
ordem relativa; a faixa não cresce com o tempo. A primeira Lendária só chega no
nível 10, portanto uma conta nível 1 — cuja faixa termina no 6 — nunca encontra
alguém que já tenha desbloqueado uma. O treino com bot continua sendo a saída
imediata para baixa população.

**Consequências.** Progressão e pareamento tornam-se regras de produto
auditáveis, mas não mudam engine, RNG, comandos, snapshots ou significado de
replays; não há bump de ruleset de combate. Dividir a fila pode aumentar a
espera, especialmente em níveis altos, e deve ser observado antes de qualquer
alteração — ampliar a faixa exige nova decisão, não timeout silencioso. As
Lendárias são obtidas apenas jogando e não são vendidas; monetização continua
cosmética conforme o GDD.

## ADR-067 — Identidade editável, senha autenticada e entrada social

**Contexto.** O perfil exibia apenas a inicial do username e não oferecia uma
forma autenticada de trocar a senha. O resumo de batalha confundia o nome do
Avatar com a identidade do jogador, fazendo o próprio usuário aparecer como
“Voren Ashhand”. Na entrada pública, a pessoa precisava criar uma conta antes
de ver cartas ou entender a mesa. Isso aumenta abandono e também incentiva
senhas novas para quem preferiria uma identidade federada.

**Decisão.** `player_profiles` passa a guardar um `avatar_id` cosmético,
validado contra os dez Avatares do ruleset competitivo. A imagem de perfil é
uma escolha entre emblemas originais já pertencentes ao jogo; não aceitamos
upload arbitrário nesta fase, evitando armazenamento de mídia pessoal,
moderação e uma nova superfície de arquivo. O username continua único e
imutável por esta tela. Resultado e replay exibem `display_name` como identidade
principal e deixam Avatar e nome do baralho como contexto de combate.

`PUT /v1/me/profile` altera somente o emblema. `PUT /v1/me/password` exige a
senha atual quando a conta já possui senha, deriva a nova pelo mesmo
PBKDF2-HMAC-SHA256 e revoga todas as sessões na mesma transação. Contas criadas
por login social podem definir a primeira senha a partir de uma sessão OAuth
válida, sem inventar uma senha temporária.

A primeira federação é Google OAuth 2.0 pelo Authorization Code Flow com
`state` e PKCE em cookies HttpOnly, `SameSite=Lax`, de cinco minutos. O backend
troca o código diretamente, consulta o `userinfo` pelo access token e usa
somente o `sub` como identidade estável. E-mail verificado pode criar uma conta;
uma conta local preexistente só é vinculada automaticamente quando o Google é
autoridade pelo endereço (`@gmail.com` ou Google Workspace). O retorno ao SPA
carrega apenas um passe aleatório de uso único e dois minutos; access/refresh
tokens do Nythara nascem somente quando esse passe é consumido. Ausência de
`GOOGLE_OAUTH_CLIENT_ID` e `GOOGLE_OAUTH_CLIENT_SECRET` desabilita o provedor
explicitamente. Nenhum segredo OAuth chega ao cliente.

A página pública passa a mostrar uma miniatura fiel da zona de confronto e
cartas reais do catálogo antes do formulário. Ela não executa engine nem
simula resultado: é uma demonstração visual e textual, enquanto toda partida
continua autenticada e server-authoritative.

**Consequências.** Perfil, topo do app e placar final compartilham a mesma
identidade persistida. Troca de senha encerra sessões esquecidas ou roubadas.
OAuth reduz a barreira de cadastro sem substituir login local e adiciona duas
tabelas pequenas (`oauth_identities` e `oauth_login_tickets`) com limpeza por
expiração/consumo. Publicar o botão Google exige cadastrar
`https://nythara.fun/v1/auth/google/callback` no console do provedor e injetar
as duas credenciais no servidor. Nada muda na engine, no ruleset ou nos
replays históricos.
