# Validação de experiência — Nythara Alpha 0.9.1

**Data:** 2026-08-11  
**Escopo:** atração, clareza, fluidez e aprendizado; regras e balanceamento do
Modo Confronto preservados.

## Resultado

**Aprovado.** Um jogador novo cria a conta, entende a regra em três decisões,
abre o treino diretamente e completa o duelo usando a mesma superfície
autoritativa do PvP. A mesa inteira cabe em 1280×720 sem rolagem.

## Pesquisa aplicada

- [Masters of Cards — Deck de cartas](https://www.mastersofcards.com.br/ojogo/deck-de-cartas/29): deck de 30 e identidade por composição.
- [Masters of Cards — Card Game](https://www.mastersofcards.com.br/ojogo/card-game/2): ataque, defesa reativa e magia como vocabulário imediato.
- [Memórias de Heróis e Vampiros](https://www.reddit.com/r/Nostalgia_Br/comments/1k9vxni/her%C3%B3is_e_vampiros/): coleção, duelo social e legibilidade da mesa como memória recorrente.
- [Riot Games — Bringing Features to Life in Legends of Runeterra](https://www.riotgames.com/en/news/bringing-features-life-legends-runeterra): experimentação, consistência e progressão que incentiva decks novos.
- [Google — Playables design best practices](https://developers.google.com/youtube/gaming/playables/certification/best_practices_design): tutorial curto/contextual, estados de botão, teclado, contraste, haptics e fim inequívoco.
- [Marvel — guia inicial de Marvel Snap](https://www.marvel.com/articles/games/a-starter-guide-on-how-to-play-marvel-snap): partidas curtas, retorno rápido e feedback de progressão.

Nada de propriedade intelectual dessas obras foi copiado. O pacote usa apenas
princípios de gênero e ergonomia; conteúdo, arte, interface e implementação
são originais.

## Entregas validadas

- onboarding por usuário, reduzido de cinco para três telas;
- diagrama animado próprio de Assalto 7 − Guarda 4 = 3 de dano;
- botão “Treinar agora” encerra o onboarding abrindo uma partida real;
- “Treino instantâneo” na Home e alternativa de bot cedo na fila;
- fases com concluído/atual/futuro, sem duas fases acesas ao mesmo tempo;
- banner de turno efetivamente renderizado;
- emblema original do Avatar e barra proporcional de Vitalidade;
- mão com custo, tipo, Poder/Prevenção/Efeito e tecla 1–7;
- prévia de jogada com custo, Vitalidade restante e resultado base;
- prévia/botão somente para IDs presentes em `state.playable`;
- Espaço passa a fase, H abre ajuda e Esc fecha painéis;
- compra anima a entrada da nova carta;
- som sintetizado e vibração opcional, além de movimento reduzido, alto
  contraste, texto ampliado e dicas configuráveis;
- histórico ativo sem nomes internos em inglês;
- pós-partida explica timeout, custo, prevenção ou golpe decisivo e liga a
  Crônica.

## Evidência no navegador real

### Conta nova e primeiro treino

1. Conta criada pela interface com username válido e e-mail único.
2. O onboarding apareceu para a conta nova mesmo no mesmo dispositivo de uma
   conta antiga — regressão encontrada e corrigida (antes era global por
   dispositivo).
3. Três telas completas e “Treinar agora” abriu WebSocket real.
4. O primeiro estado acionável foi uma Guarda: Assalto rival no centro,
   previsão `Prevenção 4 / dano base 1`, custo `−1` e Vitalidade restante 29.

### Controles e duelo completo

- tecla `4` jogou `Vidro de Retorno` na janela legal;
- o confronto resolveu bloqueado, a carta foi ao descarte e o turno avançou;
- partida natural concluída em **7 turnos**, com **9 decisões** do usuário;
- dano final encerrou a Vitalidade, sem timeout, erro de comando ou estado
  divergente;
- viewport 1280×720: `innerHeight=720`, `document.scrollHeight=720`,
  `body.scrollHeight=720`, Arena com `overflow=hidden` — mão integralmente
  visível.

## E2E HTTP/WebSocket

`make e2e-real` no código final:

- catálogo: 130 cartas;
- deck: 30 cartas, ruleset `alpha-0.9.1`;
- PvP: término natural por Vitalidade em 11 rodadas, 19 cartas jogadas,
  confronto central e estilhaço presentes, mão rival oculta, rating 1016/984;
- treino: término natural por Vitalidade em 15 rodadas, bot e humano jogaram,
  rating ignorado corretamente;
- e-mail único, username case-insensitive único e formato de username
  validados.

## Gates técnicos

- `make test-race`: aprovado em todos os pacotes;
- `go vet ./...`: aprovado;
- `npm run build`: TypeScript e Vite aprovados, 99 módulos;
- `git diff --check`: aprovado;
- regra, catálogo e balanceamento não mudaram; os dois relatórios oficiais de
  100 mil partidas do Alpha 0.9.1 permanecem válidos.

## Limites honestos e próximas medições

Este gate prova comportamento e legibilidade funcional, não retenção de
mercado. O próximo ciclo deve medir jogadores reais: tempo até primeira carta,
abandono no onboarding, hesitação por fase, uso de dicas/atalhos, revanche e
retenção D1/D7. Campanha, música adaptativa, vozes e cosméticos profissionais
pedem conteúdo e decisão de produto; não foram simulados como se já
existissem.
