# Plano de Desenvolvimento — IDP POC

> POC de Internal Developer Platform rodando em VPS, com Traefik para roteamento por subdomínio, RabbitMQ como event bus e Docker como runtime inicial.

## 🎯 Objetivo da POC

Endpoint único (`POST /deploy`) que recebe a **intenção** de uma aplicação completa (app + dependências como banco e cache) e provisiona tudo em containers, com TLS automático via Let's Encrypt e roteamento por subdomínio.

## 📐 Regra de ouro

> O desenvolvedor descreve a intenção. A plataforma decide como executa.

Concretamente:

- Payload aceita só campos declarativos: `app`, `image`, `replicas`, `port`, `dependencies`.
- Sem `docker-compose`, sem volumes arbitrários, sem rede customizada, sem labels Traefik no payload.
- Sidecars, limites de recurso, secrets, labels de roteamento e provisão de deps são **adicionados pela plataforma na etapa de enriquecimento**.

## 🧭 Decisões da POC

| Tema | Decisão |
| --- | --- |
| Subdomínio | Plataforma deriva `<payload.app>.<IDP_BASE_DOMAIN>`. Validação `^[a-z][a-z0-9-]{1,30}$`. Conflito de nome → `409`. |
| DNS | Fora da plataforma por enquanto. Wildcard A record `*.<IDP_BASE_DOMAIN>` apontando pra VPS. Worker `dns_provision` entra depois sem mexer no resto. |
| Auth | Bearer único (`IDP_API_TOKEN`), comparação timing-safe. RBAC fica pra Fase 3. |
| Allowlist de dependências | `redis`, `postgres`, `mongo`. Cada uma vira arquivo em `shared/spec/catalog/` na Fase 2. |
| TLS | Let's Encrypt com TLS-ALPN challenge. Staging primeiro, flip pra produção removendo a flag `--certificatesresolvers.letsencrypt.acme.caserver`. |
| Runtime | Docker via SDK Go (`github.com/docker/docker/client`). Kubernetes fica fora do escopo da POC. |

## 📊 Status

| Fase | Status | Notas |
| --- | --- | --- |
| 0 — Scaffold | ✅ Concluída (2026-05-04) | Estrutura de pastas, compose, Dockerfiles, esqueleto Go com `/health` + Bearer |
| 1 — Fluxo MVP end-to-end | ⏳ Próxima | Validação de payload, publicação na fila, worker consumindo, deploy básico |
| 2 — Enriquecimento + Traefik + Deps | 📋 Planejada | Catálogo de deps, geração de labels, rede por deploy, persistência Mongo |
| 3 — Workers separados + Trivy real | 📋 Planejada | Quebra do worker em processos, scan real, timeline de eventos |
| 4 — Operação na VPS | 📋 Planejada | LE produção, restart policies, backups, bootstrap script |

---

## ✅ Fase 0 — Scaffold (concluída)

**Objetivo:** estrutura mínima rodando vazia, validando que api ↔ rabbit ↔ worker sobem juntos atrás do Traefik.

### Estrutura criada

```
.
├── api/
│   ├── main.go                 # http.ServeMux, graceful shutdown, fail-fast em env
│   ├── handler/
│   │   ├── health.go           # GET /health (público)
│   │   └── deploy.go           # POST /deploy + GET /deploy/{id} — stubs 501
│   └── middleware/
│       └── auth.go             # Bearer com subtle.ConstantTimeCompare
├── worker/
│   └── main.go                 # Conecta no RabbitMQ, idle até Fase 1
├── shared/
│   ├── events/events.go        # Constantes dos eventos (deploy_requested, ...)
│   └── queue/queue.go          # Wrapper amqp091 (Dial/Channel/Close)
├── docker-compose.yml          # traefik + rabbitmq + api + worker
├── Dockerfile.api              # multi-stage Go 1.23 → distroless:nonroot
├── Dockerfile.worker           # multi-stage Go 1.23 → distroless
├── .env.example
├── .gitignore
└── go.mod                      # github.com/ixcsoft/idp, amqp091-go v1.10.0
```

### Componentes do compose

- **traefik** (`traefik:v3.1`) — provider Docker em modo read-only, redirecionamento 80→443, dashboard em `traefik.<domínio>` com basic auth, Let's Encrypt staging.
- **rabbitmq** (`rabbitmq:3.13-management-alpine`) — credenciais via env, healthcheck com `rabbitmq-diagnostics ping`, volume persistente.
- **api** — exposta em `api.<domínio>` via labels Traefik, na rede `traefik` + `idp-internal`, depende do RabbitMQ saudável.
- **worker** — só na rede `idp-internal`, com `/var/run/docker.sock` montado rw (necessário pra criar containers de apps; aceitável pra POC, hardening com `docker-socket-proxy` depois).

### Verificações feitas

- `docker compose config` valida sintaxe.
- Estrutura de pastas reflete o layout descrito em [tech-overview.md](../tech-overview.md).

### Pendências para o usuário antes da Fase 1

1. `cp .env.example .env` e preencher (`IDP_BASE_DOMAIN`, `IDP_API_TOKEN`, `LETSENCRYPT_EMAIL`, senha do RabbitMQ, htpasswd do dashboard).
2. Apontar A record wildcard `*.<IDP_BASE_DOMAIN>` pra VPS.
3. `docker compose build && docker compose up -d`.
4. Confirmar que `https://api.<domínio>/health` responde `{"status":"ok"}` (com warning de cert por causa do staging).

---

## ⏳ Fase 1 — Fluxo MVP end-to-end

**Objetivo:** `POST /deploy` → fila → worker → container rodando atrás do Traefik com TLS.

### Entregas

- **Validador de payload** (`api/validator/payload.go`)
  - Aceita só `app`, `image`, `replicas`, `port`, `dependencies`. Rejeita campos extras com 400.
  - `app` casa `^[a-z][a-z0-9-]{1,30}$`.
  - `image` aceita `<registry>/<path>:<tag>`, recusa `:latest`.
  - `port` em range razoável (1024–65535).
  - `dependencies` ⊆ allowlist (`redis`, `postgres`, `mongo`).

- **Handler `POST /deploy`**
  - Gera `deploy_id` (ULID, biblioteca `github.com/oklog/ulid/v2`).
  - Publica evento `deploy_requested` no exchange `idp.deploys` com routing key `deploy.requested`.
  - Retorna `202 { deploy_id, status: "queued", url: "https://<app>.<domínio>" }`.

- **Handler `GET /deploy/{id}`**
  - Lê status de um store em memória (`sync.Map`); Mongo entra na Fase 2.
  - 404 se não existir.

- **Worker único** (`worker/pipeline/`)
  - Consome `deploy.requested` e roda os estágios sequencialmente, **publicando cada evento na fila** (mesmo consumido pelo próprio processo). Isso permite separar em processos na Fase 3 sem reescrever lógica.
  - Estágios:
    1. `security_scan` — stub que sempre passa (Trivy real na Fase 3).
    2. `policy_check` — porta no range, deps na allowlist.
    3. `enrich` — stub que retorna spec mínima (catálogo real na Fase 2).
    4. `deploy` — `docker run` real via SDK Go com labels Traefik, na rede `traefik`.
  - Em qualquer falha, publica `failed` com motivo e marca o deploy.

- **Status reporter**
  - Worker emite atualizações de status via fila de feedback (`idp.status`); API consome e atualiza o store.

### Critérios de sucesso

- `curl -H "Authorization: Bearer ..." -X POST https://api.<domínio>/deploy -d '{...}'` retorna 202.
- Após ~30s, `https://<app>.<domínio>` serve a aplicação.
- `GET /deploy/{id}` mostra `running`.
- Payload com campo extra retorna 400 com a lista de campos rejeitados.
- Token errado retorna 401.

### Fora do escopo da Fase 1

- Provisão de dependências (Fase 2).
- Persistência (Fase 2).
- Trivy real (Fase 3).
- Sidecars (Fase 2).

---

## 📋 Fase 2 — Enriquecimento + Traefik + Dependências

**Objetivo:** transformar `IntentPayload` em `InternalSpec` completa, com deps provisionadas e wiring automático. Esse é o **núcleo da plataforma**.

### Entregas

- **Catálogo de dependências** (`shared/spec/catalog/`)
  - Um arquivo por dep: `redis.go`, `postgres.go`, `mongo.go`.
  - Cada um define: imagem, env vars geradas (com senhas aleatórias), env vars injetadas no app (`REDIS_URL`, `DATABASE_URL`, `MONGO_URL`), comando de healthcheck.

- **Enriquecimento** (`worker/pipeline/enrich.go`)
  - Cria nome de rede dedicado: `idp-<app>-<deploy_id_short>`.
  - Materializa containers de deps a partir do catálogo.
  - Monta container do app com:
    - rede do deploy + rede `traefik`
    - env vars das deps injetadas
    - limites default (`--memory=256m --cpus=0.5`)
    - **labels Traefik geradas pela plataforma:**
      ```
      traefik.enable=true
      traefik.docker.network=traefik
      traefik.http.routers.<app>.rule=Host(`<app>.<domínio>`)
      traefik.http.routers.<app>.entrypoints=websecure
      traefik.http.routers.<app>.tls.certresolver=letsencrypt
      traefik.http.services.<app>.loadbalancer.server.port=<port>
      ```

- **Deploy service** (`worker/pipeline/deploy.go`)
  - Cria a rede `idp-<app>-<id>`.
  - Sobe deps **antes** do app, healthcheck TCP.
  - Sobe o app na rede do deploy + na rede `traefik`.
  - Em falha: rollback (remove containers + rede).

- **Persistência MongoDB**
  - Collections `apps`, `deploys`, `deploy_events` (audit trail do pipeline).
  - Schema conforme [tech-overview.md §"Modelo de dados"](../tech-overview.md).
  - API troca o `sync.Map` pelo Mongo.

### Critérios de sucesso

- Deploy com `"dependencies": ["redis", "postgres"]` sobe redis + postgres + app, todos na mesma rede privada.
- App recebe `REDIS_URL` e `DATABASE_URL` via env, dev nunca escreve isso.
- Falha no startup do postgres → rollback completo, deploy marcado `failed`, sem containers órfãos.
- `GET /deploy/{id}` mostra timeline dos eventos.

---

## 📋 Fase 3 — Workers separados + Trivy real

**Objetivo:** quebrar o worker monolítico em processos por estágio, validando o desacoplamento por eventos. Adicionar segurança real.

### Entregas

- **Binários separados** dentro do mesmo módulo Go:
  - `worker/security/` (consome `security.scan`)
  - `worker/policy/` (consome `policy.check`)
  - `worker/deploy/` (consome `deploy.execute`)
- **Compose** ganha 3 services novos (todos compartilham o mesmo Dockerfile via build target).
- **Trivy real**
  - Service `security` invoca `trivy image --severity HIGH,CRITICAL --exit-code 1` no container.
  - `security_failed` curto-circuita o pipeline.
- **Endpoint de auditoria**
  - `GET /deploy/{id}/events` retorna timeline da collection `deploy_events`.

### Critérios de sucesso

- Imagem com CVE HIGH não passa do estágio security; deploy fica `failed` com motivo.
- Cada worker pode ser reiniciado isoladamente sem perder a fila.
- Timeline mostra ordem e timestamps dos eventos.

---

## 📋 Fase 4 — Operação na VPS

**Objetivo:** deixar a POC operável de verdade.

### Entregas

- Let's Encrypt em **produção** (remove flag staging).
- `restart: unless-stopped` em tudo (já está, validar).
- Backup periódico dos volumes RabbitMQ + Mongo (cron + `tar` + offsite).
- Script `scripts/bootstrap.sh` que sobe a POC do zero numa VPS limpa (instala Docker, clona repo, gera token, sobe compose).
- Documentação operacional (`docs/runbook.md`): como reiniciar, como ver logs, como rotacionar token, como restaurar backup.

### Critérios de sucesso

- VPS limpa → POC rodando em < 10 minutos via `bootstrap.sh`.
- Reboot da VPS → tudo volta sozinho.
- Restauração de backup testada manualmente uma vez.

---

## 🚫 Fora do escopo da POC (explicitamente)

- Kubernetes (entra na evolução pós-POC).
- Build de imagem dentro da plataforma (a POC assume que o CI já fez o push).
- SAST/Semgrep (Fase 5+).
- Vault/secrets externos (Fase 5+).
- Multi-tenant / RBAC completo (Fase 5+).
- Portal web (Fase 5+).
- CLI interna (Fase 5+).

---

## 📌 Resumo

A POC entrega o **deploy de uma app completa com dependências, via endpoint único, controlado pela plataforma**. Tudo o que vier além disso fica pra evolução.

> "Não executo o que o dev manda. Eu interpreto, valido e controlo como roda."
