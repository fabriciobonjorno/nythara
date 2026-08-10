# Projeto VÉU RUBRO
## Game Design Document — Alpha 0.1

**Status:** conceito original / pré-produção  
**Plataforma inicial:** Web/PWA, desktop e mobile browser  
**Objetivo:** criar um card game competitivo de fantasia sombria que recupere a sensação de partidas rápidas, personagens assimétricos, defesa reativa e combos dos card games brasileiros de navegador dos anos 2000, mas com universo, cartas, nomes, arte, regras de combinação e implementação totalmente novos.

---

## 1. Princípios de design

1. **Fácil de começar, difícil de dominar.** O jogador entende Assalto, Guarda e Rito em poucos minutos; domínio vem de Ressonância, leitura do adversário e controle do Eclipse.
2. **Sem tabuleiro lotado.** A partida é focada na mão e nas decisões, com no máximo 2 Manifestações e 2 Relíquias ativas por jogador.
3. **Defesa é uma decisão real.** Certos Assaltos abrem uma janela de Guarda; o defensor escolhe gastar recurso ou aceitar dano.
4. **Combos são sequências, não “solitários”.** O sistema de Ressonância recompensa ordenar cartas, mas limita loops infinitos.
5. **O universo deve ser próprio.** Nenhum nome, personagem, ilustração, texto ou arquivo do jogo histórico é reutilizado.
6. **PvP justo.** Cartas cosméticas/variantes podem monetizar; poder competitivo não deve depender de gasto.

---

## 2. Fantasia e mundo

O mundo de **Véu Rubro** vive preso entre duas forças físicas reais dentro da ficção: **Aurora** e **Noite**. Séculos atrás, um eclipse permanente abriu o Véu e transformou pactos, memórias e sangue em fontes de poder. As grandes facções não lutam simplesmente entre “bem” e “mal”; todas têm razões para empurrar o mundo para a luz, para a noite ou mantê-lo no crepúsculo.

### Facções

- **Casa Vhal** — aristocracia predatória. Sacrifício de Vitalidade, Dreno, Sangramento e pressão de Noite.
- **Ordem Solara** — caçadores e sentinelas. Ward, prevenção, contra-ataque, purificação e Aurora.
- **Conclave Mirr** — ocultistas do espelho. Informação, manipulação, cópias temporárias e Ressonância.
- **Matilha Varka** — metamorfos. Ataques em sequência, múltiplas janelas de Assalto e explosões de dano.
- **Sínodo Cinéreo** — alquimistas da morte. Descarte, exílio, Maldições e recursão.
- **Errantes** — cartas neutras.

---

## 3. Condição de vitória

Cada Campeão começa normalmente com **30 de Vitalidade** (alguns têm 29–32 como parte do balanceamento).

Você vence quando a Vitalidade do Campeão inimigo chega a **0 ou menos**.

Concessão, timeout competitivo e desconexão prolongada também podem encerrar a partida.

---

## 4. Deck

- 1 Campeão.
- **36 cartas**.
- Máximo de **2 cópias** de uma carta comum/incomum/rara/épica.
- Máximo de **1 cópia** de Lendárias.
- Pelo menos 24 cartas precisam ser da facção do Campeão ou neutras.
- Até 12 cartas podem pertencer a uma **facção aliada** definida pela temporada.
- Mão inicial: 5 cartas.
- Mulligan: qualquer quantidade, uma vez.
- Limite normal da mão: 7.
- Quando o deck acaba, o descarte não volta gratuitamente: o jogador sofre **Fadiga crescente** (1, depois 2, 3, 4...) e embaralha o descarte para formar novo deck.

---

## 5. Recurso: Essência

- Rodada 1: 3 de Essência máxima.
- Aumenta +1 por rodada até máximo 8.
- Recarrega no início da rodada.
- “Essência temporária” expira no fim da rodada.
- Algumas cartas usam Vitalidade, descarte ou exílio como custo adicional.

Isto cria um jogo mais legível que simplesmente permitir qualquer carta a qualquer momento e dá espaço para custos/balanceamento.

---

## 6. Estrutura da rodada

1. **Preparação** — efeitos de início, recarga de Essência.
2. **Compra** — cada jogador compra 1.
3. **Postura** — cada jogador escolhe secretamente uma postura; revelação simultânea:
   - **Predação:** primeiro Assalto da rodada causa +1 se acertar.
   - **Vigília:** primeira Guarda custa 1 a menos.
   - **Arcano:** primeiro Rito custa 1 a menos, mas não pode causar dano direto.
4. **Rito** — jogador com iniciativa pode usar Ritos/Relíquias/Manifestações; oponente recebe sua janela.
5. **Confronto** — janela principal de Assalto. Assaltos elegíveis podem gerar janela de Guarda.
6. **Crepúsculo** — resolve Sangramento, Maldições, expirações e efeitos de fim.
7. **Iniciativa alterna**.

O servidor é o único árbitro da ordem dos eventos.

---

## 7. Medidor de Eclipse — mecânica central original

O Medidor vai de **-3 a +3**:

`AURORA TOTAL -3 ← -2 ← -1 ← 0 → +1 → +2 → +3 ECLIPSE NOTURNO`

Cartas podem deslocá-lo. Ao chegar a -3 ou +3:

1. dispara um **Evento de Eclipse**;
2. Campeões e cartas com texto de Eclipse podem despertar;
3. o estado dura até o fim da rodada;
4. depois retorna a 0.

Isso cria uma “segunda partida” sobre a mesa: causar dano não é a única disputa; o jogador também decide quando permitir ou negar o pico de poder global.

---

## 8. Ressonância — sistema de combos

Toda carta emite um **Sigilo** ao resolver:

- Presa
- Sol
- Espelho
- Garra
- Cinza
- Coroa

Os Sigilos resolvidos na rodada formam a **Trilha de Ressonância**.

Exemplo:

`Presa → Coroa → Espelho`

Uma carta pode dizer:

> Ressonância Presa→Coroa: cure 2.

A condição verifica a sequência recente. O limite padrão é **5 Sigilos por jogador por rodada**; efeitos excepcionais podem aumentar temporariamente esse limite.

### Regras anti-loop

- Um mesmo gatilho de Ressonância de uma mesma instância de carta só dispara uma vez por rodada.
- Cópias não podem copiar “limite 1 por deck”.
- Cartas geradas não entram no deck/coleção.
- Cadeias forçadas têm limite de profundidade no motor.
- Se múltiplos gatilhos surgirem juntos, usa-se uma fila determinística.

---

## 9. Tipos de carta

### Assalto
Dano direto e pressão. Alguns ataques são “defensáveis”; outros possuem condições especiais.

### Guarda
Reações usadas quando uma janela de defesa é aberta. Prevêm dano, geram Ward ou contra-efeitos.

### Rito
Manipulação, cura, compra, descarte, Eclipse, Maldições e combos.

### Relíquia
Permanente; máximo de 2 ativas. Uma terceira exige substituir/destruir uma existente.

### Manifestação
Personagem/entidade de apoio; máximo de 2 ativas. Não possui combate próprio no Alpha: fornece efeitos persistentes e gatilhos.

---

## 10. Glossário inicial

- **Ward X:** escudo que absorve X de dano e persiste até consumido ou até indicação em contrário.
- **Sangramento X:** no Crepúsculo da próxima rodada do alvo, ele sofre X; depois remove.
- **Exposto:** próximo dano recebido ganha +2; depois remove.
- **Véu:** não pode ser alvo de Ritos adversários até o fim da duração.
- **Recuperar:** mover do descarte para a mão.
- **Exilar:** remover da partida atual, salvo efeito explícito.
- **Maldição:** efeito negativo com duração.
- **Eclipse Noturno / Aurora Total:** estados disparados em +3/-3.
- **Sigilo:** símbolo usado pela Ressonância.
- **Fadiga:** dano crescente ao tentar reciclar deck vazio.

---

## 11. Campeões

O Alpha possui **10 Campeões originais**, dois por facção. Cada um tem:
- Vitalidade própria;
- passiva;
- habilidade única de uso uma vez por partida;
- Forma de Eclipse.

Arquivo completo: `champions_alpha.json`.

---

## 12. Set Alpha

O primeiro conjunto possui:
- **80 cartas jogáveis originais**;
- 10 Campeões;
- 5 facções + neutras;
- 5 tipos de carta;
- 6 Sigilos;
- raridades Comum, Incomum, Rara, Épica e Lendária.

Arquivos:
- `cards_alpha.json`
- `cards_alpha.csv`

As cartas foram desenhadas para validar o motor. Antes de qualquer monetização competitiva, precisam passar por simulação, telemetria e testes humanos.

---

## 13. Modos

### Alpha
- Tutorial PvE
- Bot treino
- Casual 1v1
- Ranked 1v1
- Duelo por código/convite

### Pós-MVP
- História episódica
- Draft
- Arena selada
- Torneios
- Espectador
- Replays
- Guildas
- eventos cooperativos
- PvP 2v2 experimental

---

## 14. Progressão e monetização

### Recomendação
**Não vender vantagem competitiva diretamente.**

Monetizar:
- passes cosméticos;
- skins de Campeão;
- molduras;
- backs de carta;
- animações;
- tabuleiros/arenas;
- emotes;
- versões foil/animadas;
- campanhas PvE;
- bundles com proteção contra duplicata.

Para ranked, manter caminho realista de aquisição de cartas por jogo e oferecer um modo competitivo de “coleção completa” em torneios oficiais internos quando apropriado.

---

## 15. Interface

### Tela de batalha
- oponente no topo;
- Vitalidade + Campeão;
- Medidor de Eclipse central, sempre visível;
- área de Relíquias/Manifestações;
- cartas resolvidas no centro;
- mão do jogador em leque;
- Trilha de Ressonância logo acima da mão;
- botões de janela/tempo claros;
- histórico de eventos acessível.

### Direção artística
Dark fantasy barroco + ocultismo astronômico.
Evitar reproduzir molduras, fontes, personagens ou composição visual do título histórico.
Paleta pode explorar carvão, vinho escuro, marfim, ouro envelhecido e azul lunar, mas o layout precisa ser novo.

---

## 16. Metas de experiência

- tutorial até 8 minutos;
- primeira partida completa até 12 minutos;
- média ranked desejada: 8–15 minutos;
- decisões significativas desde a rodada 1;
- nenhuma partida deve depender de “quem gastou mais dinheiro”;
- taxa de vitória ideal dos melhores decks após estabilização: evitar extremos; balancear com dados, não sensação.

---

## 17. Diferenciação clara

O produto não é uma reconstrução 1:1.

**Inspirado na sensação:**
- personagem + deck;
- vida;
- ataque/defesa/magia como pilares;
- defesa reativa;
- combos;
- partidas online.

**Original:**
- universo e lore;
- Campeões;
- todas as cartas;
- Eclipse global bidirecional;
- Ressonância por sequência de Sigilos;
- Posturas simultâneas;
- Essência;
- Relíquias/Manifestações limitadas;
- regras de fadiga;
- progressão, interface e implementação.

---

## 18. Critérios para sair do Alpha

- engine determinística com 100% de replay;
- nenhuma divergência entre servidor e cliente em 100k partidas simuladas;
- nenhuma carta com loop infinito;
- bots capazes de executar todas as regras;
- matchmaking e reconexão;
- testes de carga WebSocket;
- coleção/decks persistentes;
- telemetria por carta;
- painel de balanceamento;
- segurança contra cliente adulterado;
- regras versionadas por temporada.
