package cmd

import (
	"fmt"

	"github.com/nucleuscloud/cli/internal/auth"
	"github.com/nucleuscloud/cli/internal/config"
	clienv "github.com/nucleuscloud/cli/internal/env"
	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logs a user out of their Nucleus account.",
	Long:  "Logs a user out of their Nucleus account.",

	RunE: func(cmd *cobra.Command, args []string) error {
		authClient, err := auth.NewAuthClientByEnv(clienv.GetEnv())
		if err != nil {
			return err
		}

		serviceAccount, err := cmd.Flags().GetBool("service-account")
		if err != nil {
			return err
		}

		if !serviceAccount {
			logoutUrl, err := authClient.GetLogoutUrl()
			if err != nil {
				return err
			}

			err = webbrowser.Open(logoutUrl)
			if err != nil {
				fmt.Println("There was an issue opening the web browser, proceed to the following url to fully logout of the system", logoutUrl)
			}
		}

		err = config.ClearNucleusAuthFile()
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
	logoutCmd.Flags().BoolP("service-account", "s", false, "logout service account")
}
