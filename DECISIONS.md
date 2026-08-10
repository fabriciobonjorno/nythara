# Registro de Decisões — Véu Rubro

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
- API JSON com CSP `default-src 'none'`; a PWA carrega CSP própria por meta
  (`style-src 'unsafe-inline'` é exigido por React/Pixi no alpha; endurecer
  com nonces quando o pipeline de build ganhar suporte).
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
