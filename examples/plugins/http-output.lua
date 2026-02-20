-- http-output.lua
-- Plugin destination: posts agent responses to a webhook URL.
--
-- Config:
--   url          — webhook endpoint (required)
--   content_type — Content-Type header (default: "application/json")

local http = require("http")
local json = require("json")

function publish(config, data)
    local payload = json.encode({
        content = data,
    })

    local headers = {
        ["Content-Type"] = config.content_type or "application/json",
    }

    local resp = http.post(config.url, payload, headers)
    if resp.status >= 400 then
        log("error", "http-output: webhook returned status " .. resp.status)
    else
        log("info", "http-output: posted to " .. config.url)
    end
end
