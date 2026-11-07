// Command lorehallow-content validates the game content in content/.
// Run through `make content-lint` (part of `make check`).
//
//	lorehallow-content validate [-root ..]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dunamismax/lorehallow/server/internal/mapdata"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: lorehallow-content validate [-root dir]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	root := fs.String("root", "..", "repository root")
	_ = fs.Parse(os.Args[2:])
	if err := validate(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validate(root string) error {
	maps, err := filepath.Glob(filepath.Join(root, "content", "maps", "*.tmj"))
	if err != nil {
		return err
	}
	if len(maps) == 0 {
		return fmt.Errorf("content: no maps in %s", filepath.Join(root, "content", "maps"))
	}
	failed := 0
	for _, p := range maps {
		if err := mapdata.Validate(root, p); err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("content: %d of %d maps failed validation", failed, len(maps))
	}
	fmt.Printf("content: %d map(s) ok\n", len(maps))
	return nil
}
