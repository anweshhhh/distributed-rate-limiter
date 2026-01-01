-- Sliding Window Counter (Redis-backed)
-- Authoritative time source: Redis TIME
-- Approximate sliding window using two fixed buckets

-- KEYS[1] = base key prefix (e.g. rl:{namespace}:{hash}:swc)
--
-- ARGV[1] = limit
-- ARGV[2] = window size in milliseconds (W)
-- ARGV[3] = ttl in milliseconds

local base   = KEYS[1]
local limit  = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local ttl    = tonumber(ARGV[3])

-- Fetch Redis server time
local time = redis.call("TIME")
local nowMs = (time[1] * 1000) + math.floor(time[2] / 1000)

-- Compute window boundaries
local currStart = math.floor(nowMs / window) * window
local prevStart = currStart - window
local elapsed   = nowMs - currStart
local weightNum = window - elapsed

local currKey = base .. ":" .. currStart
local prevKey = base .. ":" .. prevStart

-- Read counters (missing keys count as 0)
local currCount = tonumber(redis.call("GET", currKey) or "0")
local prevCount = tonumber(redis.call("GET", prevKey) or "0")

-- Fixed-point comparison:
-- currCount * W + prevCount * (W - elapsed) < limit * W
local left  = (currCount * window) + (prevCount * weightNum)
local right = limit * window

if left >= right then
  return 0
end

-- Allow request
redis.call("INCR", currKey)
redis.call("PEXPIRE", currKey, ttl)
redis.call("PEXPIRE", prevKey, ttl)

return 1
