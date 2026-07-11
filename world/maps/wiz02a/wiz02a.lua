-- [Openworld] wiz02a (Ix, Town of the Wizards) — return gate to wiz01a.
--
-- The gate places itself: the player's position on map entry is exactly
-- where the wiz01a transition drops them. Walk away to arm the gate, walk
-- back onto the arrival point to travel back. Works for any zone pair.

local origin = nil
local armed = false
local done = false
local AWAY = 150 -- units to walk away before the gate arms
local NEAR = 40  -- proximity that triggers the return transition

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
        Nox.Players.Print("[Openworld] Return gate to the forest placed here.")
        return
    end
    local dx = x - origin.x
    local dy = y - origin.y
    local d2 = dx * dx + dy * dy
    if not armed then
        if d2 > AWAY * AWAY then
            armed = true
            Nox.Players.Print("[Openworld] Return gate armed.")
        end
        return
    end
    if d2 < NEAR * NEAR then
        done = true
        Nox.Players.Print("[Openworld] Returning to the forest...")
        Nox.LoadMap("wiz01a")
    end
end
