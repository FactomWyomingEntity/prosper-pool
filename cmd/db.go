package cmd

import (
	"fmt"

	"github.com/FactomWyomingEntity/private-pool/authentication"
	"github.com/FactomWyomingEntity/private-pool/database"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	db.AddCommand(makeAdmin)
	rootCmd.AddCommand(db)
}

var db = &cobra.Command{
	Use:   "db",
	Short: "Any direct db interactions can be done through this cli.",
	Long: "All db calls require the db parts of the config to be defined. " +
		"The cli calls interact directly with the database, so care should be taken.",
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
