package main

import (
	"cashflow_backend/internal/platform/app"
	"fmt"
	"os"
)

func main() {
	a := app.New()

	if err := a.Bootstrap(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed: %v\n", err)
		os.Exit(1)
	}

	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "application failed: %v\n", err)
		os.Exit(1)
	}
}
