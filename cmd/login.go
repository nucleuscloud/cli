package cmd

import (
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account and stores an access token locally for later use.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		return utils.Login(ctx)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
