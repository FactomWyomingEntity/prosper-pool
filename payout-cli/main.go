package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/FactomWyomingEntity/prosper-pool/config"

	"github.com/Factom-Asset-Tokens/factom"

	"github.com/FactomWyomingEntity/prosper-pool/accounting"
	"github.com/pegnet/pegnetd/fat/fat2"

	"github.com/spf13/cobra"
)

func init() {
	// Defaults
	rootCmd.PersistentFlags().StringP("factomdhost", "s", "http://localhost:8088/v2", "factomd api url")
	rootCmd.PersistentFlags().StringP("walletdhost", "w", "http://localhost:8089", "factom-walletd url")

	rootCmd.AddCommand(pay)
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
	Use:   "payout-cli",
	Short: "payout assist tool",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var pay = &cobra.Command{
	Use:   "pay <pay.json file> <source-FA> <ECAddress> <reciept.json>",
	Short: "Pay users on pegnet",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename, source, payer, receipt := args[0], args[1], args[2], args[3]

		info, err := os.Stat(receipt)
		exists := info != nil && !os.IsNotExist(err)
		if exists {
			return fmt.Errorf("%s already exists. Receipt must be a new file", receipt)
		}

		cl := factomdClient(cmd)
		file, err := os.OpenFile(filename, os.O_RDONLY, 0777)
		if err != nil {
			return fmt.Errorf("error opening file: %s", err.Error())
		}
		defer file.Close()

		recFile, err := os.OpenFile(receipt, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			return err
		}
		defer recFile.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			return fmt.Errorf("error reading file: %s", err.Error())
		}

		var payments []accounting.Paid
		err = json.Unmarshal(data, &payments)
		if err != nil {
			return fmt.Errorf("unable parsing file: %s", err.Error())
		}

		poolAddr, err := factom.NewFAAddress(source)
		if err != nil {
			return fmt.Errorf("bad FA address: %s", err.Error())
		}

		// Construct the transaction
		var batch fat2.TransactionBatch
		batch.Version = 1
		batch.ChainID = factom.NewBytes32(config.TransactionChain[:])
		for _, pay := range payments {
			var tx fat2.Transaction
			if pay.PaymentAmount < 0 {
				return fmt.Errorf("%s is below 0 in paymen", pay.PayoutAddress)
			}
			tx.Input.Amount = uint64(pay.PaymentAmount)
			tx.Input.Address = poolAddr
			tx.Input.Type = fat2.PTickerPEG
			tx.Transfers = make([]fat2.AddressAmountTuple, 1)
			tx.Transfers[0].Amount = uint64(pay.PaymentAmount)
			tx.Transfers[0].Address, err = factom.NewFAAddress(pay.PayoutAddress)
			if err != nil {
				return fmt.Errorf("%s is not a valid payout adress: %s", pay.PayoutAddress, err.Error())
			}

			batch.Transactions = append(batch.Transactions, tx)
		}

		err = batch.MarshalEntry()
		if err != nil {
			return fmt.Errorf("failed to marshal tx: %s", err.Error())
		}

		if _, err := batch.Entry.Cost(); err != nil {
			fmt.Println("If your entry is over 10KB, you can split the pay.json into parts manually.")
			return fmt.Errorf("error with entry: %s", err.Error())
		}

		priv, err := poolAddr.GetFsAddress(cl)
		if err != nil {
			return fmt.Errorf("unable to get private key: %s\n", err.Error())
		}

		batch.Sign(priv)

		payment, err := factom.NewECAddress(payer)
		if err != nil {
			return fmt.Errorf("unable to get private key: %s\n", err.Error())
		}

		es, err := payment.GetEsAddress(cl)
		if err != nil {
			return fmt.Errorf("unable to get private key: %s", err.Error())
		}

		txid, err := batch.ComposeCreate(cl, es)
		if err != nil {
			return fmt.Errorf("unable to submit entry: %s", err.Error())
		}

		for i := range payments {
			payments[i].EntryHash = batch.Entry.Hash.String()
		}

		data, err = json.Marshal(payments)
		if err != nil {
			fmt.Printf("failed to make reciept: %s\n", err.Error())
		} else {
			_, err = recFile.Write(data)
			if err != nil {
				fmt.Printf("failed to make reciept: %s\n", err.Error())
			}
		}

		fmt.Println("Payment submitted to the network")
		fmt.Printf("EntryHash: %s\n", batch.Entry.Hash.String())
		fmt.Printf("   Commit: %s\n", txid.String())

		return nil
	},
}

func factomdClient(cmd *cobra.Command) *factom.Client {
	cl := factom.NewClient()
	cl.FactomdServer, _ = cmd.Flags().GetString("factomdhost")
	cl.WalletdServer, _ = cmd.Flags().GetString("walletdhost")

	//cl.Factomd.DebugRequest = true
	//cl.Walletd.DebugRequest = true
	return cl
}
