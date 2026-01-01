# Rate Limiter Algorithms

This directory contains production-grade rate limiting implementations
that share a common interface but differ in enforcement semantics.

---

## Sliding Window Counter (Redis-backed)

### Purpose

The Sliding Window Counter reduces burstiness at fixed-window boundaries
while maintaining bounded memory and high throughput.

It approximates a true sliding window by combining two adjacent fixed
windows with time-based weighting.

---

### Time Source

- Redis server time (`TIME`) is the authoritative clock.
- This avoids clock skew across service instances.
- All window math is performed inside the Lua script.

---

### Window Math

Let:

- `W` = window size in milliseconds
- `now` = current time in milliseconds (from Redis TIME)

Then:

currentWindowStart = floor(now / W) * W
previousWindowStart = currentWindowStart - W
elapsed = now - currentWindowStart
weight = (W - elapsed) / W


The effective request count for the last sliding window is: effective = currentCount + previousCount * weight


---

### Redis Key Model

Two keys are used per identity: rl:{namespace}:{sha1(key)}:swc:{windowStartMs}

Where:
- `swc` distinguishes this algorithm from fixed-window keys
- `windowStartMs` is either the current or previous bucket start

---

### TTL Policy

- Each bucket key is assigned a TTL of `2 * W + buffer`
- This ensures the previous bucket remains available for overlap calculation
- Buffer is implementation-defined (e.g. 1 second)

Memory remains bounded to approximately two keys per identity.

---

### Atomic Lua Script Contract

#### Inputs

**KEYS**
1. `currKey` — current window bucket
2. `prevKey` — previous window bucket

**ARGV**
1. `limit` — max allowed requests per window
2. `windowMs` — window size in milliseconds
3. `ttlMs` — TTL for bucket keys

#### Behavior

1. Fetch Redis server time using `TIME`
2. Compute current and previous window boundaries
3. Read `currCount` and `prevCount` (missing keys count as 0)
4. Compute effective count using fixed-point math: currCount * W + prevCount * (W - elapsed) < limit * W
5. If limit exceeded:
   - Return `allowed = false`
   - Do NOT increment counters
6. If allowed:
   - Atomically increment `currKey`
   - Ensure TTL is set on `currKey`
   - Return `allowed = true`

All operations occur in a single Lua script invocation.

---

### Failure Semantics

- Redis or Lua execution errors propagate to the caller
- Callers are expected to fail closed by default
- Missing previous bucket may slightly undercount, which is acceptable

---

### Trade-offs

**Pros**
- Significantly reduces burstiness vs fixed window
- Bounded memory usage
- Single Redis round-trip
- Suitable for high-throughput systems

**Cons**
- Approximate (not exact sliding window)
- Slight over/under-counting possible at boundaries

---

## Token Bucket (Redis-backed) — Phase 3B (DESIGN)

### Purpose

The Token Bucket algorithm provides:
- Explicit burst control
- Smooth steady-state rate limiting
- Continuous-time behavior

This complements Sliding Window Counter by supporting traffic patterns
where short bursts are acceptable while enforcing a long-term rate.

---

### Semantics

For each identity key:

#### Configuration
- `capacity (C)` — maximum tokens in the bucket (burst size)
- `refillRate (R)` — tokens added per second

#### State
- Current token balance
- Last refill timestamp

#### Allow Rule
1. Refill tokens based on elapsed time since last update
2. Clamp tokens to `capacity`
3. If tokens ≥ 1:
   - Allow request
   - Subtract 1 token
4. Else:
   - Deny request

---

### Time Source

- Redis server time (`TIME`) is the authoritative clock
- Application instance clocks are ignored
- Ensures consistency across distributed workers

Testing relies on:
- `miniredis.FastForward()` advancing Redis TIME
- Lua `TIME` reflecting simulated time
- No mocked or injected clocks

---

### Redis Data Model

#### Key
rl:{namespace}:{sha1(key)}:tb

#### Value (Redis Hash)

| Field | Description |
|-----|-------------|
| `t` | token balance (fixed-point integer) |
| `ts` | last refill timestamp (microseconds, Redis TIME) |

---

### Fixed-Point Math

- Tokens are stored as `tokens * SCALE`
- `SCALE = 1_000_000`
- Refill is computed using elapsed microseconds
- Avoids floating point drift
- Aligns naturally with Redis TIME resolution

---

### TTL Policy

Each token bucket key has a TTL to ensure bounded memory usage.

TTL is derived from refill characteristics: ttl ≈ 2 × (capacity / refillRate)

This ensures:
- Active identities persist
- Idle identities expire naturally
- Token state does not accumulate indefinitely

---

### Atomic Lua Enforcement (High-Level)

1. Fetch Redis server time
2. Load existing token balance and timestamp
3. Compute elapsed time
4. Refill tokens proportionally
5. Clamp to capacity
6. Check availability
7. Update state atomically
8. Refresh TTL

All logic executes in a single Lua script.

---

### Failure Semantics

- Redis or Lua execution errors propagate to the caller
- Callers are expected to fail closed
- Clock anomalies (time moving backwards) result in zero refill

---

### Trade-offs vs Sliding Window Counter

**Token Bucket**
- Native burst support
- Continuous refill
- Expressive configuration (burst + rate)
- One Redis key per identity

**Sliding Window Counter**
- Better for strict “N requests per time window”
- More intuitive for human limits
- Approximate smoothing
- Two keys per identity

Both models are intentionally implemented to demonstrate different
rate-limiting strategies used in production systems.

---

### Non-Goals

- Weighted tokens
- Per-request costs
- Metrics or observability
- Cross-cluster global coordination

This phase focuses strictly on minimal, correct, atomic enforcement.









