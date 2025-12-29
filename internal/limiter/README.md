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

sql
Copy code

The effective request count for the last sliding window is:

effective = currentCount + previousCount * weight

yaml
Copy code

---

### Redis Key Model

Two keys are used per identity:

rl:{namespace}:{sha1(key)}:swc:{windowStartMs}

markdown
Copy code

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
4. Compute effective count using fixed-point math:
currCount * W + prevCount * (W - elapsed) < limit * W

markdown
Copy code
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