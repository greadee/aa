// Command archtest enforces aa module layering rules.
//
// Usage (from the tools module):
//
//	go run ./cmd/archtest -root ..
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/greadee/aa/tools/archtest"
)

func main() {
	root := flag.String("root", "..", "repository root containing the product modules")
	flag.Parse()

	violations, err := archtest.CheckWorkspace(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "archtest:", err)
		os.Exit(2)
	}
	boundaries, err := archtest.CheckVisualizerBoundaries(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "archtest:", err)
		os.Exit(2)
	}
	if len(violations) == 0 && len(boundaries) == 0 {
		fmt.Println("architecture boundaries ok")
		return
	}
	for _, v := range violations {
		fmt.Fprintln(os.Stderr, v.String())
	}
	for _, v := range boundaries {
		fmt.Fprintln(os.Stderr, v.String())
	}
	os.Exit(1)
}
