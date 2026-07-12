-- Galava Academy of Wizardry — apprentice enrollment and spell classrooms.
-- This file is APPENDED to the generated ow_wiz02a gate script by
-- tools/owworld. It defines academyFrame(p, x, y), called each frame after
-- the gate logic. All anchors are map waypoints (walkable AI nodes).
--
-- Flow: walk to the reception desk to enrol (get apprentice robes + a room),
-- then visit each classroom station. An instructor lectures on the spell, a
-- spell tome appears (pick it up to learn the spell permanently), and a few
-- crates appear to practise on.

local RECEPTION = { x = 1808, y = 3960 } -- Inn1WP, just inside the town gate

-- Each classroom: station anchor, the spell taught, its display name, and the
-- instructor's short lecture.
local CLASSES = {
    {
        x = 3047, y = 3392, spell = "SPELL_MAGIC_MISSILE", name = "Magic Missile",
        lines = {
            "Instructor Vale: Welcome, apprentice. Every wizard begins here.",
            "Magic Missile is a bolt of pure force — it flies true and never tires.",
            "Take the tome. Then loose it upon the crates yonder!",
        },
    },
    {
        x = 3599, y = 3082, spell = "SPELL_LIGHTNING", name = "Lightning",
        lines = {
            "Instructor Sable: Ah — you have some skill already. Good.",
            "Lightning leaps from your hand to sear those who'd do you harm.",
            "The tome is yours. Practise on the crates — mind the scorch marks!",
        },
    },
    {
        x = 2185, y = 1954, spell = "SPELL_CHANNEL_LIFE", name = "Channel Life",
        lines = {
            "Matron Ilsa: A wizard who cannot mend themselves does not last long.",
            "Channel Life draws vitality from your foe into your own veins.",
            "Learn it well. Test it on the crates, then go forth restored.",
        },
    },
}

local GAP = 3.5 -- seconds between spoken lines

local greeted = false
local enrolled = false
local nudged = false
local classDone = {} -- index -> true once its lecture has played

local function near(x, y, tx, ty, r)
    local dx, dy = x - tx, y - ty
    return dx * dx + dy * dy < r * r
end

-- speak prints lines in sequence and returns the total duration.
local function speak(p, lines)
    for i, line in ipairs(lines) do
        Nox.SecondTimer((i - 1) * GAP, function()
            p:Print(line)
        end)
    end
    return #lines * GAP
end

local function spawnProps(cx, cy)
    local t = Nox.ObjectType("Crate")
    if not t then t = Nox.ObjectType("Pot") end
    if not t then t = Nox.ObjectType("Barrel") end
    if not t then return end
    t:Create(cx + 45, cy)
    t:Create(cx - 45, cy)
    t:Create(cx, cy + 55)
end

local function giveRobe()
    local t = Nox.ObjectType("WizardRobe")
    if t then
        t:Create(RECEPTION.x + 30, RECEPTION.y)
    end
end

function academyFrame(p, x, y)
    -- Reception: enrol the apprentice.
    if not enrolled then
        if near(x, y, RECEPTION.x, RECEPTION.y, 150) then
            if not greeted then
                greeted = true
                enrolled = true
                speak(p, {
                    "Receptionist Mabel: Welcome to the Academy of Galava, apprentice!",
                    "You are enrolled. Your quarters and robes are here — take them.",
                    "The classrooms lie north: the well, the tower, and the inner sanctum.",
                })
                Nox.SecondTimer(2.0, giveRobe)
            end
        end
        return
    end

    -- Classrooms: lecture, hand over the tome, set out practice crates.
    for i, c in ipairs(CLASSES) do
        if not classDone[i] and near(x, y, c.x, c.y, 140) then
            classDone[i] = true
            local dur = speak(p, c.lines)
            Nox.SecondTimer(dur, function()
                if Nox.GiveSpellBook(c.x, c.y - 25, c.spell) then
                    p:Print("[Academy] A spell tome appears — pick it up to learn " .. c.name .. ".")
                    spawnProps(c.x, c.y + 80)
                else
                    p:Print("[Academy] (The tome shimmers and fades — this class is not ready.)")
                end
            end)
        end
    end
end
