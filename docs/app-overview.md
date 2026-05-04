# App Overview — Internal Developer Platform (IDP)

## 🎯 Objetivo

Criar uma **plataforma interna padronizada** para deploy e execução de aplicações, com foco em:

* Segurança (AppSec by default)
* Padronização
* Escalabilidade
* Autonomia dos times (self-service)

> A plataforma não executa infraestrutura enviada pelos times.
> Ela recebe **intenção** e decide **como executar** com controle.

---

## 🧠 Conceito central

**ANTES (problema atual):**

* Times criam suas próprias ferramentas
* Cada um usa padrões diferentes
* Risco de segurança alto
* Baixa visibilidade

**DEPOIS (com a plataforma):**

* Deploy padronizado
* Segurança automática
* Governança central
* Experiência simplificada para devs

---

## ⚙️ Modelo mental

> O dev define **"o que quer rodar"**
> A plataforma define **"como isso roda"**

---

## 🔁 Fluxo de alto nível

```text
Dev → CI/CD → Platform API → Queue → Workers → Runtime
```

---

## 🧩 Componentes principais

### 1. API da Plataforma

Responsável por:

* Receber requisições de deploy
* Validar payload
* Publicar eventos na fila

---

### 2. Orquestração via eventos

* Arquitetura orientada a eventos
* Fila (RabbitMQ)
* Cada etapa do deploy é um worker separado

---

### 3. Workers (pipeline interna)

Pipeline padrão:

1. Build (opcional)
2. Security Scan
3. Policy Check
4. Deploy
5. Pós-deploy (observabilidade)

---

### 4. Runtime

Inicialmente:

* Docker

Futuro:

* Kubernetes

---

### 5. Observabilidade

* Logs centralizados
* Métricas
* Healthchecks

---

## 📦 Entrada da plataforma (payload)

A plataforma recebe **declaração de intenção**, não infraestrutura:

```json
{
  "app": "payments-api",
  "image": "registry.local/payments-api:1.0",
  "replicas": 2,
  "port": 8080,
  "dependencies": ["redis"]
}
```

---

## 🚫 O que NÃO é permitido

* Envio de docker-compose
* Controle direto de rede
* Montagem de volumes arbitrários
* Execução irrestrita de containers

> Isso garante segurança e governança.

---

## ✅ O que a plataforma faz automaticamente

* Scan de vulnerabilidades
* Injeção de secrets
* Configuração de rede
* Definição de limites (CPU/memória)
* Adição de sidecars (logs, métricas)

---

## 🔐 Segurança (pilar principal)

Toda aplicação deve:

* Passar por scan de segurança
* Não conter secrets hardcoded
* Usar imagens válidas/aprovadas
* Seguir políticas da plataforma

---

## 🔁 Pipeline interna (event-driven)

Exemplo:

```text
deploy_requested
    ↓
build_requested
    ↓
security_scan
    ↓
policy_check
    ↓
deploy
    ↓
running
```

---

## 🧪 Exemplo prático

### Dev faz:

```bash
git push
```

### CI faz:

* Build da imagem
* Push para registry
* Chamada para API da plataforma

---

### Plataforma faz:

* Validação
* Segurança
* Deploy
* Monitoramento

---

## 🧠 Dependências (ex: Redis)

Quando o dev envia:

```json
"dependencies": ["redis"]
```

A plataforma:

* Provisiona serviço gerenciado
* Injeta configuração automaticamente

---

## 📊 Resultado final

Dev recebe:

```json
{
  "status": "running",
  "url": "https://payments.internal"
}
```

---

## 🧱 Stack tecnológica

* Go (API + workers)
* RabbitMQ (fila)
* MongoDB (persistência futura)
* Docker (execução inicial)

---

## 🚀 Roadmap resumido

### MVP

* API + fila + worker
* Deploy básico
* Scan simples

### Evolução

* Persistência
* Segurança real (Trivy)
* Observabilidade
* CLI interna
* Portal (Backstage-like)

---

## 🔥 Insight final

Essa plataforma é:

> Uma camada de controle entre o desenvolvimento e a infraestrutura

Ela transforma:

❌ Deploy livre e inseguro
em
✅ Deploy controlado, auditável e seguro

---

## 🧭 Decisão crítica

> A plataforma controla o runtime — não o desenvolvedor

Isso é essencial para:

* segurança
* compliance
* padronização

---

## 📌 Resumo

Você está construindo:

* Uma Internal Developer Platform
* Baseada em eventos
* Com segurança embutida
* E foco em self-service

---

**Se você esquecer tudo amanhã, lembre disso:**

> "Não executo o que o dev manda.
> Eu interpreto, valido e controlo como roda."
