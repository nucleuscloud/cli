/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"net/mail"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
)

var accountsInviteCmd = &cobra.Command{
	Use: "invite <email>",
	Aliases: []string{
		"inv",
	},
	Short: "Allows you to invite a user to your account",
	Long:  "Invites a user to access your account. *This will give them admin permissions to your account today!",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("must provide an email to invite someone to your account")
		}
		if len(args) > 1 {
			return fmt.Errorf("may only invite one email at a time")
		}
		email := args[0]
		_, err := mail.ParseAddress(email)
		if err != nil {
			return err
		}

		return inviteUser(email)
	},
}

func init() {
	accountsCmd.AddCommand(accountsInviteCmd)
}

func inviteUser(email string) error {
	conn, err := utils.NewApiConnection(utils.ApiConnectionConfig{
		AuthBaseUrl:  utils.Auth0BaseUrl,
		AuthClientId: utils.Auth0ClientId,
		ApiAudience:  utils.ApiAudience,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := pb.NewCliServiceClient(conn)
	_, err = cliClient.InviteUserToAccount(context.Background(), &pb.InviteUserToAccountRequest{
		Email: email,
	})
	if err != nil {
		return err
	}
	fmt.Printf("invite for %s sent!\n", email)
	return nil
}
