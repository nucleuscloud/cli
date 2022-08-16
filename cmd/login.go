package cmd

import (
	"context"
	"fmt"

	"github.com/nucleuscloud/cli/internal/pkg/auth"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	mgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/mgmt/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
)

// loginCmd represents the login command
var auth0Cmd = &cobra.Command{
	Use:   "login",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account and stores an access token locally for later use.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		authClient, err := auth.NewAuthClientByEnv(utils.GetEnv())
		if err != nil {
			return err
		}

		deviceResponse, err := authClient.GetDeviceCode(utils.Scopes)
		if err != nil {
			return err
		}

		fmt.Println("Your activation code is: ", deviceResponse.UserCode)

		err = webbrowser.Open(deviceResponse.VerificationURIComplete)
		if err != nil {
			fmt.Println("There was an issue opening the web browser, proceed to the following URL to continue logging in: ", deviceResponse.VerificationURIComplete)
		}

		tokenResponse, err := authClient.PollDeviceAccessToken(deviceResponse)

		if err != nil {
			// handle expired token error by re-prompting
			fmt.Println("There was an error. Please try logging in again")
			return err
		}
		err = config.SetNucleusAuthFile(config.NucleusAuthConfig{
			AccessToken:  tokenResponse.AccessToken,
			RefreshToken: tokenResponse.RefreshToken,
			IdToken:      tokenResponse.IdToken,
		})

		if err != nil {
			return err
		}

		conn, err := utils.NewAuthenticatedConnection(tokenResponse.AccessToken)
		if err != nil {
			return err
		}

		defer conn.Close()

		nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)

		if verbose {
			fmt.Println("Attempting to register user in Nucleus system...")
		}

		_, err = nucleusClient.SetUser(context.Background(), &mgmtv1alpha1.SetUserRequest{}, utils.GetGrpcTrailer())
		if err != nil {
			return err
		}
		fmt.Println("User successfully resolved in Nucleus system!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(auth0Cmd)
}
