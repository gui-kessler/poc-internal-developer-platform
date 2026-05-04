# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository status

This is a **documentation-only POC** for an Internal Developer Platform (IDP). There is no source code, build system, or test suite yet — only `docs/app-overview.md` and `docs/tech-overview.md` describing the intended design. Do not invent commands; the first implementation work will need to scaffold the Go modules and `docker-compose.yml` described below.

The docs are written in Portuguese (Brazilian). Match that language when extending them.

## Planned stack and layout

Per `docs/tech-overview.md`, the MVP targets:

- **Language:** Go (API + workers)
- **Messaging:** RabbitMQ (event bus between API and workers)
- **Persistence (phase 2):** MongoDB
- **Runtime (MVP):** Docker; **future:** Kubernetes
- **Security tooling:** Trivy (image scan), Semgrep (SAST, future), Vault (secrets, future)

Intended top-level layout:

```
api/            # HTTP API: receives /deploy, publishes events
worker/         # Pipeline workers (build, security, policy, deploy)
shared/         # Shared types, queue/event contracts
docker-compose.yml
```

## Core architectural rule (do not violate)

> The developer describes **intent**. The platform decides **how it runs**.

This is the single most important constraint and it must shape every API, schema, and worker. Concretely:

- The `/deploy` payload accepts only declarative intent: `app`, `image`, `replicas`, `port`, `dependencies`. See `docs/app-overview.md` for the canonical shape.
- The platform **must not accept** docker-compose files, custom networks, arbitrary volume mounts, or any field that lets a caller dictate runtime topology. If you find yourself adding such a field, stop — you're solving the problem at the wrong layer.
- Sidecars (log agent, metrics), resource limits, secrets injection, and network config are **added by the platform during enrichment**, not supplied by the caller.
- Dependencies (e.g. `"dependencies": ["redis"]`) are provisioned and wired by the platform — the caller never sees connection strings in the request.

## Event pipeline

The deploy flow is event-driven through RabbitMQ. The canonical sequence is:

```
deploy_requested → build_requested → security_scan → policy_check → deploy → running
```

Each stage is a **separate worker** consuming and emitting events; do not collapse stages into a monolithic handler. Security must be able to **block** the pipeline (`security_failed` short-circuits the rest) — this is the AppSec-by-default pillar and a hard requirement, not a phase-2 nicety.

The "enrichment" step (tech-overview §4.4) is the heart of the platform: it transforms the caller's intent payload into the internal container spec (containers + sidecars + resources + secrets). Treat this as a first-class component, not glue code.

## Phased roadmap (use to scope work)

When the user asks for a feature, locate it on this roadmap before implementing — phase 3+ items should not land before phase 1/2 foundations exist.

1. **MVP:** API + RabbitMQ + single worker, simulated or basic Docker execution.
2. **Base:** MongoDB persistence, `GET /deploy/:id`, split workers, Trivy integration.
3. **Usable:** Auth (JWT/SSO), RBAC, observability (metrics/logs), healthchecks.
4. **Scale/DX:** Internal CLI (`platform deploy`), project templates, service catalog, Backstage-like portal.
5. **Kubernetes:** Generated manifests, autoscaling, namespace isolation.

## When implementing

- Treat the docs in `docs/` as the spec. If a request conflicts with them (especially the "what is NOT permitted" list in `app-overview.md`), surface the conflict before coding.
- The data model in tech-overview §"Modelo de dados" (collections `apps`, `deploys`) is the target shape for phase 2 — align Go structs with it from the start to avoid a rewrite.
