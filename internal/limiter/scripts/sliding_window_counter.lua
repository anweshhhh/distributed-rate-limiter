-- Sliding Window Counter (Redis-backed)
-- Authoritative time source: Redis TIME
-- Approximate sliding window using two fixed buckets

-- KEYS[1] = current window key
-- KEYS[2] = previous window key
--
-- ARGV[1] = limit
-- ARGV[2] = window size in milliseconds (W)
-- ARGV[3] = ttl in milliseconds

local limit   = tonumber(ARGV[1])
local window  = tonumber(ARGV[2])
local ttl     = tonumber(ARGV[3])

-- Fetch Redis server time
local time = redis.call("TIME")
local nowMs = (time[1] * 1000) + math.floor(time[2] / 1000)

-- Compute window boundaries
local currentWindowStart = math.floor(nowMs / window) * window
local elapsed = nowMs - currentWindowStart
local weightNumerator = window - elapsed

-- Read counters (missing keys count as 0)
local currentCount = tonumber(redis.call("GET", KEYS[1]) or "0")
local previousCount = tonumber(redis.call("GET", KEYS[2]) or "0")

-- Fixed-point comparison:
-- currentCount * W + previousCount * (W - elapsed) < limit * W
local left = (currentCount * window) + (previousCount * weightNumerator)
local right = limit * window

if left >= right then
    return 0
end

-- Allow request: increment current window
redis.call("INCR", KEYS[1])
redis.call("PEXPIRE", KEYS[1], ttl)

return 1
