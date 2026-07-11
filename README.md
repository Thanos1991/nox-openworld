# Nox Openworld

An open-world expansion project for **Nox** (Westwood, 2000), built on
[OpenNox](https://github.com/opennox/opennox) and playable on the
[Android port](https://github.com/Thanos1991/nox-android).

## Vision

Connect and adapt existing locations from **all three class campaigns** (Warrior,
Conjurer, Wizard) into one shared, freely traversable world:

- Maps remain separate zones, connected through **bidirectional transitions**
  (the original campaigns are one-way chains).
- Any class can eventually travel to regions that were inaccessible in their
  original campaign.
- Reuse existing maps and assets; **replace or adapt** linear campaign scripts,
  class restrictions, quest assumptions, and one-way transitions.
- Hub-based structure: a home settlement with quest-givers, vendors and storage,
  radiating out into the reconnected campaign regions.

## What's here

- `tools/mapatlas/` — Go tool that dumps every original map into planning artifacts
  (requires a sibling checkout of [opennox-libs](https://github.com/Thanos1991/opennox-libs)
  and your own Nox game data).
- `data/maps.json` — structured dump of all maps: metadata, object type inventory,
  script strings, cross-map references.
- `docs/atlas.md` — human-readable world atlas, grouped by campaign, with rendered
  map images.
- `docs/world-graph.md` — mermaid graph of cross-map references (the original
  transition topology to be rewired).
- `renders/` — top-down renders of every map, drawn from real game data.

## Regenerating the dumps

```
git clone https://github.com/Thanos1991/opennox-libs ../opennox-libs
go build ./tools/mapatlas
./mapatlas -maps "C:\GOG Games\Nox\maps" -data "C:\GOG Games\Nox" -out . -render
```

Nox game data is required and never committed to this repo.

## Engine facts that matter (for planning)

- One map ("zone") is loaded at a time; transitions are load-based. Loads take
  ~1 s on target hardware, so a zone web plays fluidly.
- Maps carry their own scripts. OpenNox supports the original compiled NoxScript
  plus modern NoxScript v4 (Go) and Lua per-map scripts — that's where the quest
  logic replacement happens.
- Cross-map persistent world state has no first-class engine support yet; it will
  need a small engine-side store (tracked as an open task).
- Class restrictions and quest gating live in map scripts and object placements,
  not in the map geometry — which is why script replacement unlocks cross-class
  travel without touching the beautiful parts.
