-- [Openworld] ow_wiz01a — the forest outside Galava.
-- Campaign scripts are stripped from this map; this file owns the zone.
--
-- Travel gate: fixed at the map's own FromGalavaWP waypoint (1246,1413) —
-- the spot where arrivals from Galava land in the original campaign. The
-- gate arms only after the player has been away from it once, so spawning
-- on top of it can't bounce you straight back.

-- The engine only injects the Nox global when a map has NO Lua file
-- (see opennox-libs script/lua/vm.go ExecFile); scripts must require it.
local Nox = require("Nox.Map.Script.v0")

local GATE = { x = 1246, y = 1413 } -- Wiz01A.map:FromGalavaWP
local AWAY = 120
local NEAR = 50

local announced = false
local armed = false
local done = false

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
    if not announced then
        announced = true
        p:Print("[Openworld] The forest. The road back to Galava lies to the north-west.")
    end
    local dx = x - GATE.x
    local dy = y - GATE.y
    local d2 = dx * dx + dy * dy
    if not armed then
        if d2 > AWAY * AWAY then
            armed = true
        end
        return
    end
    if d2 < NEAR * NEAR then
        done = true
        p:Print("[Openworld] Taking the road to Galava...")
        Nox.LoadMap("ow_wiz02a")
    end
end
