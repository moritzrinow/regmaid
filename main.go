package main

import (
	"os"

	"github.com/moritzrinow/regmaid/internal/regmaid"
)

func main() {
	if err := regmaid.Execute(); err != nil {
		os.Exit(1)
	}
}
