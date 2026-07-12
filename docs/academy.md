# The Galava Academy of Wizardry

An in-world apprentice system for the open world: the wizard player enrols at
the academy in Galava (ow_wiz02a) and attends classes to learn the basic
spells the campaign would normally hand out. This is the first piece of
open-world *content* on top of the traversal layer.

## How it works in-game

1. **Enrolment.** Walk to the reception desk just inside Galava's town gate
   (waypoint Inn1WP). Receptionist Mabel enrols you, grants apprentice robes,
   and points you to the classrooms.
2. **Classrooms.** Three teaching stations around the town:
   | Station | Waypoint | Instructor | Spell taught |
   |---|---|---|---|
   | The well | WellWP | Vale | Magic Missile |
   | The tower | ExitTowerWP | Sable | Lightning |
   | Inner sanctum | CryptChairStartWP | Ilsa | Channel Life (heal) |
   Approach a station: the instructor lectures (a few lines), a **spell tome**
   appears, and a few **crates** are set out to practise on. Pick up the tome
   to learn the spell **permanently** (it stays on your character).

## How it works technically

- **Friendliness.** Handled by the faction layer (see below): in Galava all
  wizards are on your team, so instructors and townsfolk are friendly.
- **Teaching.** `Nox.GiveSpellBook(x, y, "SPELL_MAGIC_MISSILE")` spawns the
  engine's own spell-reward book. On pickup the engine runs
  `spellGrantToPlayer`, permanently teaching the spell to a wizard/conjurer
  player (warriors are refused by the engine). Capability:
  `script.SpellTeacher` (opennox-libs) implemented in the fork
  (`src/script_teach.go`).
- **Dialogue & triggers.** Proximity triggers (distance checks in the frame
  loop, like the travel gates) fire the lectures; lines are spaced with
  `Nox.SecondTimer`. Practice props via `Nox.ObjectType("Crate"):Create`.
- **Where the code lives.** Hand-authored content is
  `world/academy/wiz02a.lua`, which defines `academyFrame(p, x, y)`.
  `tools/owworld` appends it to the generated `ow_wiz02a.lua`, whose gate loop
  calls `academyFrame` each frame. Any zone can get authored content by adding
  `world/academy/<map>.lua`.

## Faction model (why your own kind is friendly)

The engine treats two units as enemies unless they share a team. Each zone
script, on entry, puts the host player on their **class** team and every placed
NPC on the **zone's campaign** team (`Nox.SetHostTeam` / `Nox.SetMapUnitsTeam`,
capability `script.Factions`). So a wizard in a wizard zone is among allies; a
wizard in the warrior castle finds everyone hostile. Known limitation (accepted
for now): genuine monsters inside a same-faction zone also become friendly.

## Extending the academy

- **Use the real castle levels.** The warrior-assault maps of Galava
  (`war07a`–`war07h`) are the castle interiors a wizard never normally sees —
  ideal dedicated classrooms/dormitories. To use them: add
  `world/academy/war07x.lua` content, and either re-tag those clones to the
  wizard faction for wizard visitors or gate them behind enrolment. They are
  already cloned and connected in the warrior region.
- **More spells / tiers.** Add rows to `CLASSES` in the academy file (Burn,
  Shock, Fireball, Force of Nature, Death Ray are all wizard spells). Gate
  advanced classes behind having learned the basics.
- **Persistent enrolment & a personal room.** Needs the planned cross-map
  world-flag store (AGENTS.md task) so enrolment and room state survive zone
  changes; today enrolment is per-visit and spells persist on the character.
- **Conjurer & warrior academies.** Mirror the pattern in their hub zones with
  their own trainers (conjurers learn summoning; warriors get ability books via
  a analogous `AbilityBook` reward — a sibling capability to add later).
