-- [Openworld] wiz01a (Forest's Edge) — presence marker.
-- The forward transition to wiz02a is the original one; this script only
-- proves per-map Lua loading and announces the openworld layer.

local announced = false

function OnFrame()
    if announced then
        return
    end
    local p = Nox.Players.host
    if p == nil then
        return
    end
    announced = true
    Nox.Players.Print("[Openworld] World layer active. The road to Ix runs both ways now.")
end
