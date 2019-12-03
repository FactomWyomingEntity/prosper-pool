package main

import (
	"github.com/FactomWyomingEntity/prosper-pool/cmd"
	_ "github.com/mattn/go-sqlite3"
)

// Pool entry point
func main() {
	cmd.Execute()
}
