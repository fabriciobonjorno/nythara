# Relatório de balanceamento — alpha-0.6.0

Baseline: **100000 partidas**, `heuristic` vs `heuristic`, seed base `1`, replay integral `true`.

## Saúde do gate

| concluídas | crashes | loops | estados mortos | estados inválidos | comandos rejeitados | divergências |
|---:|---:|---:|---:|---:|---:|---:|
| 100000 | 0 | 0 | 0 | 0 | 0 | 0 |

## Ritmo e iniciativa

- Rodadas: média **35.93**, p50 **30**, p95 **70**, máximo **102**.
- Comandos: média **353.98**, p50 **313**, p95 **661**, máximo **979**.
- Primeiro jogador: **49.81%** (49810/100000).

## Win rate por Campeão

| Campeão | Partidas | Vitórias | Win rate |
|---|---:|---:|---:|
| Voren Ashhand, Mestre da Retorta (`CH-CI-01`) | 20000 | 8449 | 42.24% |
| Edda, Escriba das Cinzas (`CH-CI-02`) | 20000 | 8848 | 44.24% |
| Nyra dos Sete Reflexos (`CH-MI-01`) | 20000 | 7751 | 38.76% |
| Oren, Leitor do Véu (`CH-MI-02`) | 20000 | 9773 | 48.86% |
| Mara Vale, Lâmina do Primeiro Sol (`CH-SO-01`) | 20000 | 13217 | 66.09% |
| Ilyan, Portador da Cicatriz Dourada (`CH-SO-02`) | 20000 | 10909 | 54.55% |
| Rauk Fenclaw, Alfa da Fenda (`CH-VA-01`) | 20000 | 11411 | 57.05% |
| Saela, Pele-de-Lua (`CH-VA-02`) | 20000 | 11274 | 56.37% |
| Seris Vhal, Herdeira Carmesim (`CH-VH-01`) | 20000 | 8436 | 42.18% |
| Kaedor, o Sem-Pulso (`CH-VH-02`) | 20000 | 9932 | 49.66% |

## Sinais de cartas dominantes

Candidatas com amostra mínima, ordenadas por played win rate; correlação não implica causalidade.

| Carta | Jogos em que foi usada | Played WR | Comprada WR | Jogadas | Rodada média | Dead in hand | Alerta |
|---|---:|---:|---:|---:|---:|---:|:---:|
| Coração da Lua Feral (`VR-048`) | 7860 | 68.77% | 56.36% | 7860 | 6.43 | 60.46% | ⚠ |
| Sentença do Meio-Dia (`VR-024`) | 38074 | 61.21% | 60.51% | 70566 | 29.13 | 0.75% | — |
| Flecha de Vidro Solar (`VR-020`) | 39878 | 60.38% | 60.36% | 142556 | 28.01 | 0.75% | — |
| Golpe da Alvorada (`VR-013`) | 39912 | 60.37% | 60.34% | 146427 | 27.89 | 0.81% | — |
| Sentinela de Solenne (`VR-019`) | 39787 | 60.36% | 60.31% | 76076 | 14.05 | 0.06% | — |
| Marca do Caçador (`VR-015`) | 39964 | 60.34% | 60.33% | 154832 | 28.10 | 0.39% | — |
| Segundo Horizonte (`VR-022`) | 39207 | 60.34% | 60.42% | 119121 | 29.36 | 6.99% | — |
| Círculo de Sal Dourado (`VR-021`) | 39898 | 60.32% | 60.33% | 149344 | 27.90 | 0.43% | — |
| Mandato da Aurora (`VR-023`) | 39825 | 60.31% | 60.33% | 143218 | 28.55 | 0.41% | — |
| Interrogatório ao Amanhecer (`VR-018`) | 39919 | 60.29% | 60.31% | 149284 | 27.91 | 0.43% | — |
| Égide de Lumen (`VR-014`) | 39186 | 60.11% | 60.42% | 130260 | 29.58 | 5.53% | — |
| Rosário de Ferro Branco (`VR-016`) | 25555 | 60.09% | 60.30% | 32098 | 7.97 | 52.02% | — |
| Passo entre Sombras (`VR-017`) | 37149 | 58.50% | 60.38% | 93486 | 31.75 | 14.66% | — |
| Frenesi de Três Batidas (`VR-044`) | 38971 | 57.86% | 57.20% | 93602 | 17.96 | 0.90% | — |
| Caçada em Espiral (`VR-047`) | 38771 | 57.41% | 57.01% | 92454 | 17.84 | 0.94% | — |

## Alertas

- VR-048 excedeu 65% de played win rate

## Interpretação

Este baseline usa decks preconstruídos determinísticos sem facção aliada. Use a matriz de matchups e os sinais por carta para formular hipóteses; mudanças de regra exigem ADR, bump de ruleset e nova simulação.
