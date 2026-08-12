# Validação ponta a ponta do core de duelos — Alpha 0.9.1

Data do fechamento: 11 de agosto de 2026  
Ruleset validado: `alpha-0.9.1`  
Conteúdo: 130 cartas no catálogo, 62 cartas competitivas no Modo Confronto, 10 avatares cosméticos e baralho único de 30 cartas

## Veredito

**Aprovado nos gates objetivos do Modo Confronto.**

O core determinístico, o servidor de batalha, a persistência PostgreSQL, o matchmaking, o treino contra bot, as projeções privadas do WebSocket, a dedução de Vitalidade, o rating e o build de produção passaram. Os dois baselines oficiais somam 200.000 duelos completos com replay integral e terminaram sem crash, loop, estado morto, estado inválido, comando rejeitado ou divergência.

Os 10 antigos Campeões não concedem mais poderes na Alpha 0.9.1: funcionam apenas como avatares. O jogador mantém um único baralho competitivo por ruleset e só pode alterá-lo depois do período de bloqueio definido na regra publicada.

## Regra competitiva validada

- baralho com exatamente 30 cartas;
- mínimo de 8 Assaltos, 8 Guardas e 4 Ritos;
- até 2 cópias por carta, ou 1 se Lendária;
- nenhuma restrição de facção;
- 30 de Vitalidade; pagar Vitalidade nunca pode reduzir o jogador abaixo de 1;
- mão inicial de 5 cartas e compra no começo do turno, inclusive para o primeiro jogador;
- Assalto vai ao centro, o rival responde com Guarda ou passa, a diferença é resolvida e a carta perdedora estilhaça;
- as duas cartas centrais seguem para o descarte após o confronto;
- janela de Rito depois do confronto e alternância do jogador inicial;
- limite de 7 cartas na mão, queima de excesso, Fadiga e Pressão de Nythara a partir da rodada 25;
- primeiro Assalto da partida recebe `-2` de Poder; conversão competitiva de Poder recebe `+4`;
- toda decisão é comandada e validada pela engine; comando recusado permanece atômico.

## Gate oficial de balanceamento

| Métrica | Precons — 100 mil | Decks variados — 100 mil | Limite | Resultado |
|---|---:|---:|---:|---|
| Partidas concluídas | 100.000/100.000 | 100.000/100.000 | 100% | passou |
| Primeiro jogador | 47,98% | 49,86% | 47,5–52,5% | passou |
| Rodadas médias | 12,40 | 8,63 | diagnóstico | — |
| p50 de rodadas | 11 | 7 | diagnóstico | — |
| p95 de rodadas | 22 | 25 | ≤ 30 | passou |
| Máximo de rodadas | 29 | 29 | < 30 | passou |
| Faixa dos avatares | 49,40–50,39% | 49,59–50,36% | 40–60% | passou |
| Faixa dos matchups | 46,40–53,20% | 46,80–53,90% | 20–80% / 25–75% | passou |
| Cartas compradas fora de 35–65% | 0 | 0 | 0 | passou |
| Términos por Assalto | 82,86% | 83,13% | ≥ 35% | passou |
| Divergências de replay | 0 | 0 | 0 | passou |

Relatórios-fonte:

- `backend/artifacts/confront-approved-precon-100k.json` e `.md`;
- `backend/artifacts/confront-approved-varied-100k.json` e `.md`.

### Telemetria acompanhada

Nos decks variados, `VR-081`, `VR-009`, `VR-001` e `VR-025` ultrapassaram 65% quando efetivamente jogadas. Nenhuma ultrapassou o gate de 65% quando apenas comprada; o maior valor foi 62,99%. O sinal fica registrado para telemetria humana, sem evidência suficiente para um nerf antes de partidas reais.

## E2E real de rede

O cliente `backend/cmd/e2e-duel` usa apenas a superfície pública HTTP/WebSocket e executou o fluxo natural, sem encerrar por concessão.

### PvP ranqueado

- cadastro de duas contas e criação/seleção de baralhos de 30 cartas;
- matchmaking por fila, tickets e dois WebSockets autenticados;
- mãos privadas: o oponente recebeu somente contagem/versos, nunca os IDs da mão rival;
- compra depois da mão inicial confirmada;
- 20 cartas jogadas e ciclo central observado: confronto aberto, resolvido e carta perdedora estilhaçada;
- alterações de mão, deck, descarte e Vitalidade propagadas em estado autoritativo;
- término natural por Vitalidade em 17 rodadas, estado autoritativo final `4/-5`
  e apresentação limitada visualmente a `4/0`;
- rating persistido: vencedor `1016`, perdedor `984`, uma partida para cada usuário.

### Treino contra bot

- bot ocupou o segundo assento usando o mesmo ruleset `alpha-0.9.1`;
- bot comprou, atacou, respondeu e passou pelas janelas usando a mesma engine;
- humano jogou 9 cartas;
- término natural por Vitalidade em 12 rodadas;
- treino não alterou partidas nem rating ranqueado.

O mesmo roteiro confirmou unicidade de e-mail, unicidade de username sem diferenciar caixa e rejeição de username com espaços ou caracteres fora de letras, números, `-` e `_`.

## Persistência e robustez

- migrações de identidade, formato do baralho, modo do ruleset, composição mínima e baralho competitivo único aplicadas;
- PostgreSQL limpo aprovou 12/12 cenários: rejeição de baralho ilegal, save
  idempotente, snapshot/catch-up da batalha, segurança de sessão, propagação
  do catálogo, seed único, bloqueio de alteração, composição, rotação de
  ruleset e persistência de progresso/recompensas;
- inicializações repetidas do catálogo não recriam o baralho inicial nem
  quebram a restrição de um baralho por conta (regressão descoberta e coberta);
- snapshot do ruleset carrega explicitamente o modo e todos os parâmetros do Confronto, preservando replay de versões anteriores;
- `make test-race`: todos os pacotes aprovados, sem corrida de dados;
- `go vet ./...`: aprovado;
- build web TypeScript/Vite de produção: aprovado.

## Arena e interface

A Arena consome os eventos autoritativos do duelo e representa:

- mão própria ilustrada e mão rival com cartas viradas;
- pilhas de compra e descarte para os dois lados;
- Assalto viajando para o centro e Guarda respondendo do lado oposto;
- placar de Poder/Guarda, dano, impacto, partículas e estilhaçamento da carta perdedora;
- prompts curtos por fase, brilho somente nas cartas legalmente jogáveis e aviso da Pressão de Nythara;
- lista `playable` calculada pela própria engine: custo, alvo e pré-requisito
  inválido desabilitam a carta antes do clique, sem duplicar regra no web;
- Vitalidade final limitada visualmente a zero;
- construtor com salvamento sempre visível, composição `8/8/4`, contagem `30/30`, auto-montagem e bloqueio temporal.

O navegador interno percorreu a interface real autenticada. A inspeção
confirmou Home, coleção de regras atual, construtor e Arena em desktop. Um
duelo completo contra o bot terminou naturalmente em **12 turnos**, vitória
por Vitalidade **5/0**, com **22 cartas jogadas**, **20 de dano** e **7 cartas
estilhaçadas**. Foram observados voo ao centro, comparação, dano, explosão da
carta perdedora, cartas rivais viradas, pilhas e descarte. Resultado, histórico
e Crônica persistiram; dois rituais completados creditaram 90 Fragmentos. O
construtor também foi exercitado de verdade: removeu uma cópia, adicionou uma
carta substituta, salvou 30/30 e exibiu a trava de 24 horas.

Durante esse E2E, dois defeitos foram encontrados e corrigidos antes do
fechamento: sobreposição visual de dano e custo, e um Rito condicional
iluminado apesar de requisito não satisfeito. O segundo originou o ADR-046 e
teste de regressão na engine.

## Gates finais

- [x] Engine determinística, pura e reproduzível
- [x] Cálculos de ataque, guarda, dano, custo, compra, descarte, fadiga e pressão exercitados
- [x] Baralho único, composição e bloqueio validados na engine e no banco
- [x] 200 mil replays sem divergência
- [x] Matchmaking, ocultação, WebSocket e rating E2E
- [x] Treino contra bot E2E
- [x] Unicidade e formato de identidade
- [x] Race detector, vet e build web
- [x] Arena integrada aos eventos autoritativos e compilada para produção
- [x] Inspeção visual final e duelo completo em navegador local
