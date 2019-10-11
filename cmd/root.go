package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/FactomWyomingEntity/private-pool/exit"
	"github.com/FactomWyomingEntity/private-pool/stratum"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
}
