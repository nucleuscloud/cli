package cmd

import (
	"fmt"
	"strings"

	"github.com/nucleuscloud/cli/internal/utils"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account and stores an access token locally for later use.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		serviceAccount, err := cmd.Flags().GetBool("service-account")
		if err != nil {
			return err
		}
		clientId, err := cmd.Flags().GetString("client-id")
		if err != nil {
			return err
		}
		if serviceAccount {
			clientId = strings.TrimSpace(clientId)
			if clientId == "" {
				return fmt.Errorf("must provide client id")
			}

			secretResult, err := getSecretValue()
			if err != nil {
				return err
			}

			return utils.ClientLogin(ctx, clientId, secretResult.value)
		}
		return utils.OAuthLogin(ctx)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().BoolP("service-account", "s", false, "login using a service account")
	loginCmd.Flags().StringP("client-id", "", "", "service account client id")
}
