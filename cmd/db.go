package cmd

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/FactomWyomingEntity/prosper-pool/web"

	"github.com/FactomWyomingEntity/prosper-pool/accounting"

	"github.com/Factom-Asset-Tokens/base58"
	"github.com/FactomWyomingEntity/prosper-pool/authentication"
	"github.com/FactomWyomingEntity/prosper-pool/database"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	db.AddCommand(makeAdmin)
	db.AddCommand(makeCode)
	db.AddCommand(makePayments)
	db.AddCommand(recordPayments)
	rootCmd.AddCommand(db)
}

var db = &cobra.Command{
	Use:   "db",
	Short: "Any direct db interactions can be done through this cli.",
	Long: "All db calls require the db parts of the config to be defined. " +
		"The cli calls interact directly with the database, so care should be taken.",
}

var recordPayments = &cobra.Command{
	Use:     "record <receipt.json>",
	Short:   "Record the pool payout",
	Example: "prosper db record",
	Args:    cobra.ExactArgs(1),
	PreRun:  SoftReadConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := os.Stat(args[0])
		exists := info != nil && !os.IsNotExist(err)
		if !exists {
			return fmt.Errorf("no recipt file found at %s", args[0])
		}

		db, err := database.New(viper.GetViper())
		if err != nil {
			return err
		}

		a, err := accounting.NewAccountant(viper.GetViper(), db.DB)
		if err != nil {
			return err
		}

		file, err := os.OpenFile(args[0], os.O_RDONLY, 0666)
		if err != nil {
			return err
		}
		defer file.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		var payments []accounting.Paid
		err = json.Unmarshal(data, &payments)
		if err != nil {
			return err
		}

		err = a.WritePayments(payments)
		if err != nil {
			return err
		}

		fmt.Println("Payment data recorded")
		return nil
	},
}

var makePayments = &cobra.Command{
	Use:     "payout <pay.json>",
	Short:   "Will construct a payout tx for the pool",
	Example: "prosper db payout",
	Args:    cobra.ExactArgs(1),
	PreRun:  SoftReadConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := os.Stat(args[0])
		exists := info != nil && !os.IsNotExist(err)
		if exists {
			return fmt.Errorf("%s already exists. Must be a new file", args[0])
		}

		db, err := database.New(viper.GetViper())
		if err != nil {
			return err
		}

		a, err := accounting.NewAccountant(viper.GetViper(), db.DB)
		if err != nil {
			return err
		}

		payments, err := a.CalculatePayments()
		if err != nil {
			return err
		}

		var totalPay int64
		for _, pay := range payments {
			totalPay += pay.PaymentAmount
		}

		data, err := json.Marshal(payments)
		if err != nil {
			return err
		}

		file, err := os.OpenFile(args[0], os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.Write(data)
		if err != nil {
			return err
		}

		fmt.Println("Payment data written to file")
		fmt.Printf("%s PEG needed for the TX\n", web.FactoshiToFactoid(uint64(totalPay)))
		return nil
	},
}

var makeAdmin = &cobra.Command{
	Use:     "admin",
	Short:   "Makes the target user an admin",
	Example: "prosper db admin <email>",
	PreRun:  SoftReadConfig, // TODO: Do a hard read
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.New(viper.GetViper())
		if err != nil {
			panic(err)
		}

		dbErr := db.DB.Model(&authentication.User{}).Where("uid = ?", args[0]).Update("role", "admin")
		if dbErr.Error != nil {
			panic(dbErr.Error)
		}

		fmt.Printf("%d rows affected\n", dbErr.RowsAffected)
	},
}

var makeCode = &cobra.Command{
	Use:     "code",
	Short:   "Makes a new invite code",
	Example: "prosper db code",
	PreRun:  SoftReadConfig, // TODO: Do a hard read
	Run: func(cmd *cobra.Command, args []string) {
		db, err := database.New(viper.GetViper())
		if err != nil {
			panic(err)
		}

		data := make([]byte, 20)
		_, _ = crand.Read(data)
		code := base58.Encode(data)

		a, err := authentication.NewAuthenticator(viper.GetViper(), db.DB)
		if err != nil {
			panic(err)
		}

		err = a.NewCode(code)
		if err != nil {
			fmt.Println("failed to make code")
		}

		fmt.Printf("New Code: %s\n", code)
	},
}
