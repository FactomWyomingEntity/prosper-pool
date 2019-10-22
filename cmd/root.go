package cmd

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/FactomWyomingEntity/private-pool/accounting"
	"github.com/FactomWyomingEntity/private-pool/authentication"
	"github.com/FactomWyomingEntity/private-pool/config"
	"github.com/FactomWyomingEntity/private-pool/database"
	"github.com/FactomWyomingEntity/private-pool/engine"
	"github.com/FactomWyomingEntity/private-pool/exit"
	"github.com/FactomWyomingEntity/private-pool/loghelp"
	"github.com/FactomWyomingEntity/private-pool/pegnet"
	"github.com/FactomWyomingEntity/private-pool/stratum"
	"github.com/qor/session/manager"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(testMiner)
	rootCmd.AddCommand(testSync)
	rootCmd.AddCommand(testAccountant)
	rootCmd.AddCommand(testAuth)
	rootCmd.AddCommand(testStratum)

	rootCmd.PersistentFlags().String("log", "info", "Change the logging level. Can choose from 'trace', 'debug', 'info', 'warn', 'error', or 'fatal'")
	rootCmd.PersistentFlags().String("phost", "192.168.32.2", "Postgres host url")
	rootCmd.PersistentFlags().Int("pport", 5432, "Postgres host port")
	rootCmd.PersistentFlags().Bool("testing", false, "Enable testing mode")
	rootCmd.PersistentFlags().Int("act", 0, "Enable a custom activation height for testing mode")
	testMiner.Flags().BoolP("verbose", "v", false, "Verbosity (if enabled, print messages)")
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

		pool, err := engine.Setup(viper.GetViper())
		if err != nil {
			log.WithError(err).Fatal("failed to launch pool")
		}

		pool.Run(ctx)
	},
}

var testStratum = &cobra.Command{
	Use:    "stratum",
	Short:  "Launch the stratum server",
	PreRun: SoftReadConfig, // TODO: Do a hard read
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)

		fmt.Println("TODO") // TODO: Config setup + launch of everything
		s, err := stratum.NewServer(viper.GetViper())
		if err != nil {
			log.WithError(err).Fatal("failed to launch stratum server")
		}

		go func() {
			keyboardReader := bufio.NewReader(os.Stdin)
			for {
				userCommand, _ := keyboardReader.ReadString('\n')
				words := strings.Fields(userCommand)
				if len(words) > 0 {
					switch words[0] {
					case "listclients", "listminers":
						fmt.Println(strings.Join(s.Miners.ListMiners()[:], ", "))
					case "showmessage":
						if len(words) > 2 {
							s.ShowMessage(words[1], strings.Join(words[2:], " "))
						}
					case "getversion":
						if len(words) > 1 {
							s.GetVersion(words[1])
						}
					case "notify":
						if len(words) > 3 {
							s.SingleClientNotify(words[1], words[2], words[3], "")
						}
					case "reconnect":
						if len(words) > 4 {
							s.ReconnectClient(words[1], words[2], words[3], words[4])
						}
					default:
						fmt.Println("Server command not supported: ", words[0])
					}
				}
			}
		}()

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

	// Handle testing mode
	if ok, _ := cmd.Flags().GetBool("testing"); ok {
		act, _ := cmd.Flags().GetUint32("act")
		config.GradingV2Activation = act
		config.PegnetActivation = act
		config.GradingV2Activation = act
		config.TransactionConversionActivation = act
	}

}

// TODO: Move testMiner to its own binary
var testMiner = &cobra.Command{
	Use:    "miner",
	Short:  "Launch a miner",
	PreRun: SoftReadConfig,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)

		verbosityEnabled, _ := cmd.Flags().GetBool("verbose")
		client, err := stratum.NewClient(verbosityEnabled)
		if err != nil {
			panic(err)
		}

		exit.GlobalExitHandler.AddExit(func() error {
			return client.Close()
		})

		err = client.Connect("localhost:1234")
		if err != nil {
			panic(err)
		}

		client.Handshake()

		keyboardReader := bufio.NewReader(os.Stdin)
		go func() {
			for {
				userCommand, _ := keyboardReader.ReadString('\n')
				words := strings.Fields(userCommand)
				if len(words) > 0 {
					switch words[0] {
					case "getopr":
						if len(words) > 1 {
							client.GetOPRHash(words[1])
						}
					case "suggesttarget":
						if len(words) > 1 {
							client.SuggestTarget(words[1])
						}
					default:
						fmt.Println("Client command not supported: ", words[0])
					}
				}
			}
		}()

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
		exit.GlobalExitHandler.AddExit(func() error {
			return db.Close()
		})

		p, err := pegnet.NewPegnetNode(conf, db)
		if err != nil {
			panic(err)
		}

		p.DBlockSync(ctx)
		var _ = ctx
	},
}

var testAccountant = &cobra.Command{
	Use:    "accountant",
	Short:  "Run the pool accountant",
	PreRun: SoftReadConfig,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)
		conf := viper.GetViper()

		db, err := database.New(conf)
		if err != nil {
			panic(err)
		}

		a, err := accounting.NewAccountant(conf, db.DB)
		if err != nil {
			panic(err)
		}

		go func() {
			users := 3
			for i := 0; i < 10; i++ {
				if ctx.Err() != nil {
					return
				}

				job := fmt.Sprintf("%d", i)
				a.NewJob(job) // Force the new job
				for u := 0; u < users; u++ {
					for w := 0; w < 3; w++ {
						a.ShareChannel() <- &accounting.Share{
							JobID:      job,
							Difficulty: rand.Float64() * 20,
							Accepted:   false,
							MinerID:    fmt.Sprintf("user-%d_%d", u, w),
							UserID:     fmt.Sprintf("user-%d", u),
						}
					}
				}

				time.Sleep(100 * time.Millisecond)
				a.RewardChannel() <- &accounting.Reward{
					JobID:      job,
					PoolReward: 200 * 1e8 * 12,
					Winning:    12,
					Graded:     15,
				}
			}
		}()

		a.Listen(ctx)
		var _ = ctx
	},
}

var testAuth = &cobra.Command{
	Use:    "auth",
	Short:  "Run the pegnet authenticator",
	PreRun: SoftReadConfig,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		exit.GlobalExitHandler.AddCancel(cancel)
		conf := viper.GetViper()

		db, err := database.New(conf)
		if err != nil {
			panic(err)
		}
		exit.GlobalExitHandler.AddExit(func() error {
			return db.Close()
		})

		a, err := authentication.NewAuthenticator(viper.GetViper(), db.DB)
		if err != nil {
			panic(err)
		}

		mux := http.NewServeMux()

		// Mount Auth to Router
		mux.Handle("/auth/", a.NewServeMux())
		var _ = ctx
		srv := http.Server{Addr: ":9000", Handler: manager.SessionManager.Middleware(mux)}
		go func() {
			err := srv.ListenAndServe()
			if err != nil {
				fmt.Println(err)
			}
		}()

	InfiniteLoop:
		for {
			select {
			case <-ctx.Done():
				_ = srv.Close()
				break InfiniteLoop
			}
		}
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

	log.StandardLogger().Hooks.Add(&loghelp.ContextHook{})
}
