# Validação ponta a ponta do core de duelos — Alpha 0.8

Data do fechamento: 11 de agosto de 2026  
Ruleset validado: `alpha-0.8.0`  
Conteúdo: 130 cartas, 10 Campeões e decks de 36 cartas

## Veredito

**Aprovado como release candidate nos gates objetivos definidos pelos ADRs 037–040.**

A engine, o servidor de batalha, a persistência, o matchmaking, o treino contra bot e a Arena web atravessaram testes reais. Os dois baselines oficiais somam 200.000 duelos heurísticos com replay integral e terminaram sem crash, loop, estado inválido, comando rejeitado ou divergência determinística.

Esse fechamento demonstra saúde de engenharia e equilíbrio dentro do meta simulado. Como em qualquer TCG, telemetria de jogadores humanos ainda deve ser acompanhada após publicação; ela não invalida os gates desta versão.

## Gate oficial de balanceamento

| Métrica | Precons — 100 mil | Decks variados — 100 mil | Limite | Resultado |
|---|---:|---:|---:|---|
| Partidas concluídas | 100.000/100.000 | 100.000/100.000 | 100% | passou |
| Primeiro jogador | 49,77% | 50,05% | 48–52% | passou |
| Rodadas médias | 23,23 | 22,52 | diagnóstico | — |
| p95 de rodadas | 29 | 29 | ≤ 40 | passou |
| Faixa dos Campeões | 40,21–57,18% | 42,97–55,96% | 40–60% | passou |
| Faixa dos matchups | 20,90–77,50% | 36,20–61,90% | 20–80% / 25–75% | passou |
| Cartas compradas fora de 35–65% | 0 | 0 | 0 | passou |
| Cobertura das 130 cartas | — | 130 incluídas, compradas e jogadas | 100% | passou |
| Divergências de replay | 0 | 0 | 0 | passou |

Relatórios-fonte:

- `backend/artifacts/balance-alpha-0.8.0-precon-100k.json` e `.md`
- `backend/artifacts/balance-alpha-0.8.0-varied-100k.json` e `.md`

### Causas de término

Nos precons: 25,44% Assalto, 6,12% efeito de carta, 28,66% Fadiga, 1,86% Maldição, 35,45% Ruptura do Véu e 2,47% Sangramento.

Nos decks variados: 23,62% Assalto, 17,25% efeito de carta, 20,89% Fadiga, 2,39% Maldição, 35,24% Ruptura do Véu e 0,62% Sangramento.

A Ruptura do Véu impede partidas intermináveis a partir da rodada 25, aplica perda simultânea antes de escolher o vencedor e mantém o p95 em 29 sem favorecer a iniciativa.

### Alertas diagnósticos que não reprovam o gate

- `VR-048 Coração da Lua Feral`: 67,14% quando efetivamente jogada no baseline de precons, mas 57,31% quando comprada. A diferença é seleção contextual: custa 5 e só é baixada quando a posição já permite aproveitar seu texto.
- `VR-108 Matilha à Espreita`: 66,04% quando efetivamente jogada nos decks variados, mas 56,40% quando comprada.

Ambas permanecem dentro do gate de 35–65% pelo indicador de carta comprada, menos contaminado pela decisão do bot de só jogar uma carta cara/contextual em estados favoráveis.

## Engine e regras

- 130 definições e 130 efeitos compilados; JSON de engine e `docs/design` sincronizados.
- Validade de deck, cópias, Lendárias, facção principal, Errantes e até 12 cartas de uma facção aliada exercitados.
- Mulligan, Posturas simultâneas/ocultas, Rito, reação, Confronto, Guarda, Assalto extra, decisões, ativadas, ultimate, Eclipse, Ressonância, Ward, Sangramento, Maldição, Véu, Fadiga e Ruptura cobertos pela suíte.
- Comando rejeitado é atômico: não altera estado nem consome sequência.
- Jogo restaurado por snapshot + comandos produz o mesmo estado e o mesmo log.
- Fuzz de comandos rejeitados: 291.462 execuções em 10 segundos sem falha.
- `go test -race -count=1 ./...`: todos os pacotes aprovados, sem corrida de dados.
- `go vet ./...`: aprovado.

## Persistência e servidor de batalha

A suíte de integração em PostgreSQL limpo passou 9/9:

1. rejeição de deck ilegal;
2. salvamento legal e idempotente;
3. snapshot/catch-up da batalha;
4. reutilização e revogação de refresh token;
5. propagação do catálogo e concessões Alpha;
6. rotação de ruleset;
7. progresso/rating da partida;
8. consulta de telemetria;
9. recompensa de temporada idempotente.

O seed operacional foi versionado para Alpha 0.8: temporada, decks do bot e payload imutável não colidem com versões anteriores.

## E2E real de rede

O cliente `backend/cmd/e2e-duel` usa somente HTTP e WebSocket públicos e respeita o limite de 120 comandos/minuto.

### PvP natural

- duas contas reais, dois decks e matchmaking em fila;
- tickets e dois WebSockets;
- mão do oponente e Postura comprometida ocultas;
- compra após a abertura e 32 cartas jogadas;
- partida natural em 17 rodadas, por Vitalidade (`27/28 → 6/0`);
- rating persistido: vencedor `1016`, perdedor `984`.

### Treino natural contra bot

- bot no assento 1 usando o mesmo servidor e ruleset `alpha-0.8.0`;
- bot jogou cartas e respondeu às janelas automaticamente;
- humano jogou 20 cartas;
- partida natural em 22 rodadas, por Vitalidade (`27/28 → -2/12`);
- treino não alterou partidas nem rating ranqueado.

O mesmo E2E também confirmou: e-mail único, username único sem diferenciar caixa e username limitado a letras, números, `-` e `_`, sem espaços.

## Arena web

Fluxo percorrido num navegador real:

- cadastro com validação nativa de username inválido;
- coleção carregou 130 cartas com custo, facção, tipo, raridade, texto, Sigilo, Eclipse e quantidade;
- existem exatamente 130 artes, de `/card-art/vr-001.webp` a `/card-art/vr-130.webp`;
- construtor selecionou Seris, montou 36 cartas, habilitou o botão e salvou o deck com confirmação da engine;
- fila exibiu o deck como `36 cartas · alpha-0.8.0`;
- treino abriu a mesa com cinco cartas próprias ilustradas e a mão rival somente como versos ocultos;
- mulligan, compra, Posturas, Rito, decisão obrigatória e histórico responderam a cliques;
- `Dívida de Sangue` mostrou o cálculo esperado: Essência `3 → 2`, Vitalidade `27 → 25`, duas compras, descarte obrigatório, Sigilo Coroa e Eclipse `0 → +1`;
- timeout de ação encerrou e persistiu o resultado quando o relógio chegou a zero.

Para impedir que coleção/construtor parecessem travados, a interface agora renderiza lotes de cartas e oferece “Carregar/Mostrar mais”; nenhum conteúdo foi removido. O resumo do deck mantém o botão de salvar visível e a landing page identifica o ruleset 0.8.

## Correções encontradas durante a validação

- decisão pendente sem opções depois da mão inteira de Kaedor;
- divergências entre texto público e DSL em 13 cartas;
- duração incorreta do Véu e alvo temporal de `VR-067`;
- bots sem avaliar continuações, ativações e custos contextuais;
- ritmo longo e excesso de partidas por Fadiga;
- polarização de Campeões/precons;
- ausência de causa de término e gate executável nos relatórios;
- seed de temporada/decks do bot colidindo entre rulesets;
- E2E antigo encerrando cedo por concessão;
- cliente de teste excedendo a proteção antispam;
- coleção/construtor renderizando cartas demais de uma só vez;
- selo visual ainda indicando ruleset 0.7.

## Gates finais

- [x] Engine determinística, pura e reproduzível
- [x] Todas as regras e cálculos críticos exercitados
- [x] 130 cartas exercitadas em decks variados
- [x] 10 Campeões dentro da faixa
- [x] 200 mil replays sem divergência
- [x] PostgreSQL real 9/9
- [x] Matchmaking, WebSocket, ocultação e rating E2E
- [x] Treino contra bot E2E
- [x] Arena, compra, mão virada, decisões e salvamento de deck
- [x] Race detector, vet e build web

