# Validação de experiência — onda 2 do Alpha 0.9.1

Data: 2026-08-11  
Escopo: interação tátil, paisagem sonora, retorno pós-partida e leitura do
baralho. Nenhuma mudança de regra, catálogo, comando ou fórmula da engine.

## Resultado

**APROVADO.** A segunda onda preserva a autoridade do servidor e os gates de
balanceamento do Modo Confronto.

## Navegador real

Jornada executada no cliente local com uma conta nova criada pela própria UI:

1. Home carregou um baralho inicial legal de 30 cartas.
2. Construtor exibiu 62 cartas jogáveis e 68 arquivadas.
3. “Leitura do baralho” calculou no deck real custo médio 1,0, Poder médio
   6,2, Prevenção média 2,8 e três papéis de Rito.
4. O botão superior “Salvar baralho” permaneceu visível e habilitado no
   viewport desktop enquanto a lista lateral continuou rolável.
5. O treino abriu em uma ação. Na primeira janela havia duas cartas
   autorizadas: somente elas receberam brilho, comando e instrução de arraste;
   Guardas e Ritos fora da fase permaneceram desabilitados.
6. A ajuda da Arena mostrou os quatro caminhos equivalentes: arraste, 1–7,
   Espaço e H/Esc.
7. O resultado mostrou análise do duelo, “Jogar agora contra o bot” e “Buscar
   rival” como escolhas distintas.
8. “Jogar agora contra o bot” abriu outra sala no turno 1 sem passar por Home
   ou fila.
9. Ajustes exibiu “Trilha ambiente” separada de “Som”, ativada por padrão e
   com início declarado somente após a primeira ação.

O gesto de ponteiro é cancelável abaixo do limiar de 96 px e só é instalado em
cartas presentes em `state.playable`. Ao completar o gesto, chama o mesmo
`play` usado pelo botão e pelo teclado; `LegalPlayIDs` e `Apply` continuam sendo
as duas barreiras authoritative.

## Gates automatizados

- `make test-race`: aprovado em todos os pacotes, incluindo engine, battle,
  HTTP, storage, simulação e segurança.
- `make lint`: `go vet ./...` aprovado.
- `npm run build --prefix web`: TypeScript e bundle Vite aprovados.
- `git diff --check`: aprovado.
- `make e2e-real`: aprovado; PvP natural em 25 rodadas/22 cartas e treino
  natural em 10 rodadas/8 cartas humanas. Ocultação da mão rival, confronto
  central, estilhaço, rating, unicidade de e-mail/username e formato de
  username permaneceram verdadeiros.
- `make sim-smoke`: 1.000/1.000 partidas, zero crash/loop/estado morto/estado
  inválido/comando rejeitado/divergência; iniciativa 49,60%, média 12,42,
  p95 22 e 79,80% dos finais por Assalto.

Os dois relatórios oficiais anteriores de 100 mil partidas continuam válidos,
pois este pacote não altera o snapshot `alpha-0.9.1`, o pool ou a engine.

## Limites honestos

- Áudio adaptativo é sintetizado no dispositivo e depende de WebAudio; falha
  ou bloqueio do navegador deixa a partida silenciosa, nunca bloqueada.
- A leitura do baralho é estrutural e consultiva. Não estima matchmaking, mão
  inicial ou taxa de vitória.
- Revanche contra o mesmo jogador exige protocolo de aceite bilateral e fica
  fora desta onda; o retorno instantâneo atual é treino, explicitamente
  rotulado, enquanto PvP retorna à busca normal.
