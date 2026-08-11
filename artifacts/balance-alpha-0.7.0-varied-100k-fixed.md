# Relatório de balanceamento — alpha-0.7.0

Baseline: **100000 partidas**, `heuristic` vs `heuristic`, seed base `1`, replay integral `true`.

## Saúde do gate

| concluídas | crashes | loops | estados mortos | estados inválidos | comandos rejeitados | divergências |
|---:|---:|---:|---:|---:|---:|---:|
| 100000 | 0 | 0 | 0 | 0 | 0 | 0 |

## Ritmo e iniciativa

- Rodadas: média **38.99**, p50 **30**, p95 **96**, máximo **186**.
- Comandos: média **425.15**, p50 **324**, p95 **1086**, máximo **2058**.
- Primeiro jogador: **49.96%** (49959/100000).

## Win rate por Campeão

| Campeão | Partidas | Vitórias | Win rate |
|---|---:|---:|---:|
| Voren Ashhand, Mestre da Retorta (`CH-CI-01`) | 20000 | 8306 | 41.53% |
| Edda, Escriba das Cinzas (`CH-CI-02`) | 20000 | 8987 | 44.94% |
| Nyra dos Sete Reflexos (`CH-MI-01`) | 20000 | 10214 | 51.07% |
| Oren, Leitor do Véu (`CH-MI-02`) | 20000 | 12036 | 60.18% |
| Mara Vale, Lâmina do Primeiro Sol (`CH-SO-01`) | 20000 | 9788 | 48.94% |
| Ilyan, Portador da Cicatriz Dourada (`CH-SO-02`) | 20000 | 9333 | 46.66% |
| Rauk Fenclaw, Alfa da Fenda (`CH-VA-01`) | 20000 | 10299 | 51.50% |
| Saela, Pele-de-Lua (`CH-VA-02`) | 20000 | 10097 | 50.48% |
| Seris Vhal, Herdeira Carmesim (`CH-VH-01`) | 20000 | 9750 | 48.75% |
| Kaedor, o Sem-Pulso (`CH-VH-02`) | 20000 | 11190 | 55.95% |

## Sinais de cartas dominantes

Candidatas com amostra mínima, ordenadas por played win rate; correlação não implica causalidade.

| Carta | Jogos em que foi usada | Played WR | Comprada WR | Jogadas | Rodada média | Dead in hand | Alerta |
|---|---:|---:|---:|---:|---:|---:|:---:|
| Trono de Espinhos (`VR-088`) | 10473 | 69.47% | 55.21% | 10656 | 5.85 | 21.78% | ⚠ |
| Matilha à Espreita (`VR-108`) | 12745 | 66.61% | 53.78% | 13208 | 5.01 | 8.79% | ⚠ |
| Frenesi de Três Batidas (`VR-044`) | 21515 | 64.21% | 54.89% | 30633 | 15.06 | 0.76% | — |
| Soberania Carmesim (`VR-012`) | 11821 | 63.72% | 57.04% | 17717 | 15.75 | 1.31% | — |
| Refração Dolorosa (`VR-025`) | 21350 | 63.66% | 55.47% | 36222 | 17.52 | 0.66% | — |
| Alcateia Inteira (`VR-112`) | 21695 | 63.63% | 55.00% | 31775 | 15.70 | 0.96% | — |
| Carta que Não Existia (`VR-026`) | 19613 | 63.42% | 53.89% | 33806 | 18.60 | 0.48% | — |
| Reflexo Emprestado (`VR-097`) | 19179 | 63.05% | 53.73% | 32981 | 19.16 | 0.38% | — |
| Cão sem Sombra (`VR-073`) | 47187 | 63.04% | 51.83% | 49217 | 4.70 | 6.48% | — |
| Cheiro de Medo (`VR-039`) | 18348 | 62.63% | 51.73% | 25491 | 14.35 | 0.57% | — |
| Mordida Dupla (`VR-105`) | 20077 | 61.65% | 52.30% | 28238 | 13.91 | 0.84% | — |
| Rasgo de Presa (`VR-037`) | 19872 | 61.46% | 51.64% | 27974 | 13.92 | 0.83% | — |
| Juízo do Primeiro Sol (`VR-096`) | 22492 | 61.40% | 53.26% | 38283 | 19.47 | 1.08% | — |
| Marca do Caçador (`VR-015`) | 19102 | 61.17% | 50.36% | 29751 | 17.49 | 0.44% | — |
| Caçada em Espiral (`VR-047`) | 21303 | 61.02% | 52.83% | 30790 | 15.64 | 1.02% | — |

## Alertas

- VR-088 excedeu 65% de played win rate
- VR-108 excedeu 65% de played win rate

## Interpretação

Este baseline usa decks preconstruídos determinísticos sem facção aliada. Use a matriz de matchups e os sinais por carta para formular hipóteses; mudanças de regra exigem ADR, bump de ruleset e nova simulação.
