# Validação de experiência — Onda 4

Data: 2026-08-11  
Ruleset: `alpha-0.9.1`

## Escopo

- replay visual derivado do log redigido da sessão;
- play, pausa, passo, scrub, velocidade e teclado;
- cartas, fases, Vitalidade e contadores de mão/baralho no replay;
- ritmo de animação local cinematográfico, normal e rápido;
- Arena ampliada com espaços explícitos de Assalto e Guarda;
- correção da métrica mostrada na carta central para usar o valor confirmado
  pelo evento, não o valor base do catálogo.

## Verificação no navegador

Viewport de referência: 1280×720.

- Arena abriu conectada, sem scroll interno da mesa e com mão/ação principal
  visíveis;
- zona vazia mostrou `ASSALTO`, `Poder − Prevenção = dano` e `GUARDA`;
- após 850 ms de uma jogada, as duas cartas ainda estavam legíveis no centro,
  confirmando o novo ritmo cinematográfico;
- o replay abriu no topo da página e exibiu duelistas, fases, mesa e linha do
  tempo, em vez do leitor textual anterior;
- play avançou de evento 1 para 2 em 1,15 s; pausa interrompeu o avanço;
- uma compra alterou, na projeção, baralho `30 → 29` e mão `0 → 1`;
- o replay de confronto exibiu uma arte no Assalto e duas após a Guarda;
- sequência authoritative conferida:
  - Assalto `Rasgo de Presa`: Poder 4;
  - Guarda `Passo Calculado`: Prevenção 2;
  - resolução: 2 de dano;
  - Vitalidade rival: 29 após custo da Guarda e 27 após dano;
- a carta central da Arena ao vivo passou a receber o Poder/Prevenção do
  evento corrente, eliminando divergência visual com o placar central;
- configurações expõem `Cinemático · mais legível` como padrão, além de
  `Normal` e `Rápido`; movimento reduzido continua independente.

## Gates executados

- `make lint`: aprovado (`go vet ./...`);
- `make test-race`: aprovado em todos os pacotes;
- `npm run build` em `web/`: aprovado;
- `make e2e-real`: aprovado, incluindo:
  - catálogo com 130 cartas;
  - baralho com 30 cartas;
  - confronto central e carta perdedora estilhaçada;
  - mão rival oculta;
  - partida PvP encerrada naturalmente por Vitalidade em 14 rodadas;
  - treino com bot encerrado naturalmente em 12 rodadas;
  - rating ignorado no treino;
  - unicidade de e-mail/username e formato de username;
- `make sim-smoke`: 1000/1000 partidas, first-player 49,60%, média 12,42
  rodadas, p95 22 e máximo 28.

## Resultado

A onda é de apresentação e não altera engine, protocolo ou cálculo. O replay
agora funciona como reprodução visual navegável do estado que cada evento
redigido permite conhecer; nenhuma carta oculta é inferida. Arena, engine,
E2E e distribuição de partidas permaneceram consistentes.
