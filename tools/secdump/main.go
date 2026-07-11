// secdump: low-level section debugging for a single map.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/opennox/libs/maps"
)

func main() {
	mapsDir := flag.String("maps", `C:\GOG Games\Nox\maps`, "maps dir")
	name := flag.String("map", "wiz01a", "map name")
	flag.Parse()

	m, err := maps.ReadMap(filepath.Join(*mapsDir, *name))
	if err != nil {
		fmt.Println("read err:", err)
	}
	if m == nil {
		return
	}
	if m.Waypoints != nil {
		fmt.Println("waypoints:")
		for _, w := range m.Waypoints.Waypoints {
			fmt.Printf("  id=%-4d pos=(%.0f,%.0f) flags=%d name=%q links=%d\n",
				w.ID, w.Pos.X, w.Pos.Y, w.Flags, w.Name, len(w.Links))
		}
	}
	fmt.Println("unknown sections:")
	for _, s := range m.Unknown {
		fmt.Printf("  %-24s %8d bytes  head=%s\n", s.Name, len(s.Data), hex.EncodeToString(s.Data[:min(24, len(s.Data))]))
	}
	for _, s := range m.Unknown {
		if s.Name == "ObjectTOC" {
			var toc maps.ObjectsTOC
			if err := toc.UnmarshalBinary(s.Data); err != nil {
				fmt.Println("toc err:", err)
				continue
			}
			fmt.Printf("TOC vers=%d entries=%d\n", toc.Vers, len(toc.TOC))
			for i, t := range toc.TOC {
				if i < 25 {
					fmt.Printf("  ind=%d type=%q\n", t.Ind, t.Type)
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
