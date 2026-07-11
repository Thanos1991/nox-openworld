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

## Current state / next steps

- [x] Reconnaissance dumps (this repo: data/, docs/, renders/)
- [ ] Decode ObjectData records for exit objects (destination map + waypoint)
- [ ] Hello-world Lua/NS4 script on a campaign map (verify load on device)
- [ ] Two-zone proof: bidirectional transition between two campaign maps
- [ ] Engine-side persistent world-flag store
- [ ] Hub town selection and quest framework design
