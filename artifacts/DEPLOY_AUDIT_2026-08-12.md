# Auditoria de prontidão para VPS — Nythara

Data: 12 de agosto de 2026
Escopo: aplicação Go, web React/PWA, PostgreSQL, credenciais, páginas, design responsivo, acessibilidade operacional, imagens de produção, proxy TLS, backup e documentação.

## Veredito

O projeto está **tecnicamente pronto para ser instalado em uma VPS**, com uma pilha de produção reproduzível e validada localmente. Isso não significa que houve deploy em uma VPS real: ainda são necessários domínio, DNS, servidor, credenciais definitivas e a execução do checklist operacional.

Para um lançamento público, permanecem quatro decisões externas ao código:

1. publicar termos de uso e política de privacidade;
2. proteger contas administrativas com MFA ou uma segunda barreira de acesso;
3. configurar cópia externa criptografada dos backups e provar uma restauração periódica;
4. fazer a revisão final de marca, classificação etária e direitos dos assets.

## Achados críticos corrigidos

| Área | Achado | Correção |
| --- | --- | --- |
| Credenciais | API e migrador aceitavam uma URL de banco de desenvolvimento quando `DATABASE_URL` não existia | Inicialização agora falha de forma explícita; segredo pode vir de `DATABASE_URL` ou `DATABASE_URL_FILE`, nunca dos dois |
| Proxy/IP | `X-Forwarded-For` poderia ser aceito sem delimitar proxies confiáveis | Rede do proxy é configurada por CIDR; cabeçalho só é usado quando o peer é confiável e a cadeia é percorrida da direita para a esquerda |
| Superfície web | `/internal/*` podia cair no fallback da SPA | Proxy agora responde 404 antes do fallback |
| Regras/design | README, GDD e builder divergiam do ruleset servido | Documentos e composição do baralho foram alinhados ao alpha 0.13; mínimos são obtidos do ruleset da API |
| Senhas | Documentação afirmava Argon2id, mas o código usa PBKDF2-SHA256 | Documentação corrigida para refletir PBKDF2-SHA256 com 600 mil iterações; migração futura não foi feita silenciosamente |
| Backup | Teste usava porta pública fixa e colidia com serviços locais | Teste agora cria contêiner único, publica somente em `127.0.0.1` e escolhe porta aleatória por padrão |
| Mobile | Cabeçalho a 390 px se sobrepunha; diversos controles tinham alvo menor que 44 px | Marca compactada, textos truncados com segurança e alvos táteis ampliados no shell, construtor, tutorial e mesa |
| Acessibilidade | Duelo e página de acesso administrativo negado não tinham `h1` | Títulos acessíveis adicionados; controles decorativos duplicados do tutorial deixaram de ser botões |

## Credenciais e segurança

- Nenhuma chave privada, token de API, senha de produção ou segredo com padrão forte foi encontrado nos arquivos versionados ou no histórico pesquisado.
- Senhas presentes em Compose de desenvolvimento e CI são marcadas como locais/teste e não são reutilizadas na pilha de produção.
- Tokens de sessão são aleatórios, armazenados no navegador por sessão e persistidos no banco somente por hash; refresh token é rotacionado.
- A pilha de produção usa arquivos de segredo com permissão `0600`; o bootstrap não sobrescreve segredos existentes.
- PostgreSQL não publica porta no host e fica em rede interna.
- API roda como usuário não-root, com filesystem somente leitura, capabilities removidas e diretório temporário em memória.
- Caddy termina TLS e aplica CSP, HSTS, `nosniff`, política de referrer e de permissões.
- Dependências web: `npm audit` retornou **0 vulnerabilidades**.
- Go: `govulncheck` não encontrou vulnerabilidade em caminho chamado pela aplicação. Há uma ocorrência em módulo requerido, mas não importada pelo código executável; deve continuar sendo acompanhada.

Riscos residuais de segurança:

- autenticação ainda não oferece MFA para administradores;
- PBKDF2-SHA256 está corretamente parametrizado, porém recomenda-se migração gradual para Argon2id em um ADR próprio;
- CSP precisa manter `style-src 'unsafe-inline'` enquanto componentes dependerem de estilos inline;
- falta automatizar SBOM e scanner de CVE das imagens no CI.

## Design, páginas e mobile first

Auditoria visual executada em **390 × 844 px**, com inspeção adicional em desktop e em uma partida real.

| Página/fluxo | Resultado móvel |
| --- | --- |
| Entrada, login e criação de conta | Sem overflow e sem asset quebrado; cabeçalho compacto |
| Início | Sem overflow; CTAs e atalhos com alvos adequados |
| Coleção | Sem overflow ou imagens quebradas |
| Avatares | Sem overflow ou imagens quebradas |
| Construtor de baralho | Regras vindas da API; botão de remover ampliado para 44 × 44 px |
| Fila | Sem overflow; guia com alvo tátil adequado |
| Arena | Estado vazio coerente e responsivo |
| Perfil | Hierarquia de título válida e sem overflow |
| Tutorial | Etapas continuam navegáveis; indicadores redundantes são decorativos |
| Configurações | Linhas inteiras são os alvos dos controles; inputs visuais ocultos são intencionais |
| Salão de Controle | Conta comum recebe mensagem explícita, sem dados administrativos |
| 404 | Estado de recuperação com título e retorno ao início |
| Mesa de duelo | Ocupa um viewport sem rolagem vertical da página; mão usa rolagem horizontal intencional; sem imagens quebradas |

Em todas as rotas auditadas: **0 overflow horizontal**, **0 imagens quebradas**, **0 IDs duplicados**. Os alvos de ação relevantes ficaram com ao menos 44 px após as correções.

## Pilha de produção entregue

- `Dockerfile` multi-stage para API e web;
- `compose.production.yml` com PostgreSQL privado, migração one-shot, API e Caddy;
- healthchecks e política de restart;
- redes separadas para dados e borda;
- limites de logs para evitar consumo ilimitado do disco;
- segredos montados por arquivo;
- Caddy com HTTPS automático;
- script idempotente de bootstrap dos segredos;
- backup PostgreSQL e unidade/timer do systemd;
- runbook de instalação, atualização, rollback, backup e restauração.

## Evidências de validação

| Validação | Resultado |
| --- | --- |
| `make test-race` | passou |
| `go test -count=1 ./...` | passou |
| `go vet ./...` | passou |
| build web de produção | passou |
| testes unitários de UI | 8/8 passaram |
| testes reais de navegador | 3/3 passaram |
| `npm audit` | 0 vulnerabilidades |
| `govulncheck` | nenhum caminho vulnerável chamado |
| build das duas imagens Docker | passou |
| inicialização completa isolada | PostgreSQL, migração, API e web saudáveis |
| TLS e headers | passaram |
| `/internal/*` | 404 |
| `/healthz`, `/readyz`, `/v1/version`, ruleset | passaram |
| duelo ponta a ponta | PvP e treino concluíram naturalmente; status `ok` |
| backup → destruição → restore | marcador preservado; 31 tabelas restauradas |

## Pendências não bloqueantes

- Algumas dependências têm versões mais novas disponíveis. Não foi feita atualização em massa durante a auditoria para não introduzir regressões antes do deploy; React 19 é uma migração major e merece trabalho separado.
- O bundle principal web tem cerca de 485 kB (152 kB gzip). Está dentro de uma faixa utilizável, mas divisão por rota e carregamento tardio de efeitos visuais melhorariam a primeira visita em redes móveis lentas.
- Monitoramento está documentado, mas métricas, alertas e retenção dependem da infraestrutura escolhida para a VPS.

## Checklist de liberação

- [ ] VPS com Docker, Compose, firewall e atualizações automáticas
- [ ] domínio e registros DNS apontando para a VPS
- [ ] portas públicas somente 22 (restrita), 80 e 443
- [ ] segredos gerados no servidor; nenhum segredo copiado para Git
- [ ] primeira subida e verificação de health/readiness/version
- [ ] backup externo criptografado configurado
- [ ] restauração executada em ambiente descartável
- [ ] MFA/segunda barreira para administração
- [ ] termos, privacidade, classificação e revisão de marca publicados
- [ ] alertas de disponibilidade, erro, disco, certificado e backup ativos

O procedimento detalhado está em `ops/DEPLOY_VPS.md`.
