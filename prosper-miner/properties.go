package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var properties = &cobra.Command{
	Use:               "properties",
	Short:             "Properties of the miner executable",
	PersistentPreRunE: OpenConfig,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Properties")
		fmt.Printf("\tGoLang Version: %s\n", runtime.Version())
	},
}
