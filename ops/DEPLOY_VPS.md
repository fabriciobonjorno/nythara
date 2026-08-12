# Deploy do Nythara em VPS

Esta stack atende uma VPS Linux única com Docker Compose. Ela publica somente
HTTP/HTTPS pelo Caddy; API e PostgreSQL não abrem portas no host. O Caddy obtém
e renova o certificado TLS automaticamente quando o DNS já aponta para a VPS.

## 1. Requisitos e DNS

- VPS amd64 ou arm64 com 2 vCPU, 4 GB de RAM e 40 GB SSD como ponto inicial;
- Docker Engine com o plugin Compose;
- portas TCP 22, 80 e 443 e UDP 443 liberadas no firewall;
- registro DNS A (e AAAA, se a VPS tiver IPv6 funcional) apontando o domínio
  raiz para a VPS; `www` pode ser um CNAME para o domínio raiz e será
  redirecionado para a origem canônica;
- usuário de deploy sem login direto como root e com acesso controlado ao
  Docker.

Não exponha 5432, 8080 ou as rotas `/internal/*` no firewall ou em outro proxy.
O grupo `docker` equivale, na prática, a privilégio de root; limite quem entra
nele e mantenha SSH por chave, sem senha.

## 2. Preparar configuração e segredos

No checkout da VPS:

```bash
cp .env.production.example .env.production
./ops/bootstrap-vps-secrets.sh
```

Edite apenas `NYTHARA_DOMAIN` e um identificador imutável de imagem em
`.env.production`. O bootstrap cria dois arquivos `0600` em `secrets/`:

- `postgres_password`: senha aleatória do usuário do PostgreSQL;
- `database_url`: URL interna completa consumida pela API e pelo migrador.

Esses arquivos e `.env.production` são ignorados pelo Git. Em uma instalação
mais rígida, defina `NYTHARA_SECRETS_DIR=/etc/nythara/secrets`, mantenha o
diretório fora do checkout e ajuste proprietário/permissões para o operador do
Compose. Nunca copie os valores para issue, log, chat ou linha de comando.

## 3. Validar e subir

```bash
docker compose --env-file .env.production -f compose.production.yml config --quiet
docker compose --env-file .env.production -f compose.production.yml build --pull
docker compose --env-file .env.production -f compose.production.yml up -d
docker compose --env-file .env.production -f compose.production.yml ps
```

O fluxo é ordenado: PostgreSQL saudável → migrações concluídas → API saudável
→ edge. Se uma migração falhar, a API não inicia. Antes de trocar uma versão em
produção, rode os gates descritos em `CLAUDE.md`, crie um backup e use o SHA do
commit em `NYTHARA_IMAGE_TAG`.

Validação externa:

```bash
curl --fail --silent --show-error https://SEU_DOMINIO/healthz
curl --fail --silent --show-error https://SEU_DOMINIO/readyz
curl --fail --silent --show-error https://SEU_DOMINIO/version
curl --fail --silent --show-error https://SEU_DOMINIO/internal/metrics
```

As três primeiras devem responder; a última deve retornar 404. Confirme também
cadastro, login, criação do baralho, treino, reconexão WebSocket, replay e
logout em uma conta de teste. Remova a conta de teste antes de abrir o Alpha.

## 4. Backup, restauração e retenção

Crie um backup antes de cada deploy e diariamente:

```bash
./ops/vps-backup.sh
```

O caminho do dump é impresso no final. Copie os dumps criptografados para outro
provedor ou volume: backup no mesmo disco da VPS não protege contra perda da
máquina. O repositório inclui unidades de exemplo em `ops/systemd/`; ajuste
`WorkingDirectory`, usuário e grupo antes de instalá-las.

Teste restauração periodicamente em uma stack isolada. Não restaure sobre a
base ativa. O procedimento seguro é criar um PostgreSQL temporário, aplicar o
dump com `pg_restore`, iniciar a mesma versão da API contra ele e validar
`readyz`, catálogo, contagens e um replay histórico. O teste local automatizado
continua disponível em `make backup-test`.

Defina retenção fora do script (por exemplo, 7 diários, 4 semanais e 6 mensais)
somente depois de confirmar a cópia remota. Exclusão automática sem prova de
restauração não é retenção segura.

## 5. Atualização e rollback

Atualização:

1. rode `make test-race`, testes web, scans e backup;
2. faça checkout de uma tag/commit revisado;
3. atualize `NYTHARA_IMAGE_TAG` para o SHA;
4. execute `build --pull` e `up -d`;
5. confira `ps`, `readyz`, logs e um duelo real.

Rollback do binário/web: volte ao commit anterior e repita build/up. Migrações
são aplicadas antes da API e devem ser retrocompatíveis durante a janela de
rollback. Não execute `migrate down` automaticamente: faça isso apenas com
backup validado e uma análise específica da migração.

## 6. Operação e observabilidade

```bash
docker compose --env-file .env.production -f compose.production.yml ps
docker compose --env-file .env.production -f compose.production.yml logs --since=30m api web postgres
docker compose --env-file .env.production -f compose.production.yml restart api
```

Os logs JSON têm rotação local. Encaminhe-os para armazenamento externo antes
do Alpha público e conecte `OTEL_EXPORTER_OTLP_ENDPOINT` somente a um coletor
confiável na rede privada. Siga os alertas de `ops/observability.md`: prontidão,
5xx, 429, panics, flood de WebSocket, latência, espaço em disco, conexões do
banco, expiração do certificado e idade do último backup.

## 7. Checklist de lançamento

- DNS e TLS válidos, sem aviso no navegador;
- apenas 22/80/443 TCP e 443 UDP públicos;
- `.env.production` e segredos `0600`, ausentes do Git e do backup do código;
- `/internal/metrics` e `/internal/implementation-report` retornam 404 no edge;
- `make test-race`, `make lint`, build/testes web e scans verdes;
- migração e `readyz` verdes após reinício completo da VPS;
- backup externo recente e restauração provada;
- conta administrativa separada, sem uso cotidiano e sem credencial
  compartilhada;
- termos/privacidade, canal de suporte e política de retenção definidos antes
  de coletar dados de jogadores reais;
- busca de marca “Nythara” concluída antes do lançamento público.
