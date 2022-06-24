package cmd

import (
	"fmt"

	"github.com/nucleuscloud/cli/internal/pkg/auth"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logs a user out of their Nucleus account.",
	Long:  "Logs a user out of their Nucleus account.",

	RunE: func(cmd *cobra.Command, args []string) error {
		authClient, err := auth.NewAuthClient(utils.Auth0BaseUrl, utils.Auth0ClientId, utils.ApiAudience)
		if err != nil {
			return err
		}

		logoutUrl, err := authClient.GetLogoutUrl()
		if err != nil {
			return err
		}

		err = webbrowser.Open(logoutUrl)
		if err != nil {
			fmt.Println("There was an issue opening the web browser, proceed to the following url to fully logout of the system", logoutUrl)
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
}
