# Plano de Balanceamento e Simulação

## Métricas por carta
- inclusion rate
- drawn win rate
- played win rate
- mulligan keep rate
- average damage prevented/caused
- average Eclipse displacement
- combo trigger rate
- average card advantage
- matchup delta por Campeão
- turn played
- dead-in-hand rate

## Metas Alpha
- nenhuma carta, com amostra mínima, fora de 35–65% de drawn-win-rate;
- first-player entre 47,5–52,5%;
- p95 até 30 turnos no Modo Confronto;
- Campeões entre 40–60% no campo variado e matchups entre 25–75%;
- precons entre 40–60% por Campeão e matchups entre 20–80%;
- nenhuma combinação determinística mata a partir de 30 sem janela real de resposta antes da rodada tardia;
- decks devem ter múltiplas linhas de decisão;
- evitar locks de fase permanentes;
- evitar recursão infinita;
- controlar geração de recurso;
- observar first-player advantage.

## Bots
Criar 4 níveis:
1. random legal;
2. heuristic;
3. search curto;
4. self-play/avaliação offline.

Implementados na Fase 6: níveis 1 (`random`) e 2 (`heuristic`). O baseline usa
o baralho inicial determinístico e cobre a matriz ordenada 10×10 de Avatares
cosméticos. O modo `varied` sorteia toda a pool legal do Confronto sob a
composição mínima 8/8/4; fuzz de comandos e escolhas roda na suíte. Níveis
3–4 permanecem como expansão.

Execução local:

```bash
make sim-smoke # 1.000 partidas
make sim-100k  # gate completo, JSON + Markdown em backend/artifacts/
```

O relatório falha diante de crash, loop, estado morto/inválido, comando
rejeitado ou divergência de replay. O mesmo seed/configuração produz as mesmas
métricas independentemente da quantidade de workers.

Rodar pelo menos:
- 100k partidas por ruleset e por modo (`precon` e `varied`);
- matriz Champion x Champion;
- decks preconstruídos;
- decks gerados por restrições;
- fuzz decks legais.

O relatório `balance-report.v2` também separa Assalto, Guarda, Rito,
Sangramento, Maldição e Pressão de Nythara como causas de término. O gate
exige ≥35% de finais por Assalto.

Baseline histórico alpha-0.9.1 (100 mil + replay integral em cada modo):

- precon: iniciativa 47,98%, média 12,40, p95 22, Assalto 82,86%;
- varied: iniciativa 49,86%, média 8,63, p95 25, Assalto 83,13%.

Baseline aprovado alpha-0.10.2 (100 mil + replay integral em cada modo):

- precon: iniciativa 47,87%, média 12,09, p95 23, Assalto 82,24%;
- varied: iniciativa 49,54%, média 8,84, p95 25, Assalto 83,35%;
- saúde agregada: 200.000/200.000 concluídas, zero crash, loop, estado morto,
  estado inválido, comando rejeitado ou divergência determinística;
- nenhuma carta com amostra mínima ficou fora de 35–65% de drawn win rate;
- relatórios: `backend/artifacts/confront-tactical-approved-precon-100k.*` e
  `backend/artifacts/confront-tactical-approved-varied-100k.*`.

## Gates
Uma carta nova só entra em produção se:
- schema válido;
- teste unitário;
- teste de interação;
- simulação;
- sem loop;
- sem estado impossível;
- texto UI compatível;
- tradução pronta;
- telemetria registrada.
