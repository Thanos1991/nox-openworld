# Orientation for AI collaborators

This file gives any AI assistant (or human) the context needed to continue this
project without re-deriving it.

## The project

Open-world expansion for Nox (2000) on the OpenNox engine: merge the three class
campaigns' maps into one freely traversable, hub-based world. Zones stay separate
maps; transitions become bidirectional; campaign scripts/class gates get replaced.
See README.md for the full vision. The owner plays on Android via
github.com/Thanos1991/nox-android (see its README for the porting story).

## Sibling repositories (clone side by side, exact dir names)

```
dev/
├── nox-openworld/    this repo
├── opennox/          engine fork, branch android-port  (github.com/Thanos1991/opennox)
├── opennox-libs/     libs fork, branch android-port    (github.com/Thanos1991/opennox-libs)
├── go-sdl2/          github.com/Thanos1991/go-sdl2-android
└── nox-android/      Android app + build scripts       (github.com/Thanos1991/nox-android)
```

Game data (not in git): owner's GOG install at `C:\GOG Games\Nox`.

## Key engine knowledge

- `opennox-libs/maps` parses and **writes** every map section (Reader/Writer) —
  map surgery can be done in Go without the original editor.
- `opennox-libs/maps/maprender` renders maps to images using real game data.
- Map scripts: original compiled NoxScript lives in the `ScriptObject` section;
  OpenNox also loads per-map Go (NoxScript v4, `github.com/opennox/noxscript/ns/v4`)
  and Lua scripts from the map directory — this is the intended path for new quest
  logic (engine loads them at map start; see `[script]` log lines).
- Object placements: `ObjectTOC` (type table) + `ObjectData` (raw records, format
  in `opennox/src` xfer code). Exits/teleporters are objects; the graph edges in
  docs/world-graph.md come from string references in script+object data.
- Transitions: campaign progression uses script-driven map switches; multiplayer
  maps use exit objects. Both end up loading a new map — rewiring means editing
  scripts and/or exit object destinations.
- No cross-map persistent world state exists yet. Plan: small engine-side flag
  store saved with the game (the engine fork is ours; adding Go there is routine —
  see `gameexOnKeyboardPress` in `opennox/src/gameex.go` for how the Android port
  hooks the engine).

## Working agreements

- The owner tests on a OnePlus Open over adb; engine logs land in
  `Download/Nox/opennox.log` on-device.
- Anything data-driven (maps, scripts) lands on the phone automatically via the
  app's data mirror after copying to `Download/Nox`.
- Don't commit game data or derived assets containing original art beyond the
  small map renders used for planning.

## Architecture: the open world is a separate game mode

The open world never touches the original campaign. Instead:

- `tools/owgen` clones campaign maps into an `ow_` namespace (`wiz02a` →
  `maps/ow_wiz02a/ow_wiz02a.map`), stripping the compiled NoxScript sections
  (`ScriptObject`, `ScriptData`) — story logic, class gates and one-way
  transitions gone; everything else byte-identical. Generated maps contain
  original game data and are NOT committed; `deploy-world.ps1` regenerates
  them from the local install and pushes maps+scripts to PC and phone.
- Each `ow_` map dir gets a Lua script (this repo, `world/maps/ow_*/`) that
  owns the zone: transitions via `Nox.LoadMap`, later quests/flags. Scripts
  MUST `local Nox = require("Nox.Map.Script.v0")` — the engine only injects
  the global for maps without a Lua file.
- The engine fork adds an **Open World** main-menu button (`gui_main_menu.go`,
  injected via `newWindowFromString` — no game-data edits, button ID 141).
  It arms `nox_openworld_newgame` (legacy/GAME1.c), which remaps the class
  start maps inside `nox_xxx_gameSetMapPath_409D70`: wizard → `ow_wiz02a`
  (Galava), warrior → `ow_war01a`, conjurer → `ow_con01a`. The flag resets
  every time the main menu shows.
- Class-select and character creation are the stock flows; Open World always
  goes straight to class creation (no save-slot list).

Gate placement facts: `data/ow_waypoints.json` has named waypoints per zone.
wiz01a's `FromGalavaWP` (1246,1413) is the canonical road-to-Galava spot.
In ow_wiz02a the wizard PlayerStart doubles as the forest gate (self-placing
script pattern: record spawn, arm at 150 units away, trigger within 40).

## Current state / next steps

- [x] Reconnaissance dumps (this repo: data/, docs/, renders/)
- [x] Object inventories per map (required fixing RawSection aliasing in opennox-libs
      maps/reader.go — our fork has the fix; consider upstreaming)
- [x] Transition mechanism identified: `InvisibleExitArea` trigger objects (100 maps)
      + script-driven map-switch calls; destinations appear as strings in ScriptData
      (see map_refs in data/maps.json)
- [x] Per-map Lua verified loading on device (first attempt crashed: missing
      `require` — see world scripts' header comment)
- [x] `Nox.LoadMap` engine capability (script.MapSwitcher in both forks)
- [x] Open World as separate mode: menu button, start-map remap, ow_ map set
      (owgen), wizard start in Galava (ow_wiz02a), gates ow_wiz02a <-> ow_wiz01a
- [x] Nox.LoadMap deadlock fixed (never QueueInLoop from script code — Lua runs
      on the server loop goroutine that drains the channel; Server.SwitchMap
      only flags the switch and is safe to call directly)
- [x] Map transitions crashed the client in getCursorAnimFrame ("not an
      animation" panic mid-reload) — hardened in the fork
- [x] Arrival placement via `Nox.LoadMap("map:WaypointName")` (engine-native
      syntax, Server.SwitchMap) — Galava→forest lands at FromGalavaWP;
      forest→Galava lands at PlayerStart which IS the Galava gate
- [ ] On-device playtest of the Open World wizard start + both gate directions
- [ ] Strip/neutralize `InvisibleExitArea` trigger objects in ow_ maps if they
      turn out to misbehave without their scripts
- [ ] Engine-side persistent world-flag store
- [ ] Hub town confirmation (Galava is the wizard candidate; shopkeeper-rich maps
      are flagged in notable_objects)
- [ ] Scripts + gates for warrior (ow_war01a) and conjurer (ow_con01a) starts
- [ ] maprender fails on 85 of 157 maps ("invalid image size 0x0" etc.) — improve
      renderer coverage for complete atlas imagery
