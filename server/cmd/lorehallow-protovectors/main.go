// Command lorehallow-protovectors writes the golden protocol vectors to a
// directory (default ../proto/testdata, relative to server/). Run through
// `make proto`.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dunamismax/lorehallow/server/internal/protovectors"
)

func main() {
	out := flag.String("out", "../proto/testdata", "output directory")
	flag.Parse()
	if err := protovectors.Write(*out); err != nil {
		fmt.Fprintln(os.Stderr, "lorehallow-protovectors:", err)
		os.Exit(1)
	}
}
