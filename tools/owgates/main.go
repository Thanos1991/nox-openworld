// owgates extracts gate anchors from the original campaign maps:
//
//   - PlayerStart positions
//   - InvisibleExitArea triggers: position AND embedded destination map —
//     the complete original transition topology, straight from object data.
//
// Exit triggers lining one doorway are clustered into a single logical gate
// (position = cluster center, destination = the common target map).
//
// Record framing (verified byte-level against the GOG final maps):
// section = [u16 vers] then records [u16 typeInd][pad to 8-alignment
// relative to section start][u64 size][size bytes], terminated by ind 0.
// Record payload: [u16 xferVers][u16 sub (vers<61 only)][u32 id][u32 flags]
// [f32 x][f32 y][u8] then for exits [u32 len][dest string]...
//
// Usage: owgates [-maps dir] [-out data\ow_gates.json] <map> [<map> ...]
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/opennox/libs/ifs"
	"github.com/opennox/libs/maps"
)

type Pos struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Gate struct {
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	Dest string  `json:"dest,omitempty"` // destination map (lowercase, no ext)
	N    int     `json:"n"`              // trigger objects merged into this gate
}

type MapGates struct {
	Starts []Pos  `json:"starts,omitempty"`
	Gates  []Gate `json:"gates,omitempty"`
	Error  string `json:"error,omitempty"`
}

const clusterDist = 138 // ~6 tiles: exit areas lining one doorway merge

func main() {
	mapsDir := flag.String("maps", `C:\GOG Games\Nox\maps`, "Nox maps directory (source)")
	out := flag.String("out", `data\ow_gates.json`, "output JSON path")
	flag.Parse()
	res := make(map[string]*MapGates)
	for _, name := range flag.Args() {
		name = strings.ToLower(name)
		mg := &MapGates{}
		res[name] = mg
		if err := extract(*mapsDir, name, mg); err != nil {
			mg.Error = err.Error()
			fmt.Println("WARN", name+":", err)
		}
		fmt.Printf("%s: %d starts, %d gates", name, len(mg.Starts), len(mg.Gates))
		for _, g := range mg.Gates {
			fmt.Printf("  [%s @%.0f,%.0f]", g.Dest, g.X, g.Y)
		}
		fmt.Println()
	}
	os.MkdirAll(filepath.Dir(*out), 0o755)
	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(res)
}

type exitRec struct {
	pos  Pos
	dest string
}

func extract(mapsDir, name string, mg *MapGates) error {
	rc, err := ifs.Open(filepath.Join(mapsDir, name, name+maps.Ext))
	if err != nil {
		return err
	}
	defer rc.Close()
	rd, err := maps.NewReader(rc)
	if err != nil {
		return err
	}
	sects, err := rd.ReadSectionsRaw()
	if err != nil {
		return err
	}
	var toc maps.ObjectsTOC
	var d []byte
	for _, s := range sects {
		switch s.Name {
		case "ObjectTOC":
			if err := toc.UnmarshalBinary(s.Data); err != nil {
				return fmt.Errorf("toc: %w", err)
			}
		case "ObjectData":
			d = s.Data
		}
	}
	if d == nil {
		return fmt.Errorf("no ObjectData section")
	}
	types := make(map[uint16]string)
	for _, t := range toc.TOC {
		types[t.Ind] = t.Type
	}

	var exits []exitRec
	off := 2 // section version u16
	for {
		if off+2 > len(d) {
			return fmt.Errorf("object stream truncated at %d", off)
		}
		ind := binary.LittleEndian.Uint16(d[off:])
		if ind == 0 {
			break
		}
		szOff := off + 2 + (8-(off+2)%8)%8 // engine pads to 8-byte alignment
		if szOff+8 > len(d) {
			return fmt.Errorf("no room for record size at %d", off)
		}
		sz := binary.LittleEndian.Uint64(d[szOff:])
		end := szOff + 8 + int(sz)
		if int(sz) < 0 || end > len(d) {
			return fmt.Errorf("bad record size %d at %d", sz, off)
		}
		rec := d[szOff+8 : end]
		off = end

		typ := types[ind]
		if typ != "PlayerStart" && typ != "InvisibleExitArea" {
			continue
		}
		p, tail, perr := recPos(rec)
		if perr != nil {
			fmt.Printf("  %s: skip %s record: %v\n", name, typ, perr)
			continue
		}
		if typ == "PlayerStart" {
			mg.Starts = append(mg.Starts, p)
			continue
		}
		exits = append(exits, exitRec{pos: p, dest: exitDest(tail)})
	}
	mg.Gates = clusterGates(exits)
	return nil
}

// recPos parses the common xfer header and returns the position plus the
// remaining bytes after the u8 that follows it.
func recPos(rec []byte) (Pos, []byte, error) {
	if len(rec) < 4 {
		return Pos{}, nil, fmt.Errorf("record too short: %d", len(rec))
	}
	vers := binary.LittleEndian.Uint16(rec[0:])
	posOff := 0
	switch {
	case vers >= 61 && vers <= 64:
		posOff = 10 // [vers u16][extent u32][id u32]
	case vers >= 40 && vers < 61:
		posOff = 12 // [vers u16][sub u16][id u32][flags u32]
	default:
		return Pos{}, nil, fmt.Errorf("unsupported xfer vers %d", vers)
	}
	if len(rec) < posOff+9 {
		return Pos{}, nil, fmt.Errorf("record too short for pos: %d", len(rec))
	}
	x := math.Float32frombits(binary.LittleEndian.Uint32(rec[posOff:]))
	y := math.Float32frombits(binary.LittleEndian.Uint32(rec[posOff+4:]))
	if x <= 0 || y <= 0 || x > 30000 || y > 30000 {
		return Pos{}, nil, fmt.Errorf("implausible position %v,%v", x, y)
	}
	return Pos{X: x, Y: y}, rec[posOff+9:], nil
}

// exitDest extracts the destination map from an InvisibleExitArea payload
// tail: [u32 len][string]... — normalized to lowercase without extension.
func exitDest(tail []byte) string {
	if len(tail) < 4 {
		return ""
	}
	n := int(binary.LittleEndian.Uint32(tail[0:]))
	if n <= 0 || 4+n > len(tail) {
		return ""
	}
	s := string(tail[4 : 4+n])
	s = strings.TrimRight(s, "\x00")
	s = strings.ToLower(s)
	// destinations often carry a waypoint suffix ("Map.map:SomeWP")
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSuffix(s, ".map")
	return s
}

func clusterGates(exits []exitRec) []Gate {
	n := len(exits)
	if n == 0 {
		return nil
	}
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			// only cluster triggers with the same destination
			if exits[i].dest != exits[j].dest {
				continue
			}
			dx := float64(exits[i].pos.X - exits[j].pos.X)
			dy := float64(exits[i].pos.Y - exits[j].pos.Y)
			if math.Hypot(dx, dy) <= clusterDist {
				parent[find(i)] = find(j)
			}
		}
	}
	type acc struct {
		x, y float64
		n    int
		dest string
	}
	sums := make(map[int]*acc)
	for i, e := range exits {
		r := find(i)
		s := sums[r]
		if s == nil {
			s = &acc{dest: e.dest}
			sums[r] = s
		}
		s.x += float64(e.pos.X)
		s.y += float64(e.pos.Y)
		s.n++
	}
	var out []Gate
	for _, s := range sums {
		out = append(out, Gate{
			X: float32(s.x / float64(s.n)), Y: float32(s.y / float64(s.n)),
			Dest: s.dest, N: s.n,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dest != out[j].Dest {
			return out[i].Dest < out[j].Dest
		}
		return out[i].X < out[j].X
	})
	return out
}
