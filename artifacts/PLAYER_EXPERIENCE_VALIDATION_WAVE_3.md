# Validação de experiência — onda 3 do Alpha 0.9.1

Data: 2026-08-11  
Escopo: treino guiado reativo, checklist pós-partida e resumo social seguro.

## Resultado

**APROVADO.** O tutorial hands-on usa a prática e os eventos existentes; não
houve mudança de ruleset, engine, protocolo ou balanceamento.

## Jornada real no navegador

Uma conta nova foi criada pela UI e percorreu o fluxo completo:

1. onboarding avançado em três telas e “Treinar agora” acionado;
2. prática aberta diretamente com a marca de treino guiado na sessão;
3. como o bot recebeu a iniciativa, o coach começou corretamente em Guarda,
   mostrando o Poder 4 do Assalto que estava no centro;
4. uma Guarda foi jogada e o progresso mudou para 1/3 somente após o evento
   `guard_committed` confirmado;
5. na janela seguinte o coach mudou para Assalto; depois do confronto passou
   para 2/3 e ensinou Rito;
6. um Rito real fechou 3/3; nenhuma carta específica foi exigida e passe
   permaneceu disponível;
7. a partida terminou naturalmente com vitória em 18 rodadas, 22 cartas
   jogadas, maior golpe 6 e 25 de dano pessoal;
8. o resultado exibiu os três fundamentos como concluídos, cada um derivado
   dos eventos da partida;
9. “Repetir treino guiado” e “Buscar rival” permaneceram escolhas distintas;
10. “Copiar resumo seguro” concluiu com confirmação visível de que nenhuma
    carta ou informação do rival foi incluída.

Inspeção visual em 1280×720 confirmou que o coach ocupa o canto superior
direito sem cobrir a zona central, a decisão ou a mão. O resultado manteve
checklist, retorno e ações sociais em hierarquia legível.

## Gates automatizados

- `make test-race`: todos os pacotes aprovados.
- `make lint`: `go vet ./...` aprovado.
- `npm run build --prefix web`: TypeScript e Vite aprovados.
- `git diff --check`: aprovado.
- `make e2e-real`: aprovado; PvP natural em 8 rodadas/17 cartas e treino em 8
  rodadas/10 cartas humanas. Confronto central, estilhaço, mão oculta, rating,
  prática sem rating, unicidade e formato de identidade permaneceram válidos.
- `make sim-smoke`: 1.000/1.000, iniciativa 49,60%, média 12,42, p95 22 e
  saúde 0/0/0/0/0/0.

Os gates oficiais anteriores de 200 mil partidas permanecem válidos porque a
onda altera somente apresentação local.

## Privacidade e limites

- A marca guiada contém somente o ID da partida e fica em `sessionStorage`.
- O resumo copiável contém resultado, rodadas, dano, prevenção e maior golpe;
  não contém ID, nome do rival, cartas, mão, baralho, e-mail ou link privado.
- A Crônica completa continua autenticada e restrita aos participantes.
- O coach pode ser encerrado a qualquer momento sem mudar eventos, recompensa
  ou estado do duelo.
