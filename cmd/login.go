package cmd

import (
	"context"
	"fmt"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/pkg/auth"
	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
)

// loginCmd represents the login command
var auth0Cmd = &cobra.Command{
	Use:   "login",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account. `,

	RunE: func(cmd *cobra.Command, args []string) error {
		authClient, err := auth.NewAuthClient(auth0BaseUrl, auth0ClientId, apiAudience)
		if err != nil {
			return err
		}

		deviceResponse, err := authClient.GetDeviceCode(scopes)
		if err != nil {
			return err
		}

		// fmt.Println("Visit the following URL to login: ", deviceResponse.VerificationURIComplete)
		fmt.Println("Your activation code is: ", deviceResponse.UserCode)
		cliPrompt("Press [Enter] to continue in the web browser...", "")

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
		err = setNucleusAuthFile(NucleusAuth{
			AccessToken:  tokenResponse.AccessToken,
			RefreshToken: tokenResponse.RefreshToken,
			IdToken:      tokenResponse.IdToken,
		})

		if err != nil {
			return err
		}

		conn, err := newAuthenticatedConnection(tokenResponse.AccessToken)
		if err != nil {
			return err
		}

		defer conn.Close()

		nucleusClient := pb.NewCliServiceClient(conn)

		fmt.Println("Attempting to register user in Nucleus system...")

		_, err = nucleusClient.ResolveUser(context.Background(), &pb.ResolveUserRequest{}, getGrpcTrailer())
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
