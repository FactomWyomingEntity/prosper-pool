package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/FactomWyomingEntity/private-pool/exit"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Config Stuff
	ConfigHost           = "pool.host"
	ConfigNumGoRountines = "miner.threads"
	ConfigUserName       = "miner.username"
	ConfigMinerName      = "miner.minerid"
)

func init() {
	rootCmd.Flags().StringP("config", "c", "", "config path location")

	// Should be set by the user
	rootCmd.Flags().StringP("user", "u", "", "Username to log into the mining pool")
	rootCmd.Flags().StringP("minerid", "m", GenerateMinerID(), "Minerid should be unique per mining machine")

	// Defaults
	rootCmd.Flags().StringP("poolhost", "s", "localhost:1234", "URL to connect to the pool")
	rootCmd.Flags().IntP("miners", "t", runtime.NumCPU(), "Number of mining threads")
}

// Pool entry point
func main() {
	Execute()
}

// Execute is cobra's entry point
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:               "prosper-miner",
	Short:             "Launch miner to communicate with the prosper mining pool.",
	PersistentPreRunE: OpenConfig,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)

		username, minerid := viper.GetString(ConfigUserName), viper.GetString(ConfigMinerName)
		// TODO: Add version number
		log.Infof("Initiated Prosper Miner")
		log.Infof("Username: %s, MinerID: %s", username, minerid)
		var _ = ctx

		// TODO: Call miner
	},
}

func OpenConfig(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	configCustom := true
	if configPath == "" {
		// TODO: Fix windows to be /Users/
		configPath = "$HOME/.prosper/prosper-miner.toml" // Default
		configCustom = false
	}

	path := os.ExpandEnv(configPath)

	dir := filepath.Dir(path)
	name := filepath.Base(path)
	viper.AddConfigPath(dir)

	ext := filepath.Ext(name)
	viper.SetConfigName(strings.TrimSuffix(name, ext))

	info, err := os.Stat(path)
	exists := info != nil && !os.IsNotExist(err)

	// Set default config values
	SetDefaults(cmd)

	// If it does not exist, and not user specified, we will make it
	if !exists && !configCustom {
		if u, _ := cmd.Flags().GetString("user"); u == "" {
			return fmt.Errorf("no config found, username MUST be specified with -u <username>")
		}

		log.Infof("Writing config to file at %s", path)
		// Make the config
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return fmt.Errorf("failed to make config path: %s", err.Error())
		}

		err = viper.WriteConfigAs(path)
		if err != nil {
			return fmt.Errorf("failed to write config: %s", err.Error())
		}
	} else if !exists && configCustom {
		return fmt.Errorf("error loading custom config path: %s", err.Error())
	} else {
		log.Infof("Using existing config")
	}

	// Read the config
	return viper.ReadInConfig()
}

func SetDefaults(cmd *cobra.Command) {
	_ = viper.BindPFlag(ConfigHost, cmd.Flags().Lookup("poolhost"))
	_ = viper.BindPFlag(ConfigNumGoRountines, cmd.Flags().Lookup("miners"))
	_ = viper.BindPFlag(ConfigUserName, cmd.Flags().Lookup("user"))
	_ = viper.BindPFlag(ConfigMinerName, cmd.Flags().Lookup("minerid"))
}

// GenerateMinerID has to be random
func GenerateMinerID() string {
	return NewRandomName(time.Now().UnixNano()).Haikunate()
}
