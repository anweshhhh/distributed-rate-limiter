-- Token Bucket (Redis-backed) enforcement
--
-- Uses Redis server time (TIME) as authoritative clock to avoid instance clock skew.
-- Stores state in a Redis hash:
--   t  = token balance (fixed-point, tokens * SCALE)
--   ts = last refill timestamp (microseconds)
--
-- Atomic semantics:
-- 1) Refill tokens based on elapsed time since ts
-- 2) Clamp to capacity
-- 3) If tokens >= cost, decrement and allow; else deny
-- 4) Persist state and refresh TTL
--
-- KEYS[1]
--   bucketKey
--
-- ARGV
--   1) capacityTokens     (int, e.g. 10)
--   2) refillRatePerSec   (int, e.g. 5) tokens per second
--   3) costTokens         (int, e.g. 1)
--   4) scale              (int, fixed-point scale, e.g. 1000000)
--   5) ttlMs              (int, key TTL in milliseconds)

local key = KEYS[1]

local capacity = tonumber(ARGV[1])
local refillRate = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local scale = tonumber(ARGV[4])
local ttlMs = tonumber(ARGV[5])

-- Guard against invalid config; deny safely.
if capacity == nil or refillRate == nil or cost == nil or scale == nil or ttlMs == nil then
  return {0, 0, 0}
end
if capacity <= 0 or refillRate < 0 or cost <= 0 or scale <= 0 or ttlMs <= 0 then
  return {0, 0, 0}
end

-- Redis TIME => {seconds, microseconds}
local now = redis.call("TIME")
local nowMicros = (now[1] * 1000000) + now[2]

local capacityScaled = capacity * scale
local costScaled = cost * scale

local tScaled = redis.call("HGET", key, "t")
local ts = redis.call("HGET", key, "ts")

-- Initialize missing state: full bucket at current time.
if tScaled == false or ts == false then
  tScaled = capacityScaled
  ts = nowMicros
else
  tScaled = tonumber(tScaled)
  ts = tonumber(ts)

  local deltaMicros = nowMicros - ts
  if deltaMicros < 0 then
    deltaMicros = 0
  end

  -- Refill tokens using fixed-point math:
  -- tokensAddedScaled = (refillRate * scale) * (deltaMicros / 1_000_000)
  -- Use floor to avoid "creating" tokens due to fractional microseconds.
  local refillScaled = math.floor(((refillRate * scale) * deltaMicros) / 1000000)

  tScaled = tScaled + refillScaled
  if tScaled > capacityScaled then
    tScaled = capacityScaled
  end

  ts = nowMicros
end

local allowed = 0
if tScaled >= costScaled then
  allowed = 1
  tScaled = tScaled - costScaled
end

redis.call("HSET", key, "t", tScaled, "ts", ts)
redis.call("PEXPIRE", key, ttlMs)

return {allowed, tScaled, ts}
