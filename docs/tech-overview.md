# Technical Overview — Internal Developer Platform (IDP)

## 🎯 Objetivo técnico

Construir uma **plataforma interna orientada a eventos** que:

* Orquestra deploys de aplicações
* Centraliza controles de segurança (AppSec)
* Padroniza execução (runtime controlado)
* Oferece self-service para times de desenvolvimento

---

# 🧱 Stack Tecnológica

## Core

* **Linguagem:** Go
* **API:** REST (net/http, Gin ou Fiber)
* **Mensageria:** RabbitMQ
* **Banco (fase 2):** MongoDB
* **Execução (MVP):** Docker
* **Execução (futuro):** Kubernetes

---

## Segurança

* Scan de imagens: Trivy
* SAST (futuro): Semgrep
* Secrets: Vault (futuro)

---

## Observabilidade (fase 3)

* Métricas: Prometheus
* Dashboards: Grafana
* Logs: centralizados (ELK ou similar)

---

# 🧩 Arquitetura

```text
[ CI/CD ]
    ↓
[ Platform API ]
    ↓
[ RabbitMQ ]
    ↓
[ Workers ]
    ├── Build
    ├── Security
    ├── Policy
    └── Deploy
    ↓
[ Runtime (Docker/K8s) ]
```

---

# 🔁 Fluxo completo (end-to-end)

## 1. Pipeline do projeto (CI)

Exemplo:

```yaml
- build imagem
- push para registry
- POST /deploy (Platform API)
```

---

## 2. Payload enviado

```json
{
  "app": "payments-api",
  "image": "registry.local/payments-api:abc123",
  "replicas": 2,
  "port": 8080,
  "dependencies": ["redis"]
}
```

**Importante:**

* Payload representa **intenção**
* Não contém infraestrutura (sem docker-compose)

---

## 3. API da plataforma

Responsabilidades:

* Validar payload
* Gerar `deploy_id`
* Persistir (fase 2)
* Publicar evento:

```json
{
  "event": "deploy_requested",
  "deploy_id": "dep_001",
  "payload": {...}
}
```

---

## 4. Pipeline interna (event-driven)

### 4.1 Build Service (opcional)

* Pode ser usado para centralizar builds

---

### 4.2 Security Service

* Scan de imagem (Trivy)
* Bloqueia se HIGH/CRITICAL

Saída:

```json
security_passed
```

ou

```json
security_failed
```

---

### 4.3 Policy Engine

Valida:

* portas permitidas
* limites de recursos
* dependências autorizadas

---

### 4.4 Enriquecimento (core da plataforma)

Transforma payload em spec interna:

```json
{
  "containers": [
    {
      "name": "app",
      "image": "...",
      "port": 8080
    },
    {
      "name": "log-agent",
      "image": "fluent-bit"
    }
  ],
  "resources": {
    "cpu": "200m",
    "memory": "256Mi"
  },
  "secrets": ["DB_PASSWORD"]
}
```

---

### 4.5 Deploy Service

#### MVP (Docker)

* Executa containers diretamente

#### Futuro (Kubernetes)

* Gera Deployment automaticamente

---

### 4.6 Pós-deploy

* Healthcheck
* Registro de status
* Logs e métricas

---

# ⚙️ Protótipo implementado (Go + RabbitMQ)

## Estrutura

```bash
api/
worker/
shared/
docker-compose.yml
```

---

## Fluxo implementado

1. API recebe `/deploy`
2. Publica mensagem na fila
3. Worker consome
4. Executa pipeline:

```text
BUILD → SECURITY → DEPLOY
```

---

## Exemplo de execução

### Request:

```bash
curl -X POST http://localhost:8080/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "app": "payments-api",
    "image": "registry.local/payments:1.0",
    "replicas": 2,
    "port": 8080
  }'
```

---

### Worker processa:

```text
[BUILD] OK
[SECURITY] OK
[DEPLOY] Container iniciado
[SUCCESS] payments-api rodando
```

---

# 🚫 Decisões de arquitetura (importantes)

## NÃO suportar:

* docker-compose direto
* volumes arbitrários
* network custom
* execução irrestrita

## Motivo:

* segurança
* padronização
* governança

---

# 🧠 Modelo de dados (fase 2)

## Collection: apps

```json
{
  "id": "app_123",
  "name": "payments-api",
  "owner": "team-x"
}
```

---

## Collection: deploys

```json
{
  "id": "dep_001",
  "app_id": "app_123",
  "status": "running",
  "created_at": "..."
}
```

---

# 🚀 Plano de implementação

## 🥇 Fase 1 — MVP (2–4 semanas)

### Objetivo:

Validar arquitetura e fluxo

### Entregas:

* API Go (`/deploy`)
* Integração com RabbitMQ
* Worker único (pipeline simples)
* Execução simulada ou Docker básico

### Critérios de sucesso:

* Deploy assíncrono funcionando
* Pipeline executando via fila

---

## 🥈 Fase 2 — Base sólida (4–6 semanas)

### Objetivo:

Adicionar controle e rastreabilidade

### Entregas:

* Persistência com MongoDB
* Status de deploy (`GET /deploy/:id`)
* Separação de workers:

  * build
  * security
  * deploy
* Integração com Trivy

### Critérios:

* Bloqueio de deploy inseguro
* Histórico de deploys

---

## 🥉 Fase 3 — Plataforma utilizável (6–10 semanas)

### Objetivo:

Adoção por times piloto

### Entregas:

* Autenticação (JWT ou SSO)
* RBAC básico
* Observabilidade:

  * métricas
  * logs
* Healthcheck automático

### Critérios:

* Times usando em produção controlada

---

## 🏗️ Fase 4 — Escala e DX

### Entregas:

* CLI interna (`platform deploy`)
* Templates de projeto
* Catálogo de serviços
* Portal (Backstage-like)

---

## ☸️ Fase 5 — Evolução para Kubernetes

### Entregas:

* Deploy via manifests gerados
* Autoscaling
* Isolamento por namespace

---

# 📊 Métricas de sucesso

* Tempo médio de deploy
* % de apps na plataforma
* % de deploys bloqueados por segurança
* Tempo de correção de vulnerabilidades
* Adoção por times

---

# 🔐 Controles obrigatórios

* Scan de imagem antes do deploy
* Secrets fora do código
* Auditoria de deploy
* Logs centralizados

---

# 🔥 Insight final

Essa plataforma implementa:

> **Deploy como evento + execução controlada pela plataforma**

---

# 📌 Regra de ouro

> O desenvolvedor descreve a intenção
> A plataforma controla a execução

---

# 🧭 TL;DR para Tech Lead

* Arquitetura orientada a eventos (RabbitMQ)
* Pipeline desacoplado por workers
* Segurança embutida no fluxo
* Runtime controlado (Docker → Kubernetes)
* Evolução incremental (MVP → Plataforma completa)

---

Se você esquecer tudo:

> “Nunca execute diretamente o que o time manda.
> Sempre valide, enriqueça e controle antes de rodar.”
