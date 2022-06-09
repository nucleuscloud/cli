package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account. `,

	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := getNucleusAuthConfig()

		if err != nil {
			return err
		}

		fmt.Println("Checking token: ", config.AccessToken)

		err = ensureValidToken(config.AccessToken)

		if err != nil {
			return err
		} else {
			fmt.Println("access token is valid!")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
