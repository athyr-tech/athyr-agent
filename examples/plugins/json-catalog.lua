-- json-catalog.lua
-- Plugin destination: saves each LLM response as a JSON file in a directory.
--
-- Config:
--   path — directory to write catalog entries (required)

local fs = require("fs")

function publish(config, data)
    local filename = config.path .. "/" .. os.date("!%Y%m%d-%H%M%S") .. ".json"
    fs.write(filename, data)
    log("info", "json-catalog: wrote " .. filename)
end
