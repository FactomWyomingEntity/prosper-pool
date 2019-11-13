package main

import (
	"fmt"
	"runtime"

	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/spf13/cobra"
)

var properties = &cobra.Command{
	Use:   "properties",
	Short: "Properties of the miner executable",
	//PersistentPreRunE: OpenConfig,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Properties")
		fmt.Printf("\t%30s: %s\n", "Build Verison", config.CompiledInVersion)
		fmt.Printf("\t%30s: %s\n", "Build Commit", config.CompiledInBuild)
		fmt.Printf("\t%30s: %s\n", "GoLang Version", runtime.Version())
	},
}
