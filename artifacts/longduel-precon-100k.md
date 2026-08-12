# Relatório de balanceamento — alpha-0.11.0

Baseline: **100000 partidas**, `heuristic` vs `heuristic`, seed base `1`, replay integral `true`.

## Saúde do gate

| concluídas | crashes | loops | estados mortos | estados inválidos | comandos rejeitados | divergências |
|---:|---:|---:|---:|---:|---:|---:|
| 100000 | 0 | 0 | 0 | 0 | 0 | 0 |

## Ritmo e iniciativa

- Rodadas: média **41.94**, p50 **42**, p95 **55**, máximo **61**.
- Comandos: média **94.78**, p50 **97**, p95 **116**, máximo **128**.
- Primeiro jogador: **48.27%** (48275/100000).

### Causa do término

| Causa | Partidas | Participação | Média de rodadas | p95 |
|---|---:|---:|---:|---:|
| assalto | 55199 | 55.20% | 38.52 | 47 |
| fadiga | 25481 | 25.48% | 50.93 | 57 |
| maldicao | 9377 | 9.38% | 39.08 | 47 |
| pressao_de_nythara | 1092 | 1.09% | 58.84 | 60 |
| rito | 8851 | 8.85% | 38.33 | 47 |

## Win rate por Campeão

| Campeão | Partidas | Vitórias | Win rate |
|---|---:|---:|---:|
| Voren Ashhand, Mestre da Retorta (`CH-CI-01`) | 20000 | 10020 | 50.10% |
| Edda, Escriba das Cinzas (`CH-CI-02`) | 20000 | 9892 | 49.46% |
| Nyra dos Sete Reflexos (`CH-MI-01`) | 20000 | 9965 | 49.83% |
| Oren, Leitor do Véu (`CH-MI-02`) | 20000 | 9961 | 49.80% |
| Mara Vale, Lâmina do Primeiro Sol (`CH-SO-01`) | 20000 | 9988 | 49.94% |
| Ilyan, Portador da Cicatriz Dourada (`CH-SO-02`) | 20000 | 9958 | 49.79% |
| Rauk Fenclaw, Alfa da Fenda (`CH-VA-01`) | 20000 | 10056 | 50.28% |
| Saela, Pele-de-Lua (`CH-VA-02`) | 20000 | 10012 | 50.06% |
| Seris Vhal, Herdeira Carmesim (`CH-VH-01`) | 20000 | 10126 | 50.63% |
| Kaedor, o Sem-Pulso (`CH-VH-02`) | 20000 | 10022 | 50.11% |

## Sinais de cartas dominantes

Candidatas com amostra mínima, ordenadas por played win rate; correlação não implica causalidade.

| Carta | Jogos em que foi usada | Played WR | Comprada WR | Jogadas | Rodada média | Dead in hand | Alerta |
|---|---:|---:|---:|---:|---:|---:|:---:|
| Passo Calculado (`VR-062`) | 119787 | 52.77% | 50.42% | 147588 | 16.76 | 55.47% | — |
| Instinto de Recuo (`VR-038`) | 118159 | 52.45% | 50.15% | 146225 | 16.54 | 56.18% | — |
| Manto Hemático (`VR-003`) | 164324 | 52.18% | 50.45% | 243521 | 18.44 | 26.60% | — |
| Vidro de Retorno (`VR-027`) | 170423 | 51.53% | 50.37% | 259320 | 18.16 | 21.92% | — |
| Costura de Cinzas (`VR-051`) | 165506 | 51.41% | 50.21% | 245311 | 18.54 | 26.33% | — |
| Refração Dolorosa (`VR-025`) | 192442 | 50.73% | 50.58% | 334823 | 16.94 | 1.61% | — |
| Corte Rubro (`VR-001`) | 192425 | 50.71% | 50.57% | 334939 | 16.90 | 1.62% | — |
| Marca do Caçador (`VR-015`) | 193147 | 50.59% | 50.43% | 334237 | 17.09 | 1.93% | — |
| Golpe da Alvorada (`VR-013`) | 192172 | 50.55% | 50.41% | 332964 | 17.54 | 1.71% | — |
| Rasgo de Presa (`VR-037`) | 192153 | 50.54% | 50.41% | 333429 | 17.54 | 1.68% | — |
| Lâmina Carbonizada (`VR-050`) | 192154 | 50.54% | 50.40% | 333010 | 17.51 | 1.68% | — |
| Cheiro de Medo (`VR-039`) | 192364 | 50.28% | 50.15% | 329894 | 16.75 | 2.25% | — |
| Gole da Víspera (`VR-082`) | 191658 | 50.26% | 50.06% | 327869 | 16.37 | 2.84% | — |
| Sopro de Cinza (`VR-113`) | 192166 | 50.20% | 50.10% | 331308 | 17.36 | 1.83% | — |

## Gate competitivo — APROVADO

- ✓ `first_player_47_5_52_5`: 48.27%
- ✓ `p95_rounds_le_60`: 55
- ✓ `p50_rounds_ge_30`: 42
- ✓ `champions_40_60`: 49.46%–50.63%
- ✓ `matchups_in_range`: 45.30%–54.00% (limite 20%–80%)
- ✓ `sampled_cards_drawn_35_65`: 0 fora da faixa (amostra mínima 500)
- ✓ `assault_finishes_ge_35`: 55.20%

## Interpretação

Este baseline usa os precons oficiais determinísticos. Use a matriz de matchups e os sinais por carta para formular hipóteses; mudanças de regra exigem ADR, bump de ruleset e nova simulação.
