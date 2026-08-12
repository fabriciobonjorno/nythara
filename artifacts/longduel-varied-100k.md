# Relatório de balanceamento — alpha-0.11.0

Baseline: **100000 partidas**, `heuristic` vs `heuristic`, seed base `1`, replay integral `true`.

## Saúde do gate

| concluídas | crashes | loops | estados mortos | estados inválidos | comandos rejeitados | divergências |
|---:|---:|---:|---:|---:|---:|---:|
| 100000 | 0 | 0 | 0 | 0 | 0 | 0 |

## Ritmo e iniciativa

- Rodadas: média **30.10**, p50 **28**, p95 **51**, máximo **61**.
- Comandos: média **70.98**, p50 **68**, p95 **111**, máximo **127**.
- Primeiro jogador: **51.31%** (51307/100000).

### Causa do término

| Causa | Partidas | Participação | Média de rodadas | p95 |
|---|---:|---:|---:|---:|
| assalto | 68284 | 68.28% | 25.96 | 37 |
| fadiga | 16371 | 16.37% | 49.10 | 57 |
| guarda | 1349 | 1.35% | 26.49 | 36 |
| maldicao | 9034 | 9.03% | 27.60 | 38 |
| pressao_de_nythara | 509 | 0.51% | 59.12 | 61 |
| rito | 2350 | 2.35% | 26.91 | 37 |
| sangramento | 2103 | 2.10% | 26.04 | 37 |

## Win rate por Campeão

| Campeão | Partidas | Vitórias | Win rate |
|---|---:|---:|---:|
| Voren Ashhand, Mestre da Retorta (`CH-CI-01`) | 20000 | 10056 | 50.28% |
| Edda, Escriba das Cinzas (`CH-CI-02`) | 20000 | 9949 | 49.74% |
| Nyra dos Sete Reflexos (`CH-MI-01`) | 20000 | 10127 | 50.63% |
| Oren, Leitor do Véu (`CH-MI-02`) | 20000 | 9905 | 49.53% |
| Mara Vale, Lâmina do Primeiro Sol (`CH-SO-01`) | 20000 | 9996 | 49.98% |
| Ilyan, Portador da Cicatriz Dourada (`CH-SO-02`) | 20000 | 9979 | 49.90% |
| Rauk Fenclaw, Alfa da Fenda (`CH-VA-01`) | 20000 | 10000 | 50.00% |
| Saela, Pele-de-Lua (`CH-VA-02`) | 20000 | 10009 | 50.04% |
| Seris Vhal, Herdeira Carmesim (`CH-VH-01`) | 20000 | 9941 | 49.70% |
| Kaedor, o Sem-Pulso (`CH-VH-02`) | 20000 | 10038 | 50.19% |

## Sinais de cartas dominantes

Candidatas com amostra mínima, ordenadas por played win rate; correlação não implica causalidade.

| Carta | Jogos em que foi usada | Played WR | Comprada WR | Jogadas | Rodada média | Dead in hand | Alerta |
|---|---:|---:|---:|---:|---:|---:|:---:|
| Pó de Vidro (`VR-067`) | 9981 | 62.26% | 47.76% | 10025 | 15.05 | 86.74% | — |
| Veia Aberta (`VR-009`) | 50404 | 59.47% | 57.07% | 53715 | 13.25 | 7.65% | — |
| Navalha de Vésper (`VR-081`) | 52516 | 59.18% | 58.04% | 56306 | 12.44 | 3.59% | — |
| Marca do Caçador (`VR-015`) | 67458 | 59.06% | 58.20% | 73385 | 11.24 | 3.34% | — |
| Refração Dolorosa (`VR-025`) | 52505 | 57.29% | 56.37% | 56305 | 11.49 | 2.97% | — |
| Corte Rubro (`VR-001`) | 53072 | 57.27% | 56.31% | 57008 | 11.47 | 2.94% | — |
| Cheiro de Medo (`VR-039`) | 67028 | 55.98% | 55.28% | 73015 | 11.15 | 3.33% | — |
| Manto Hemático (`VR-003`) | 52502 | 55.77% | 55.43% | 56476 | 11.28 | 16.47% | — |
| Lâmina Carbonizada (`VR-050`) | 52444 | 55.22% | 54.21% | 56144 | 12.40 | 3.54% | — |
| Mordida Dupla (`VR-105`) | 51783 | 55.15% | 54.15% | 55647 | 12.46 | 3.53% | — |
| Faca de Prata Fosca (`VR-061`) | 52347 | 55.02% | 54.08% | 56097 | 12.45 | 3.51% | — |
| Lâmina Calcinada (`VR-115`) | 50303 | 54.96% | 53.09% | 53753 | 13.28 | 7.71% | — |
| Golpe da Alvorada (`VR-013`) | 52157 | 54.96% | 54.08% | 55941 | 12.45 | 3.52% | — |
| Rasgo de Presa (`VR-037`) | 52617 | 54.87% | 53.91% | 56341 | 12.45 | 3.66% | — |
| Vidro de Retorno (`VR-027`) | 53773 | 54.85% | 54.82% | 58044 | 11.20 | 14.25% | — |

## Gate competitivo — APROVADO

- ✓ `first_player_47_5_52_5`: 51.31%
- ✓ `p95_rounds_le_60`: 51
- ✓ `champions_40_60`: 49.53%–50.63%
- ✓ `matchups_in_range`: 46.20%–54.20% (limite 25%–75%)
- ✓ `sampled_cards_drawn_35_65`: 0 fora da faixa (amostra mínima 500)
- ✓ `assault_finishes_ge_35`: 68.28%

## Interpretação

Este baseline usa decks variados legais sobre todo o pool do Modo Confronto, com composição mínima 8/8/4. Use a matriz de matchups e os sinais por carta para formular hipóteses; mudanças de regra exigem ADR, bump de ruleset e nova simulação.
