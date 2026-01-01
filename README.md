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

### ✅ Phase 1 — In-memory Fixed Window

- Fixed Window algorithm implemented
- Thread-safe via mutex
- Exposed via HTTP API
- Designed for future Redis-backed distribution

### ✅ Phase 2 — Redis-backed Fixed Window (Distributed)
- Redis + Lua for **atomic INCR + EXPIRE**
- Single Redis round-trip per request
- Deterministic window calculation
- Safe Redis key design with hashing and namespacing
- Fail-closed behavior on Redis errors
- Deterministic time injection for tests
- Fully tested using `miniredis`

---
## 🚀 Phase 3 — Advanced Rate Limiting Algorithms

### ✅ Phase 3A — Sliding Window Counter (Redis-backed)

**Goal:**  
Reduce burstiness at fixed-window boundaries while maintaining bounded memory and high throughput.

**Design highlights:**
- Redis-backed **Sliding Window Counter** using two adjacent buckets
- Approximate sliding window via weighted overlap of previous window
- Atomic enforcement using Lua (single round-trip)
- **Redis server time (`TIME`)** used as the authoritative clock to avoid instance skew
- Fixed-point math inside Lua to avoid floating-point precision issues
- Memory bounded to ~2 keys per rate-limited identity

**Key trade-off:**
This approach is an approximation (unlike exact timestamp logs),
but offers a strong balance between correctness, performance, and operational simplicity.


### ✅ Phase 3B — Token Bucket (Redis-backed)

**Goal:** 
Support controlled burst traffic while enforcing a steady long-term rate.

**Design highlights:**
- Redis-backed Token Bucket with Lua-based atomic enforcement
- Per-identity bucket with configurable:
capacity (burst size)
refill rate (tokens per second)
- Fixed-point math for deterministic behavior
- Single Redis hash per identity
- TTL-based cleanup to ensure bounded memory usage
- Deterministic, production-safe testing strategy

**Why Token Bucket:**
Unlike window-based approaches, token buckets allow short bursts
without permanently penalizing clients, making them ideal for
user-facing APIs and gateways.
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

✅ Phase 1: In-memory Fixed Window
✅ Phase 2: Redis-backed Fixed Window
✅ Phase 3A: Sliding Window Counter (Redis + Lua)
✅ Phase 3B: Token Bucket (Redis + Lua)

---
