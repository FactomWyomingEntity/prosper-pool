package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/FactomWyomingEntity/private-pool/config"
	"github.com/FactomWyomingEntity/private-pool/database"
	"github.com/FactomWyomingEntity/private-pool/exit"
	"github.com/FactomWyomingEntity/private-pool/pegnet"
	"github.com/FactomWyomingEntity/private-pool/stratum"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(testMiner)
	rootCmd.AddCommand(testSync)
	rootCmd.PersistentFlags().String("log", "info", "Change the logging level. Can choose from 'trace', 'debug', 'info', 'warn', 'error', or 'fatal'")
	rootCmd.PersistentFlags().String("phost", "192.168.32.2", "Postgres host url")
	rootCmd.PersistentFlags().Int("pport", 5432, "Postgres host port")
	testMiner.Flags().Bool("v", false, "Verbosity (if enabled, print messages)")
}

// Execute is cobra's entry point
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:              "private-pool",
	Short:            "Launch the private pool",
	PersistentPreRun: rootPreRunSetup,
	PreRun:           SoftReadConfig, // TODO: Do a hard read
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)

		fmt.Println("TODO") // TODO: Config setup + launch of everything
		s, err := stratum.NewServer(viper.GetViper())
		if err != nil {
			log.WithError(err).Fatal("failed to launch stratum server")
		}

		s.Listen(ctx)
	},
}

// rootPreRunSetup is run before the root command
func rootPreRunSetup(cmd *cobra.Command, args []string) {
	// Catch ctl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		log.Info("Gracefully closing")

		// We will give it 3 seconds to close gracefully.
		// If anything is hanging beyond that, just kill it.
		ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
		err := exit.GlobalExitHandler.CloseWithTimeout(ctx)
		if err != nil {
			log.Warn("took too long to close")
			os.Exit(1)
		}
	}()

	config.SetDefaults(viper.GetViper())
	_ = viper.BindPFlag(config.ConfigSQLHost, cmd.Flags().Lookup("phost"))
	_ = viper.BindPFlag(config.ConfigSQLPort, cmd.Flags().Lookup("pport"))
	_ = viper.BindPFlag(config.LoggingLevel, cmd.Flags().Lookup("log"))

}

// TODO: Move testMiner to it's own binary
var testMiner = &cobra.Command{
	Use:    "miner",
	Short:  "Launch a miner",
	PreRun: SoftReadConfig,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)

		verbosityEnabled, _ := cmd.Flags().GetBool("v")
		client, err := stratum.NewClient(verbosityEnabled)
		if err != nil {
			panic(err)
		}

		err = client.Connect("localhost:1234")
		if err != nil {
			panic(err)
		}
		client.Listen(ctx)
	},
}

var testSync = &cobra.Command{
	Use:    "sync",
	Short:  "Run the pegnet sync",
	PreRun: SoftReadConfig,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)
		conf := viper.GetViper()

		db, err := database.New(conf)
		if err != nil {
			panic(err)
		}

		p, err := pegnet.NewPegnetNode(conf, db)
		if err != nil {
			panic(err)
		}

		p.DBlockSync(ctx)
		var _ = ctx
	},
}

// SoftReadConfig will not fail. It can be used for a command that needs the config,
// but is happy with the defaults
func SoftReadConfig(cmd *cobra.Command, args []string) {
	err := viper.ReadInConfig()
	if err != nil {
		log.WithError(err).Debugf("failed to load config")
	}

	initLogger()
}

func initLogger() {
	switch strings.ToLower(viper.GetString(config.LoggingLevel)) {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	}
}
