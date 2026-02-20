-- file-watcher.lua
-- Plugin source: polls a directory for new files and emits their contents.
--
-- Config:
--   path     — directory to watch (required)
--   interval — poll interval in seconds (default: 5)

local fs = require("fs")

function subscribe(config, callback)
    local seen = {}
    local interval = config.interval or 5

    -- Build initial set of known files
    local entries = fs.list(config.path)
    for i = 1, #entries do
        seen[entries[i]] = true
    end

    while true do
        sleep(interval)
        local current = fs.list(config.path)
        for i = 1, #current do
            local name = current[i]
            if not seen[name] then
                seen[name] = true
                local content = fs.read(config.path .. "/" .. name)
                callback(content)
                log("info", "file-watcher: new file detected: " .. name)
            end
        end
    end
end
