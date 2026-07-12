-- [Openworld] ow_wiz02a — Galava, city of the wizards. Wizard starting zone.
-- Campaign scripts are stripped from this map; this file owns the zone.
--
-- Travel gate: the player start doubles as the forest road. Walk away to
-- arm it, step back onto it to travel to the forest (ow_wiz01a).

-- The engine only injects the Nox global when a map has NO Lua file
-- (see opennox-libs script/lua/vm.go ExecFile); scripts must require it.
local Nox = require("Nox.Map.Script.v0")

local origin = nil
local armed = false
local done = false
local AWAY = 150 -- units to walk away before the gate arms
local NEAR = 40  -- proximity that triggers the transition

function OnFrame()
    if done then
        return
    end
    local p = Nox.Players.host
    if p == nil then
        return
    end
    local x, y = p:Pos()
    if x == 0 and y == 0 then
        return
    end
    if origin == nil then
        origin = { x = x, y = y }
        p:Print("[Openworld] Galava. The road to the forest is where you stand.")
        p:Print("[Openworld] Walk away, then return to this spot to travel.")
        return
    end
    local dx = x - origin.x
    local dy = y - origin.y
    local d2 = dx * dx + dy * dy
    if not armed then
        if d2 > AWAY * AWAY then
            armed = true
            p:Print("[Openworld] The forest road is open. Return to your arrival spot to travel.")
        end
        return
    end
    if d2 < NEAR * NEAR then
        done = true
        p:Print("[Openworld] Taking the road to the forest...")
        Nox.LoadMap("ow_wiz01a")
    end
end
