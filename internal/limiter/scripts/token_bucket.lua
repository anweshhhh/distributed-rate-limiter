-- Token Bucket (Redis-backed)

local key = KEYS[1]

local capacity   = tonumber(ARGV[1])
local refillRate = tonumber(ARGV[2])
local cost       = tonumber(ARGV[3])
local scale      = tonumber(ARGV[4])
local ttlMs      = tonumber(ARGV[5])
local nowArg     = ARGV[6] -- optional override (microseconds)

if capacity <= 0 or refillRate <= 0 or cost <= 0 then
  return {0, 0, 0}
end

-- Determine current time (microseconds)
local nowMicros
if nowArg ~= nil then
  nowMicros = tonumber(nowArg)
else
  local t = redis.call("TIME")
  nowMicros = (t[1] * 1000000) + t[2]
end

local capacityScaled = capacity * scale
local costScaled     = cost * scale

local tScaled = redis.call("HGET", key, "t")
local ts      = redis.call("HGET", key, "ts")

if tScaled == false or ts == false then
  tScaled = capacityScaled
  ts = nowMicros
else
  tScaled = tonumber(tScaled)
  ts = tonumber(ts)

  local deltaMicros = nowMicros - ts
  if deltaMicros > 0 then
    local refillTokens = math.floor((deltaMicros * refillRate) / 1000000)
    if refillTokens > 0 then
      tScaled = math.min(capacityScaled, tScaled + refillTokens * scale)
      ts = nowMicros
    end
  end
end

local allowed = 0
if tScaled >= costScaled then
  allowed = 1
  tScaled = tScaled - costScaled
end

redis.call("HSET", key, "t", tScaled, "ts", ts)
redis.call("PEXPIRE", key, ttlMs)

return {allowed, tScaled, ts}
