package main

import (
	"fmt"
	"os"

	"github.com/acgh213/chisel/tui"
)

func main() {
	if err := tui.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "chisel: %v\n", err)
		os.Exit(1)
	}
}
