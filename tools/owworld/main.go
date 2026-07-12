// owworld generates the open world from extracted gate data:
//
//   - world/maps/ow_<map>/ow_<map>.lua — per-zone gate scripts (travel via
//     Nox.LoadMap with "@x,y" arrival placement)
//   - docs/openworld-map.md — the world connection map (mermaid + tables)
//
// It reads data/ow_gates.json (from tools/owgates). Every connection is made
// bidirectional: where the original campaign had no return gate, a virtual
// gate is placed at the zone's PlayerStart (offset if several stack there).
//
// Usage: owworld [-gates data\ow_gates.json] [-out .]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Pos struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Gate struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Dest   string  `json:"dest"`
	DestWP string  `json:"dest_wp"`
	N      int     `json:"n"`
}

type MapGates struct {
	Starts []Pos  `json:"starts"`
	Gates  []Gate `json:"gates"`
	Error  string `json:"error"`
}

// The zones that make up the openworld. Connections to maps outside this set
// are skipped (logged as frontier — future expansion).
var include = []string{
	// wizard campaign region
	"wiz01a", "wiz02a", "wiz02b", "wiz02c", "wiz03a", "wiz03b", "wiz03c",
	"wiz04a", "wiz04b", "wiz04c", "wiz05a", "wiz05b", "wiz05c",
	"wiz06a", "wiz06b", "wiz06c",
	"wiz07a", "wiz07b", "wiz07c", "wiz07d", "wiz07e", "wiz07f",
	"wiz08a", "wiz08b", "wiz08c", "wiz08d", "wiz08e",
	"wiz09a", "wiz09b", "wiz09c", "wiz09d",
	"wiz10a", "wiz10b", "wiz10c", "wiz10d", "wiz11a",
	// warrior campaign region
	"war01a", "war02a", "war02b", "war03a", "war03b", "war03c", "war03d",
	"war04a", "war04b", "war04c", "war05a", "war05b", "war05c",
	"war06a", "war06b",
	"war07a", "war07b", "war07c", "war07d", "war07e", "war07f", "war07g", "war07h",
	"war08a", "war08b", "war08c", "war08d", "war08e",
	"war09a", "war09b", "war09c", "war09d",
	"war10a", "war10b", "war10c", "war10d", "war11a",
	// conjurer campaign region
	"con01a", "con02a", "con03a", "con03b",
	"con04a", "con04b", "con04c", "con05a", "con05b", "con05c",
	"con06a", "con06b",
	"con07a", "con07b", "con07c", "con07d", "con07e", "con07f", "con07g", "con07h",
	"con08a", "con08b", "con08c", "con08d", "con08e",
	"con09a", "con09b", "con09c", "con09d",
	"con10a", "con10b", "con10c", "con10d", "con11a",
}

// Overrides for exit triggers whose destination is script-driven (empty in
// object data). Keyed by "map@x,y" using the extracted (rounded) position.
var destOverride = map[string]string{
	"wiz03b@4652,1492": "wiz03c", // confirmed by map waypoint Wiz03cExitWP
	"wiz07c@2266,3370": "wiz08a", // wiz08a holds the return gate + FromWiz7WP
	"con10b@3151,5290": "con10a", // geometry-identical to wiz10b/war10b's 10a gate
	"war03d@945,4384":  "war03b", // war03d's only campaign reference
	"con03a@651,3421":  "con04a", // campaign flow con03a -> con04a (map_refs)
}

// Gates to drop entirely (unknown script-only destinations).
var dropGate = map[string]bool{
	"wiz03c@4476,1089": true,
}

// Story links that had no exit objects (script-driven transitions in the
// campaign). x/y 0 means "place at the zone's player start" (virtual).
type extraGate struct {
	from, to string
	x, y     float32
}

var extraGates = []extraGate{
	{from: "wiz07b", to: "wiz07c", x: 2131, y: 4931}, // waypoint Wiz08b.map:FromC
	{from: "wiz07c", to: "wiz07b", x: 4497, y: 4152}, // waypoint Wiz07C.map:FromWiz7WP
	{from: "wiz07d", to: "wiz07e"},                   // story flow, no anchor: player start
	{from: "wiz07e", to: "wiz07d"},
}

// Flavor names for travel messages; zones not listed use their map id.
var names = map[string]string{
	"wiz01a": "the forest",
	"wiz02a": "Galava",
	"wiz02b": "the Lost Library, first floor",
	"wiz02c": "the Lost Library, second floor",
	"wiz09a": "the swamp (west)",
	"wiz09b": "the swamp (east)",
	"wiz09c": "the tunnels",
	"wiz09d": "the wastelands",
	"wiz10a": "the Land of the Dead (I)",
	"wiz10b": "the Land of the Dead (II)",
	"wiz10c": "the Land of the Dead (III)",
	"wiz10d": "the Land of the Dead (IV)",
	"wiz11a": "Hecubah's lair",
}

type conn struct {
	from, to string
	pos      Pos    // gate position in `from`
	arriveWP string // canonical arrival waypoint in `to`, when the trigger carried one
	virtual  bool   // no original gate; placed at PlayerStart
}

type Waypoint struct {
	ID   uint32  `json:"id"`
	Name string  `json:"name"`
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
}

func main() {
	gatesPath := flag.String("gates", `data\ow_gates.json`, "gate data from owgates")
	wpPath := flag.String("waypoints", `data\ow_waypoints.json`, "waypoint dump from owgen")
	outDir := flag.String("out", `.`, "repo root")
	flag.Parse()

	raw, err := os.ReadFile(*gatesPath)
	if err != nil {
		panic(err)
	}
	var data map[string]*MapGates
	if err := json.Unmarshal(raw, &data); err != nil {
		panic(err)
	}
	rawWP, err := os.ReadFile(*wpPath)
	if err != nil {
		panic(err)
	}
	var wps map[string][]Waypoint
	if err := json.Unmarshal(rawWP, &wps); err != nil {
		panic(err)
	}
	// Arrival placement. Exit-trigger centers can sit inside door frames or
	// off the walkable edge and SetPos does not collision-check, so raw gate
	// coordinates strand players. Waypoints are walkable AI path nodes, but
	// audio/cinematic markers among them are not reliable ground.
	badWP := func(name string) bool {
		l := strings.ToLower(name)
		return strings.Contains(l, "sound") || strings.Contains(l, "audio") ||
			strings.Contains(l, "origin") || strings.Contains(l, "brief")
	}
	// wpByName resolves the canonical arrival waypoint carried by the
	// original exit trigger ("Map.map:SomeWP"; dump names use that form too).
	wpByName := func(m, name string) (Pos, bool) {
		want := strings.ToLower(name)
		if i := strings.LastIndexByte(want, ':'); i >= 0 {
			want = want[i+1:]
		}
		for _, w := range wps[m] {
			n := strings.ToLower(w.Name)
			if i := strings.LastIndexByte(n, ':'); i >= 0 {
				n = n[i+1:]
			}
			if n != "" && n == want {
				return Pos{X: w.X, Y: w.Y}, true
			}
		}
		return Pos{}, false
	}
	playerStart := func(m string) (Pos, bool) {
		if mg := data[m]; mg != nil && len(mg.Starts) > 0 {
			return mg.Starts[0], true
		}
		return Pos{}, false
	}
	snapWalkable := func(m string, p Pos) Pos {
		// the zone's entrance case: the back gate sits near the player
		// start, which is guaranteed walkable and historically correct
		if st, ok := playerStart(m); ok {
			dx, dy := float64(st.X-p.X), float64(st.Y-p.Y)
			if dx*dx+dy*dy <= 600*600 {
				return st
			}
		}
		best := p
		bestD := float64(400 * 400)
		found := false
		for _, w := range wps[m] {
			if badWP(w.Name) {
				continue
			}
			dx := float64(w.X - p.X)
			dy := float64(w.Y - p.Y)
			d := dx*dx + dy*dy
			if d < bestD {
				bestD = d
				best = Pos{X: w.X, Y: w.Y}
				found = true
			}
		}
		if !found {
			if st, ok := playerStart(m); ok {
				fmt.Printf("no waypoint near %s@%.0f,%.0f — arriving at player start\n", m, p.X, p.Y)
				return st
			}
		}
		return best
	}
	inc := map[string]bool{}
	for _, m := range include {
		inc[m] = true
	}

	// collect real directed gates within the world
	conns := map[string][]*conn{} // by "from"
	var frontier []string
	for m, mg := range data {
		if !inc[m] {
			continue
		}
		for _, g := range mg.Gates {
			key := fmt.Sprintf("%s@%.0f,%.0f", m, g.X, g.Y)
			dest := g.Dest
			if ov, ok := destOverride[key]; ok {
				dest = ov
			}
			if dropGate[key] || dest == "" {
				if dest == "" {
					fmt.Println("drop (no dest):", key)
				}
				continue
			}
			if !inc[dest] {
				frontier = append(frontier, fmt.Sprintf("%s -> %s (outside world)", m, dest))
				continue
			}
			conns[m] = append(conns[m], &conn{from: m, to: dest, pos: Pos{X: g.X, Y: g.Y}, arriveWP: g.DestWP})
		}
	}

	// story links without exit objects
	for _, e := range extraGates {
		if !inc[e.from] || !inc[e.to] {
			continue
		}
		c := &conn{from: e.from, to: e.to, pos: Pos{X: e.x, Y: e.y}}
		if e.x == 0 && e.y == 0 {
			mg := data[e.from]
			if mg == nil || len(mg.Starts) == 0 {
				fmt.Println("extra gate has no anchor:", e.from, "->", e.to)
				continue
			}
			c.pos = mg.Starts[0]
			c.virtual = true
		}
		conns[e.from] = append(conns[e.from], c)
	}

	// merge duplicate gates to the same destination that are far apart:
	// keep them all (multiple doorways), they simply lead to the same place.

	// ensure bidirectionality with virtual gates at PlayerStart
	for m := range inc {
		byDest := map[string]bool{}
		for _, c := range conns[m] {
			byDest[c.to] = true
		}
		for other := range inc {
			if other == m || byDest[other] {
				continue
			}
			// does other lead to m?
			leads := false
			for _, c := range conns[other] {
				if c.to == m {
					leads = true
					break
				}
			}
			if !leads {
				continue
			}
			mg := data[m]
			if mg == nil || len(mg.Starts) == 0 {
				fmt.Println("cannot place virtual gate, no PlayerStart:", m, "->", other)
				continue
			}
			st := mg.Starts[0]
			// offset stacked virtual gates deterministically
			k := 0
			for _, c := range conns[m] {
				if c.virtual {
					k++
				}
			}
			conns[m] = append(conns[m], &conn{
				from: m, to: other, virtual: true,
				pos: Pos{X: st.X + float32(k)*90, Y: st.Y},
			})
		}
	}

	// deterministic order
	for m := range conns {
		sort.Slice(conns[m], func(i, j int) bool { return conns[m][i].to < conns[m][j].to })
	}

	// arrival position for travel a->b: the b-side gate back to a (nearest if
	// several); guaranteed to exist after the virtual pass.
	arrival := func(a, b string) (Pos, bool) {
		var best *conn
		for _, c := range conns[b] {
			if c.to == a {
				if best == nil {
					best = c
				}
			}
		}
		if best == nil {
			return Pos{}, false
		}
		return best.pos, true
	}

	// pair multi-gate connections by rank (sorted by x+y on both sides) so two
	// staircases between the same two maps don't cross-teleport.
	pairArrival := func(c *conn, idx, total int) Pos {
		var backs []*conn
		for _, bc := range conns[c.to] {
			if bc.to == c.from {
				backs = append(backs, bc)
			}
		}
		if len(backs) == 0 {
			return Pos{}
		}
		sort.Slice(backs, func(i, j int) bool {
			return backs[i].pos.X+backs[i].pos.Y < backs[j].pos.X+backs[j].pos.Y
		})
		if total > 1 && len(backs) == total {
			return backs[idx].pos
		}
		return backs[0].pos
	}

	// generate lua per zone
	genCount := 0
	for _, m := range include {
		cs := conns[m]
		var b strings.Builder
		fmt.Fprintf(&b, "-- ow_%s gates — GENERATED by tools/owworld, do not edit by hand.\n", m)
		b.WriteString("-- Regenerate: go run ./tools/owgates <maps>; go run ./tools/owworld\n\n")
		b.WriteString("local Nox = require(\"Nox.Map.Script.v0\")\n\n")
		b.WriteString("local GATES = {\n")
		// index gates by destination for pairing
		byDest := map[string][]*conn{}
		for _, c := range cs {
			byDest[c.to] = append(byDest[c.to], c)
		}
		for _, c := range cs {
			group := byDest[c.to]
			sort.Slice(group, func(i, j int) bool {
				return group[i].pos.X+group[i].pos.Y < group[j].pos.X+group[j].pos.Y
			})
			idx := 0
			for i, gc := range group {
				if gc == c {
					idx = i
					break
				}
			}
			var ap Pos
			if c.arriveWP != "" {
				if p, ok := wpByName(c.to, c.arriveWP); ok {
					ap = p // canonical arrival: the waypoint the original trigger names
				} else {
					fmt.Printf("arrival waypoint %q not found in %s\n", c.arriveWP, c.to)
					ap = snapWalkable(c.to, pairArrival(c, idx, len(group)))
				}
			} else {
				ap = snapWalkable(c.to, pairArrival(c, idx, len(group)))
			}
			nm := names[c.to]
			if nm == "" {
				nm = c.to
			}
			fmt.Fprintf(&b, "    { x = %.0f, y = %.0f, dest = \"ow_%s:@%.0f,%.0f\", name = %q },\n",
				c.pos.X, c.pos.Y, c.to, ap.X, ap.Y, nm)
		}
		b.WriteString("}\n\n")
		b.WriteString(luaEngine)
		dir := filepath.Join(*outDir, "world", "maps", "ow_"+m)
		os.MkdirAll(dir, 0o755)
		if err := os.WriteFile(filepath.Join(dir, "ow_"+m+".lua"), []byte(b.String()), 0o644); err != nil {
			panic(err)
		}
		genCount++
	}

	// connectivity check: every zone must be reachable from the hub
	reach := map[string]bool{"wiz02a": true}
	queue := []string{"wiz02a"}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range conns[cur] {
			if !reach[c.to] {
				reach[c.to] = true
				queue = append(queue, c.to)
			}
		}
	}
	for _, m := range include {
		if !reach[m] {
			fmt.Println("WARNING: zone not reachable from the hub:", m)
		}
	}

	writeWorldDoc(filepath.Join(*outDir, "docs", "openworld-map.md"), conns, data, frontier)
	_ = arrival
	fmt.Printf("generated %d zone scripts\n", genCount)
	for _, f := range frontier {
		fmt.Println("frontier:", f)
	}
}

// the shared gate engine appended to every zone script
const luaEngine = `local NEAR = 50   -- stepping this close to a gate travels
local AWAY = 130  -- a gate arms once the player has been this far from it
local HINT = 250  -- distance at which the road hint prints

local armed = {}
local hinted = {}
local fired = false

function OnFrame()
    if fired then
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
    for i, g in ipairs(GATES) do
        local dx, dy = x - g.x, y - g.y
        local d2 = dx * dx + dy * dy
        if not armed[i] then
            -- gates start disarmed so arriving on one can't bounce you back
            if d2 > AWAY * AWAY then
                armed[i] = true
            end
        else
            if not hinted[i] and d2 < HINT * HINT then
                hinted[i] = true
                p:Print("[Openworld] The road to " .. g.name .. " lies here.")
            end
            if d2 < NEAR * NEAR then
                fired = true
                p:Print("[Openworld] Travelling to " .. g.name .. "...")
                Nox.LoadMap(g.dest)
                return
            end
        end
    end
end
`

func writeWorldDoc(path string, conns map[string][]*conn, data map[string]*MapGates, frontier []string) {
	var b strings.Builder
	b.WriteString("# The Open World — zone connection map\n\n")
	b.WriteString("GENERATED by tools/owworld from the original maps' exit-trigger data.\n")
	b.WriteString("Every edge is bidirectional in the openworld. Dashed edges were one-way\n")
	b.WriteString("in the original campaign (the return gate is placed at the zone's player\n")
	b.WriteString("start — marked `virtual`).\n\n")

	b.WriteString("```mermaid\ngraph TD\n")
	b.WriteString("    classDef hub fill:#7c5cff,color:#fff\n")
	seen := map[string]bool{}
	var keys []string
	for m := range conns {
		keys = append(keys, m)
	}
	sort.Strings(keys)
	for _, m := range keys {
		for _, c := range conns[m] {
			a, z := m, c.to
			key := a + "|" + z
			rkey := z + "|" + a
			if seen[key] || seen[rkey] {
				continue
			}
			seen[key] = true
			// was the reverse direction virtual?
			revVirtual := true
			for _, rc := range conns[z] {
				if rc.to == a && !rc.virtual {
					revVirtual = false
				}
			}
			edge := " <--> "
			if c.virtual || revVirtual {
				edge = " -.-> "
				if c.virtual && !revVirtual {
					a, z = z, a // draw from the real-gate side
				}
			}
			fmt.Fprintf(&b, "    %s%s%s\n", a, edge, z)
		}
	}
	b.WriteString("    class wiz02a hub\n")
	b.WriteString("```\n\n")
	b.WriteString("Dashed = original one-way transition, return gate is virtual (at the\n")
	b.WriteString("destination's player start). Solid = real gates on both sides.\n\n")

	b.WriteString("## Gates by zone\n\n")
	b.WriteString("| Zone | Gate at | Leads to | Kind |\n|---|---|---|---|\n")
	for _, m := range keys {
		for _, c := range conns[m] {
			kind := "original doorway"
			if c.virtual {
				kind = "virtual (player start)"
			}
			nm := names[c.to]
			if nm == "" {
				nm = c.to
			}
			fmt.Fprintf(&b, "| ow_%s | %.0f, %.0f | ow_%s (%s) | %s |\n", m, c.pos.X, c.pos.Y, c.to, nm, kind)
		}
	}
	b.WriteString("\n## Frontier (connections outside the current world)\n\n")
	if len(frontier) == 0 {
		b.WriteString("none\n")
	}
	sort.Strings(frontier)
	for _, f := range frontier {
		b.WriteString("- " + f + "\n")
	}
	b.WriteString("\n## Wizard start\n\nNew Open World wizards begin in **ow_wiz02a (Galava)** — the hub.\n")

	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		panic(err)
	}
}
