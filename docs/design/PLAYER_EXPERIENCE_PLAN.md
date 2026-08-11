# Nythara — plano de experiência do jogador

**Status:** primeira onda validada; segunda onda em implementação  
**Escopo:** clareza, fluidez, sensação física e retorno; nenhuma mudança na
regra já balanceada do Modo Confronto.

## 1. O que a pesquisa mostrou

O valor duradouro dos card games brasileiros de navegador do início dos anos
2000 não estava em uma interface complexa. A estrutura era fácil de narrar:
um deck de 30 cartas, compra por rodada e três famílias funcionais — ataque,
defesa reativa e magia. A mesa deixava claro quem atacava, quem podia defender
e qual recurso chegaria a zero. O deck ganhava personalidade pelas escolhas e
pelas combinações, não por uma camada extensa de regras antes do primeiro
duelo.

Referências atuais reforçam princípios compatíveis:

- onboarding curto e contextual é mais eficiente que um manual antes de
  jogar;
- toda ação precisa de estado visual distinto e resposta imediata;
- partidas curtas funcionam melhor quando o jogador volta ao próximo duelo
  sem atravessar menus desnecessários;
- experimentação de decks precisa ser acompanhada por feedback e progressão;
- tipografia, contraste, alvos de toque, teclado e movimento reduzido são
  parte da jogabilidade, não acabamento.

Fontes consultadas:

- Masters of Cards, “Deck de cartas” e “Card Game”: estrutura de 30 cartas,
  ataque/defesa/magia e progressão por jogo;
- relatos de memória da comunidade de Heróis e Vampiros/Vampiromania no
  Reddit: a lembrança recorrente é a mesa direta, a coleção e o duelo social;
- Riot Games, “Bringing Features to Life in Legends of Runeterra”:
  experimentação, prototipagem e consistência de interface;
- Google, “Playables design best practices”: tutorial breve, estados de
  controle distintos, 48 dp, teclado, contraste, resposta tátil e fim de jogo
  inequívoco;
- Marvel Snap, materiais oficiais: reduzir espera, partidas rápidas, retorno
  imediato e espetáculo audiovisual concentrado nos momentos decisivos.

As fontes são referência de gênero e ergonomia. Nomes, regras, textos,
personagens, ilustrações, molduras, áudio e código de Nythara continuam
originais.

## 2. Diagnóstico do Alpha 0.9.1

O motor já é simples, justo e rápido, mas a apresentação ainda cobra atenção
demais do jogador:

1. a introdução explica cinco telas antes de oferecer experiência prática;
2. a ação permitida aparece em texto, porém compete visualmente com a arena;
3. a mão mostra custo e tipo, mas não mostra a estatística decisiva;
4. o jogador não vê com antecedência quanto Vitalidade restará ao pagar;
5. o banner de início do turno já existia no código, mas não era renderizado;
6. Avatares são descritos como cosméticos, mas na mesa ainda aparecem como um
   ícone genérico;
7. o número de Vitalidade não comunica proporcionalmente perigo e pressão;
8. mouse, toque e teclado não têm uma linguagem de controle compartilhada;
9. o resultado mostra números, mas ensina pouco sobre a decisão que encerrou a
   partida.

## 3. Pacote implementável sem tocar na regra

### P0 — entender e agir

- reduzir a introdução a três decisões e levar diretamente ao treino;
- dar à fase atual uma chamada inequívoca: “sua vez”, objetivo e quantidade
  de cartas jogáveis;
- iluminar apenas cartas autorizadas pela engine e mostrar Poder, Prevenção ou
  Efeito de Rito na mão;
- ao apontar/focar uma carta, prever custo, Vitalidade restante e resultado
  base da Guarda;
- habilitar 1–7 para as cartas, Espaço para passar e Esc para fechar painéis;
- mostrar ajuda de controles dentro da Arena, com opção de ocultar dicas.

### P0 — sentir a mesa

- renderizar o anúncio de turno e manter o confronto como foco cinematográfico;
- mostrar barra de Vitalidade e emblema próprio do Avatar;
- reforçar compra/entrada da mão, estados jogável/bloqueado e perigo de vida;
- usar som sintetizado e vibração opcional, sempre respeitando movimento
  reduzido e preferências locais;
- manter a carta perdedora estilhaçando e o dano viajando até o duelista.

### P1 — voltar a jogar

- “Treino instantâneo” na Home com o deck ativo;
- fila deixa o treino visível sem esperar dois minutos;
- pós-partida destaca maior golpe, custo pago, prevenção e uma lição curta;
- próximo duelo, Crônica e início ficam a uma ação do resultado.

## 4. Critérios de aceite

- um usuário novo alcança o treino em no máximo três decisões após criar a
  conta;
- em qualquer janela, a tela diz quem age e o que acontece ao passar;
- nenhuma carta desabilitada parece jogável;
- o jogador consegue completar uma partida somente com mouse/toque ou somente
  com teclado;
- custo e dano têm cores, posição e rótulos diferentes;
- o duelo funciona em desktop e viewport móvel sem scroll da mesa;
- movimento reduzido, alto contraste, texto ampliado, som e vibração são
  independentes;
- build web, suíte com race detector, E2E real e uma partida completa no
  navegador permanecem verdes.

## 5. Próximas ondas após telemetria humana

Não entram automaticamente neste pacote porque precisam de dados de jogador,
assets profissionais ou decisão econômica:

- campanha PvE episódica com encontros que ensinam uma mecânica por vez;
- trilha musical adaptativa e vozes originais para Avatares;
- variantes cosméticas/foil, versos e arenas sem venda de poder;
- desafios semanais com decks temporários iguais para todos;
- torneios, espectador e compartilhamento social de Crônicas;
- matchmaking por faixa de rating quando a população justificar;
- telemetria de tempo até primeira carta, hesitação por fase, desistência e
  revanche para calibrar o UX com comportamento real.

## 6. Segunda onda — interação recorrente

Esta onda continua dentro do mesmo ruleset e fecha quatro atritos observáveis:

- arrastar uma carta jogável para o centro, com clique e teclado preservados;
- paisagem sonora adaptativa opcional, iniciada somente após gesto do usuário;
- treino imediato no resultado, sem voltar à fila para repetir a prática;
- leitura consultiva do baralho com composição, curva, ataque, defesa e
  variedade de Ritos, sem automatizar decisões do jogador.

Critérios adicionais: soltar fora do limiar cancela sem comando; uma carta não
autorizada não inicia gesto de jogo; áudio desligado não cria nós ativos; o
treino direto usa o deck ativo validado; e o assistente nunca usa linguagem de
garantia de vitória ou replica a autoridade da engine.

## 7. Terceira onda — aprender fazendo e compartilhar com segurança

- onboarding e tutorial podem abrir um treino marcado localmente como guiado;
- um coach acompanha Assalto, Guarda, Rito, espera e fim sem bloquear ação ou
  prescrever carta específica;
- o resultado mostra quais fundamentos realmente apareceram nos eventos da
  partida, mantendo pendências honestas;
- um resumo social copiável usa apenas resultado e agregados pessoais, sem
  expor oponente, cartas, IDs ou a Crônica privada.

Critérios: treino guiado e treino comum usam o mesmo endpoint e ruleset; sair
do coach não concede nem altera nada; refresh da mesma sessão recupera a marca;
checklist deriva do log authoritative; e falha de clipboard produz mensagem
visível sem afetar o resultado.

## 8. Quarta onda — replay visual e confronto legível

- reconstruir o estado visível da mesa a cada evento redigido, com duelistas,
  Vitalidade, fase, contadores conhecidos, Assalto, Guarda, dano e estilhaço;
- oferecer play/pausa, passo anterior/próximo, scrub, velocidade e atalhos de
  teclado, mantendo a linha do tempo clicável como navegação complementar;
- desacelerar voo, impacto e destruição por padrão e oferecer ritmo local
  cinematográfico, normal ou rápido;
- ampliar a zona central em telas largas e mostrar claramente onde entram o
  Assalto e a Guarda antes da primeira jogada;
- enriquecer a trilha de fases com uma frase curta sobre o papel de cada
  decisão, sem competir com a chamada de ação vigente.

Critérios: o replay nunca revela informação ausente no log redigido; o cursor
altera carta, Vitalidade e fase quando os eventos correspondentes existem;
play avança sozinho e pausa de forma estável; velocidade não altera regra nem
estado do servidor; movimento reduzido prevalece sobre o ritmo escolhido; a
Arena continua utilizável em 1280×720 e aproveita telas largas sem esconder a
mão ou a ação principal.

## 9. Quinta onda — memória persistente do duelo

- permitir abrir o replay visual de qualquer partida autorizada do histórico,
  inclusive após atualizar a página ou entrar novamente;
- usar exclusivamente o endpoint redigido da partida, sem depender da memória
  local da última sessão e sem reconstruir informação oculta;
- oferecer **Rever duelo** e **Crônica** lado a lado na Arena, separando a
  reprodução evento a evento do resumo analítico;
- levar da Crônica ao replay correspondente e devolver o jogador à Arena sem
  perder contexto;
- enriquecer cada linha do histórico com o modo, o motivo do encerramento e
  ações com alvos de toque claros.

Critérios: deep link `/replay/:matchId` sobrevive a refresh; uma partida fora
do histórico autorizado continua bloqueada pela API; treino e ranqueada usam o
mesmo projetor visual; replay da sessão atual permanece compatível; estado
vazio, carregamento e erro têm saída para a Arena; desktop, mobile, teclado e
leitores de tela alcançam as duas leituras do duelo.

## 10. Sexta onda — encerramento com significado

A tela de resultado deve fechar o arco emocional e ensinar sem transformar a
derrota em punição textual. Ela apresenta o placar com os dois Avatares, a
Vitalidade real, o motivo autoritativo do encerramento e quatro métricas
legíveis. A leitura tática explica o principal ponto de decisão usando somente
eventos confirmados.

As ações seguem uma hierarquia única: jogar novamente, buscar rival ou estudar
o duelo. Replay e Crônica deixam de competir com utilidades como copiar e
voltar ao início. Em telas pequenas, o mesmo conteúdo vira uma coluna sem
perder o placar, a análise ou os próximos passos.
