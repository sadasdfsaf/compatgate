package main

import (
	"fmt"
	"os"

	"github.com/compatgate/compatgate/internal/app/api"
)

func main() {
	if err := api.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
