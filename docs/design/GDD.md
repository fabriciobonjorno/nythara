# Projeto NYTHARA
## Game Design Document — Alpha 0.13 ("Decisões Servidas")

**Status:** conceito original / pré-produção  
**Plataforma inicial:** Web/PWA, desktop e mobile browser  
**Objetivo:** criar um card game competitivo de fantasia sombria que recupere a sensação de partidas rápidas, defesa reativa e duelos diretos dos card games brasileiros de navegador dos anos 2000, mas com universo, cartas, nomes, arte, regras e implementação totalmente novos.

> **Nota de versão.** O Alpha ≤ 0.8.3 usava um ruleset com Campeões, Essência,
> Posturas, Eclipse e Ressonância. A validação humana mostrou que a partida
> ficou ilegível (ADR-044). O Alpha 0.9 pivotou para o **Modo Confronto** e o
> Alpha 0.10 acrescentou os **Selos de Fase** (ADR-051), o 0.11 estabeleceu o
> duelo longo e a composição 8/10/4 (ADRs 054–055), o 0.12 devolveu poderes
> versionados aos Avatares (ADR-057) e o 0.13 publicou decisões de cartas
> (ADRs 058–059). O ruleset legado permanece no motor apenas para
> replay de partidas históricas; seu design está no histórico do git e nos
> ADRs 001–043.

---

## 1. Princípios de design

1. **Legível em uma partida.** O jogador entende Assalto, Guarda e Rito no primeiro duelo, sem ler manual. Uma decisão por vez, sempre óbvio de quem é a vez.
2. **O confronto é o palco.** Toda troca acontece no centro da mesa: a carta de ataque voa para lá, a defesa responde, a perdedora se estilhaça. O jogador *vê* o jogo acontecer.
3. **Defesa é uma decisão real.** Assaltos defensáveis abrem a janela de Guarda; o defensor escolhe sacrificar Vitalidade para bloquear ou aceitar o dano.
4. **Um recurso só.** Vitalidade é vida **e** custo. Cada carta jogada é sangue vertido — agressividade tem preço, e o preço é legível.
5. **O universo deve ser próprio.** Nenhum nome, personagem, ilustração, texto ou arquivo de jogos históricos é reutilizado.
6. **PvP justo.** Cosméticos podem monetizar; poder competitivo não depende de gasto.

---

## 2. Fantasia e mundo

O mundo de **Nythara** vive preso entre duas forças físicas reais dentro da ficção: **Aurora** e **Noite**. Séculos atrás, um eclipse permanente abriu o Véu e transformou pactos, memórias e sangue em fontes de poder. As grandes facções não lutam simplesmente entre “bem” e “mal”; todas têm razões para empurrar o mundo para a luz, para a noite ou mantê-lo no crepúsculo.

### Facções

- **Casa Vhal** — aristocracia predatória. Sacrifício de Vitalidade, Dreno, Sangramento.
- **Ordem Solara** — caçadores e sentinelas. Ward, prevenção, contra-ataque, purificação.
- **Conclave Mirr** — ocultistas do espelho. Informação, manipulação, cópias temporárias.
- **Matilha Varka** — metamorfos. Ataques em sequência e explosões de dano.
- **Sínodo Cinéreo** — alquimistas da morte. Descarte, exílio, Maldições e recursão.
- **Errantes** — cartas neutras.

No Modo Confronto as facções são **identidade visual e temática** das cartas (moldura, sigilo, vocabulário de efeitos) — não impõem restrição de deck.

---

## 3. Condição de vitória

Cada duelista começa com **52 de Vitalidade**.

Você vence quando a Vitalidade inimiga chega a **0 ou menos**. Duplo nocaute segue os critérios determinísticos do ADR-013.

Concessão, timeout competitivo e desconexão prolongada também encerram a
partida. No treino contra bot, expirar uma fase comum apenas passa; se uma
decisão de carta estiver aberta, o sistema confirma deterministicamente as
primeiras N opções para que a partida e o replay não travem.

---

## 4. Deck

- **30 cartas**, apenas dos tipos **Assalto, Guarda e Rito** (pool legal do modo).
- Composição mínima: **8 Assaltos, 10 Guardas e 4 Ritos**; os 8 slots
  restantes são livres dentro do pool. Isso impede baralhos sem condição real
  de confronto sem tirar a autoria do jogador.
- Sem Campeão e sem restrição de facção: qualquer combinação do pool.
- Máximo de **2 cópias** por carta; Lendárias, **1 cópia**.
- **Um deck ativo por conta.** Salvar o deck inicia uma **trava de edição de 24h** — o deck é um compromisso, não um ajuste entre partidas. O deck inicial gerado pelo sistema não trava.
- Mão inicial: 5 cartas. Sem mulligan. A Compra acontece inclusive no
  primeiro turno; quem tem a iniciativa toma sua primeira ação com 6 e o
  segundo duelista recebe uma carta adicional de compensação na abertura.
- Limite de mão: 7. Compra com a mão cheia **queima** a carta (vai ao descarte).
- Deck vazio: cada compra vira **Fadiga** crescente (2, depois 4, 6…). O descarte **não** é reembaralhado — o esgotamento pressiona o fim.

---

## 5. Recurso: Vitalidade (sacrifício)

Não há mana. **Jogar uma carta custa Vitalidade** — o custo impresso é subtraído da sua vida ao jogar.

- Um custo só é pagável se deixar você com **pelo menos 1** de Vitalidade.
- Cura devolve fôlego e, portanto, recurso; agressão pura encurta sua própria margem.
- Custos altos são apostas: a carta mais forte do jogo cobra um pedaço de quem a joga.

---

## 6. Estrutura do turno

Turnos **alternados** (iniciativa sorteada pelo RNG da partida):

1. **Alvorada** — Sangramentos sobre o jogador ativo resolvem; efeitos de início expiram/ativam.
2. **Compra** — compra 1 carta (Fadiga se o deck acabou; queima se a mão está cheia).
3. **Assalto** — pode jogar **até 1 carta de Assalto** (paga o custo em Vitalidade):
   - os dois jogadores compram antes de sua primeira ação e o segundo recebe
     uma carta adicional na abertura. No 0.13, o primeiro Assalto não recebe
     penalidade: a carta extra calibra a iniciativa nos dois perfis de baralho.
   - Assalto **defensável** abre a **janela de Guarda**: o defensor pode jogar **até 1 Guarda** (pagando o custo) ou passar.
   - **Resolução do confronto**: dano final = poder do Assalto − prevenção da
     Guarda (incluindo o bônus de 3 do formato, piso 0), passando por Ward.
     Quando uma Guarda foi comprometida, no máximo 1 de dano atravessa. A carta
     **perdedora se estilhaça**: dano 0 → o Assalto perde; dano > 0 → a Guarda
     perde; sem resposta → acerto direto. Efeitos adicionais da Guarda (Selo do
     Rito, Ward, cura) resolvem conforme o texto.
   - Assalto **indefensável** não abre janela.
4. **Rito** — pode jogar **até 1 carta de Rito** (cura, compra, Selos, Sangramento, remoção…).
5. **Crepúsculo** — durações do jogador ativo decrementam; excessos resolvem; passa a vez.

O servidor é o único árbitro da ordem dos eventos.

---

## 7. Pressão de Nythara — antitrava

No início do turno 58, a névoa fecha a arena e ambos perdem **2 de
Vitalidade**; a perda cresce para 4, 6, 8… nos turnos seguintes. A interface
avisa desde o turno 20. Se a pressão derrubar os dois com a mesma Vitalidade,
vence quem ganhou o confronto mais recente; sem confronto anterior, aplica-se
o desempate determinístico geral. A regra só alcança a cauda de baralhos
defensivos e impede que uma partida vire espera por Fadiga.

---

## 8. Glossário

- **Poder:** dano base de um Assalto. No Modo Confronto, a conversão do
  catálogo legado recebe +4 de escala; bônus condicionais aparecem no placar
  calculado do confronto.
- **Prevenção:** quanto uma Guarda subtrai do dano do confronto.
- **Revide:** dano que uma Guarda devolve ao atacante ao bloquear.
- **Ward X:** escudo que absorve X de dano e persiste até consumido.
- **Sangramento X:** na Alvorada do próximo turno do alvo, ele sofre X; depois remove.
- **Exposto / Selo da Guarda:** o próximo Assalto contra o alvo recebe **+2
  de Poder** e não abre janela de Guarda; Exposto é consumido ao declarar
  esse Assalto. É uma preparação forte, pública e respondível antes do turno
  seguinte.
- **Selo do Assalto:** a próxima fase de Assalto do alvo é pulada; ele ainda
  recebe sua Compra e pode usar um Rito.
- **Selo do Rito:** a fase de Rito do atacante após o confronto atual é
  pulada.
- **Véu:** não pode ser alvo de Ritos adversários até o fim da duração.
- **Recuperar:** mover do descarte para a mão.
- **Exilar:** remover da partida atual, salvo efeito explícito.
- **Maldição:** efeito negativo com duração.
- **Fadiga:** dano crescente ao comprar de deck vazio (2, 4, 6…).
- **Sigilo:** símbolo de afinidade impresso na carta (Presa, Sol, Espelho, Garra, Cinza, Coroa). No Modo Confronto é identidade visual — sem regra ativa.

---

## 9. Tipos de carta

### Assalto
Dano direto e pressão. Cada Assalto tem **Poder** e é **defensável** ou **indefensável**. Alguns têm efeitos anexos (Sangramento, compra, Selo…).

### Guarda
Reações jogadas **somente** na janela de Guarda. Têm **Prevenção** e podem ter efeitos anexos (Revide, Ward, cura).

### Rito
Manipulação jogada no próprio turno: cura, compra, descarte do oponente, Selos, Maldições, recursão.

> **Fora do modo:** Relíquias e Manifestações (permanentes) não participam do
> Modo Confronto. Cartas cujo efeito depende de sistemas do ruleset legado
> (Eclipse, Ressonância, Posturas, permanentes) ficam fora do pool legal —
> excluídas explicitamente via `ImplementationReport`, nunca ignoradas em
> silêncio. Assaltos e Guardas podem entrar como adaptações básicas quando
> Poder/Prevenção são independentes: o texto é substituído, a API marca
> `adapted=true` e o efeito secundário removido aparece no relatório. Ritos
> não usam fallback.

---

## 10. Avatares

Os 10 personagens do Alpha (Kaedor, Seris, Mara, Ilyan, Nyra, Oren, Rauk,
Saela, Voren, Edda) são **Avatares**: retrato, título, presença na arena e um
poder próprio versionado. Os poderes usam apenas sistemas do Confronto — cura,
Ward e desconto de custo ao atingir um limiar — e foram calibrados nos dois
perfis de baralho. A escolha muda a linha tática, mas não coleção, quantidade de
cartas nem acesso a poder comprado.

Arquivo: `champions_alpha.json` (reinterpretado como catálogo de avatares).

---

## 11. Set Alpha

- **130 cartas originais** no catálogo; o pool legal do Modo Confronto é o subconjunto de Assalto/Guarda/Rito cujos efeitos compilam no modo (relatório de implementação lista as exclusões).
- 5 facções + neutras como identidade visual.
- Raridades: Comum, Incomum, Rara, Épica, Lendária.

Arquivos: `cards_alpha.json`, `cards_alpha.csv`.

---

## 12. Modos

### Alpha
- Treino contra bot
- Casual 1v1
- Ranked 1v1 (patentes)
- Duelo por código/convite

### Pós-MVP
- História episódica, Draft, Arena selada, Torneios, Espectador, Guildas, 2v2 experimental.

---

## 13. Progressão e monetização

**Não vender vantagem competitiva.** Monetizar: cosméticos (avatares, molduras, versos de carta, animações, arenas, emotes, foils), campanhas PvE, bundles com proteção de duplicata. Caminho realista de aquisição de cartas jogando.

---

## 14. Interface

### Tela de batalha
- **Oponente no topo:** avatar, nome, Vitalidade, mão em versos, deck e estados visíveis.
- **Centro: a zona de confronto.** As cartas jogadas voam para o centro; o choque acontece ali; a perdedora se estilhaça; números de dano voam até a Vitalidade atingida.
- **Log de partida** acessível (“Fulano comprou uma carta.”) — a mesa conta a própria história.
- **Jogador na base:** avatar, Vitalidade, indicador de fase (Assalto → Guarda → Rito), mão em leque, passe secundário e timer visível.
- **Janela de Guarda:** a mesa escurece, as Guardas jogáveis acendem, o defensor decide.
- Zoom/preview grande de carta em hover ou toque longo.
- Texto tático derivado da DSL, com papel, palavras-chave e resolução completa
  da carta; o resultado final continua exclusivo da engine.
- Controles equivalentes: um clique/toque direto, arraste e teclado (1–7 joga
  a carta, Espaço passa a fase, H abre ajuda, Esc fecha painéis). Sem carta
  legal, a janela avança automaticamente depois de uma indicação breve.
- Barra proporcional de Vitalidade, emblema original do Avatar e anúncio
  inequívoco de turno; nenhuma informação depende somente de cor.
- Acessibilidade: movimento reduzido, alto contraste, texto grande — todos os efeitos respeitam as preferências.

### Direção artística
Dark fantasy barroco + ocultismo astronômico. Paleta de carvão, vinho escuro, marfim, ouro envelhecido e azul lunar. Nenhuma moldura, fonte, personagem ou composição do título histórico é reproduzida.

---

## 15. Metas de experiência

- primeira partida completa em até 12 minutos;
- treino alcançável em até três decisões após a criação da conta;
- mesa inteira, inclusive a mão, visível sem rolagem durante o duelo;
- média ranked desejada: 8–15 minutos;
- decisões significativas desde o turno 1;
- término dominante por Assalto (meta ADR-041: ≥ 35% — no Confronto, por construção);
- nenhuma partida decidida por “quem gastou mais dinheiro”.

---

## 16. Diferenciação clara

O produto não é uma reconstrução 1:1 de nenhum jogo histórico.

**Inspirado na sensação:** vida única como recurso, ataque/defesa/magia, defesa reativa, confronto central visível, partidas online rápidas.

**Original:** universo e lore de Nythara, todas as 130 cartas, Pressão de
Nythara, Selos de Fase, Ward, Fadiga sem reembaralhar, trava de
deck de 24h, patentes/temporadas, interface e implementação.

---

## 17. Critérios para sair do Alpha

- engine determinística com 100% de replay;
- nenhuma divergência servidor/cliente em partidas simuladas em massa;
- nenhuma carta com loop infinito;
- bots capazes de executar todas as regras do modo;
- matchmaking e reconexão;
- coleção/deck persistente com trava;
- telemetria por carta;
- segurança contra cliente adulterado;
- regras versionadas por temporada.
