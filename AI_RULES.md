# AI_RULES.md

Limites rígidos para agentes de IA:

1. **Sem mudança arquitetural silenciosa.** Alterações de arquitetura, formato
   de eventos, schema de dados ou regras de jogo exigem ADR em `DECISIONS.md`
   no mesmo commit.
2. **Determinismo é sagrado.** Proibido em lógica de jogo: `time.Now()`,
   `math/rand`, iteração de map, goroutines, IO. Replays antigos precisam
   continuar reproduzindo (regras versionadas por `RulesetVersion`).
3. **Nada silenciosamente ignorado.** Efeito não suportado = rejeição explícita
   + relatório. Teste pulado = justificativa escrita.
4. **Goldens só mudam com revisão.** `make golden` apenas após explicar o diff.
5. **Segurança:** cliente nunca decide resultado; segredos fora do repositório;
   economia nunca confia no cliente.
6. **IP:** nenhum conteúdo copiado de jogos existentes; nomes/textos/arte
   sempre originais.
