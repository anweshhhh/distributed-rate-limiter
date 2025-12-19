# Distributed Rate Limiter as a Service (Go)

A production-style rate limiting service written in Go, designed to demonstrate
backend, concurrency, and distributed systems design.

This project is built incrementally to mirror how real infrastructure services
evolve from single-node correctness to distributed coordination.

---

## Why this project

Rate limiting is a core building block of API gateways and cloud infrastructure
(Cloudflare, Kong, Envoy). This project focuses on:

- Clean abstractions
- Correctness under concurrency
- Thoughtful system design tradeoffs
- Gradual evolution to distributed systems

---

## Current Status

**Phase 1 – Single-node, in-memory rate limiter**

- Fixed Window algorithm implemented
- Thread-safe via mutex
- Exposed via HTTP API
- Designed for future Redis-backed distribution

---

## API

### `POST /check`

**Request**
```json
{ "key": "user123" }

```

**Response**
```json
{ "allowed": true }
```

**Running locally**
```bash
go mod init distributed-rate-limiter  
go run cmd/server/main.go
```

Then:
```bash
curl -X POST localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{"key":"user123"}'
```
Roadmap

Phase 1: Fixed Window limiter (in-memory) ✅

Phase 2: Redis-backed distributed rate limiting

Phase 3: PostgreSQL configuration, multi-tenant limits

Phase 4: Metrics, observability, and production hardening


---
