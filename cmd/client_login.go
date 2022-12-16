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
	"fmt"
	"strings"

	"github.com/nucleuscloud/cli/internal/utils"
	"github.com/spf13/cobra"
)

var clientLoginCmd = &cobra.Command{
	Use:   "client-login",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account and stores an access token locally for later use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		clientId, err := cmd.Flags().GetString("client-id")
		if err != nil {
			return err
		}

		clientId = strings.TrimSpace(clientId)
		if clientId == "" {
			return fmt.Errorf("must provide client id")
		}

		clientSecret, err := cmd.Flags().GetString("client-secret")
		if err != nil {
			return err
		}
		clientSecret = strings.TrimSpace(clientSecret)
		if clientSecret == "" {
			return fmt.Errorf("must provide client secret")
		}

		return utils.ClientLogin(ctx, clientId, clientSecret)
	},
}

func init() {
	rootCmd.AddCommand(clientLoginCmd)

	clientLoginCmd.Flags().StringP("client-id", "", "", "auth client id")
	clientLoginCmd.Flags().StringP("client-secret", "", "", "auth client secret")
}
