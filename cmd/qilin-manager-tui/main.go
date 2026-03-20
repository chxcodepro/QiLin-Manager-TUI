package main

import (
	"fmt"
	"os"

	"github.com/chxcodepro/qilin-manager-tui/internal/tui"
)

var version = "dev"

func main() {
	if err := tui.Run(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
