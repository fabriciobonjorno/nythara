# Relatório de balanceamento — alpha-0.7.0

Baseline: **100000 partidas**, `heuristic` vs `heuristic`, seed base `1`, replay integral `true`.

## Saúde do gate

| concluídas | crashes | loops | estados mortos | estados inválidos | comandos rejeitados | divergências |
|---:|---:|---:|---:|---:|---:|---:|
| 100000 | 0 | 0 | 0 | 0 | 0 | 0 |

## Ritmo e iniciativa

- Rodadas: média **36.19**, p50 **30**, p95 **70**, máximo **110**.
- Comandos: média **356.84**, p50 **313**, p95 **666**, máximo **1051**.
- Primeiro jogador: **49.94%** (49942/100000).

## Win rate por Campeão

| Campeão | Partidas | Vitórias | Win rate |
|---|---:|---:|---:|
| Voren Ashhand, Mestre da Retorta (`CH-CI-01`) | 20000 | 8746 | 43.73% |
| Edda, Escriba das Cinzas (`CH-CI-02`) | 20000 | 9119 | 45.59% |
| Nyra dos Sete Reflexos (`CH-MI-01`) | 20000 | 7780 | 38.90% |
| Oren, Leitor do Véu (`CH-MI-02`) | 20000 | 10511 | 52.55% |
| Mara Vale, Lâmina do Primeiro Sol (`CH-SO-01`) | 20000 | 9409 | 47.04% |
| Ilyan, Portador da Cicatriz Dourada (`CH-SO-02`) | 20000 | 10999 | 55.00% |
| Rauk Fenclaw, Alfa da Fenda (`CH-VA-01`) | 20000 | 12081 | 60.41% |
| Saela, Pele-de-Lua (`CH-VA-02`) | 20000 | 11996 | 59.98% |
| Seris Vhal, Herdeira Carmesim (`CH-VH-01`) | 20000 | 8949 | 44.75% |
| Kaedor, o Sem-Pulso (`CH-VH-02`) | 20000 | 10410 | 52.05% |

## Sinais de cartas dominantes

Candidatas com amostra mínima, ordenadas por played win rate; correlação não implica causalidade.

| Carta | Jogos em que foi usada | Played WR | Comprada WR | Jogadas | Rodada média | Dead in hand | Alerta |
|---|---:|---:|---:|---:|---:|---:|:---:|
| Coração da Lua Feral (`VR-048`) | 7853 | 71.48% | 60.07% | 7853 | 6.44 | 60.45% | ⚠ |
| Frenesi de Três Batidas (`VR-044`) | 38971 | 61.43% | 60.72% | 94263 | 18.15 | 0.88% | — |
| Caçada em Espiral (`VR-047`) | 38771 | 61.00% | 60.53% | 92998 | 18.00 | 0.95% | — |
| Salto sobre a Fogueira (`VR-041`) | 38913 | 60.54% | 60.39% | 92348 | 16.79 | 1.00% | — |
| Rasgo de Presa (`VR-037`) | 38977 | 60.42% | 60.30% | 95856 | 16.70 | 1.16% | — |
| Cheiro de Medo (`VR-039`) | 38923 | 60.16% | 60.19% | 95264 | 16.51 | 0.81% | — |
| Pele Grossa (`VR-042`) | 37183 | 59.86% | 60.37% | 77029 | 18.54 | 10.06% | — |
| Instinto de Recuo (`VR-038`) | 37793 | 59.81% | 60.21% | 80264 | 17.46 | 7.29% | — |
| Uivo que Rasga Selos (`VR-046`) | 38695 | 59.80% | 59.91% | 91453 | 16.76 | 0.38% | — |
| Corredora do Vale Oco (`VR-043`) | 38753 | 59.64% | 59.77% | 70779 | 10.95 | 0.11% | — |
| Lua no Sangue (`VR-045`) | 38629 | 59.58% | 59.79% | 91398 | 16.50 | 0.32% | — |
| Totem de Osso Lunar (`VR-040`) | 22670 | 58.58% | 59.65% | 25717 | 4.38 | 58.18% | — |
| Flecha de Vidro Solar (`VR-020`) | 19950 | 55.03% | 55.03% | 70399 | 27.35 | 0.71% | — |
| Sentença do Meio-Dia (`VR-024`) | 38052 | 51.91% | 51.21% | 71693 | 29.79 | 0.70% | — |
| Soberania Carmesim (`VR-012`) | 36920 | 51.86% | 49.96% | 75794 | 14.08 | 7.02% | — |

## Alertas

- VR-048 excedeu 65% de played win rate

## Interpretação

Este baseline usa decks preconstruídos determinísticos sem facção aliada. Use a matriz de matchups e os sinais por carta para formular hipóteses; mudanças de regra exigem ADR, bump de ruleset e nova simulação.
