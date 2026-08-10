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
- nenhuma carta >65% played-win-rate de forma persistente sem contexto;
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
decks preconstruídos determinísticos da facção do Campeão + Errantes e cobre a
matriz ordenada 10×10. Níveis 3–4 e decks gerados/fuzz permanecem como expansão.

Execução local:

```bash
make sim-smoke # 1.000 partidas
make sim-100k  # gate completo, JSON + Markdown em backend/artifacts/
```

O relatório falha diante de crash, loop, estado morto/inválido, comando
rejeitado ou divergência de replay. O mesmo seed/configuração produz as mesmas
métricas independentemente da quantidade de workers.

Rodar pelo menos:
- 100k partidas por ruleset em CI noturna/benchmark;
- matriz Champion x Champion;
- decks preconstruídos;
- decks gerados por restrições;
- fuzz decks legais.

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
