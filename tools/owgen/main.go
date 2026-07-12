// owgen generates the open-world (ow_) map set from the original campaign maps.
//
// For each requested map it writes maps/ow_<name>/ow_<name>.map — a byte-level
// clone with the compiled campaign script sections (ScriptObject, ScriptData)
// stripped, so the linear story logic, class gates and one-way transitions are
// gone and per-map Lua owns the zone. All other sections (walls, floor,
// objects, waypoints) are preserved unmodified.
//
// It also dumps waypoint tables (data/ow_waypoints.json) used to pick gate
// positions for zone transitions.
//
// The generated .map files contain original game data and are NOT committed;
// rerun this tool against your own install (see deploy-world.ps1).
//
// Usage: owgen -maps "C:\GOG Games\Nox\maps" -out "C:\GOG Games\Nox\maps" -dump data\ow_waypoints.json wiz01a wiz02a
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/opennox/libs/ifs"
	"github.com/opennox/libs/maps"
)

var stripSections = map[string]bool{
	"ScriptObject": true, // compiled NoxScript (story logic, class gates)
	"ScriptData":   true, // script variable state
}

type WaypointDump struct {
	ID   uint32  `json:"id"`
	Name string  `json:"name,omitempty"`
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
}

func main() {
	mapsDir := flag.String("maps", `C:\GOG Games\Nox\maps`, "Nox maps directory (source)")
	outDir := flag.String("out", `C:\GOG Games\Nox\maps`, "maps directory to write ow_ maps into")
	dump := flag.String("dump", "", "optional path for a waypoint JSON dump")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("usage: owgen [-maps dir] [-out dir] [-dump file.json] <map> [<map> ...]")
		os.Exit(2)
	}

	wps := make(map[string][]WaypointDump)
	for _, name := range flag.Args() {
		name = strings.ToLower(name)
		if err := cloneMap(*mapsDir, *outDir, name, wps); err != nil {
			fmt.Println("FAIL", name+":", err)
			os.Exit(1)
		}
	}
	if *dump != "" {
		os.MkdirAll(filepath.Dir(*dump), 0o755)
		f, err := os.Create(*dump)
		if err != nil {
			panic(err)
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		enc.Encode(wps)
		f.Close()
		fmt.Println("waypoints dumped to", *dump)
	}
}

func cloneMap(mapsDir, outDir, name string, wps map[string][]WaypointDump) error {
	srcDir := filepath.Join(mapsDir, name)
	src := filepath.Join(srcDir, name+maps.Ext)
	rc, err := ifs.Open(src)
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

	owName := "ow_" + name
	kept := make([]maps.RawSection, 0, len(sects))
	for _, s := range sects {
		if stripSections[s.Name] {
			fmt.Printf("  %s: stripped %s (%d bytes)\n", owName, s.Name, len(s.Data))
			continue
		}
		if s.Name == "WayPoints" {
			var w maps.Waypoints
			if err := w.UnmarshalBinary(s.Data); err == nil {
				for _, wp := range w.Waypoints {
					wps[name] = append(wps[name], WaypointDump{ID: wp.ID, Name: wp.Name, X: wp.Pos.X, Y: wp.Pos.Y})
				}
			}
		}
		kept = append(kept, s)
	}

	dstDir := filepath.Join(outDir, owName)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dstDir, owName+maps.Ext))
	if err != nil {
		return err
	}
	defer f.Close()
	wr, err := maps.NewWriter(f, rd.Header())
	if err != nil {
		return err
	}
	if err := wr.WriteRawSections(kept); err != nil {
		return err
	}
	if err := wr.Close(); err != nil {
		return err
	}
	fmt.Printf("  %s: written (%d sections kept)\n", owName, len(kept))
	return nil
}
