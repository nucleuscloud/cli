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
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/auth"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/secrets"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set <secret-name>",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("must provide secret-name to set secret")
		}

		// todo: probably need to validate this key
		secretKey := args[0]

		deployConfig, err := config.GetNucleusConfig()
		if err != nil {
			return err
		}

		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if utils.IsValidEnvironmentType(environmentType) {
			return errors.New("invalid value for environment")
		}

		if environmentType == "prod" {
			err := utils.CheckProdOk(cmd, environmentType, "yes")
			if err != nil {
				return err
			}
		}

		authClient, err := auth.NewAuthClient(utils.Auth0BaseUrl, utils.Auth0ClientId, utils.ApiAudience)
		if err != nil {
			return err
		}
		unAuthConn, err := utils.NewAnonymousConnection()
		if err != nil {
			return err
		}
		unAuthCliClient := pb.NewCliServiceClient(unAuthConn)
		accessToken, err := config.GetValidAccessTokenFromConfig(authClient, unAuthCliClient)
		unAuthConn.Close()
		if err != nil {
			return err
		}

		conn, err := utils.NewAuthenticatedConnection(accessToken)
		if err != nil {
			return err
		}

		defer conn.Close()

		nucleusClient := pb.NewCliServiceClient(conn)

		if verbose {
			log.Println("Attempting to retrieve public key for encrypting secrets...")
		}

		// todo: cache this key
		publicKeyReply, err := nucleusClient.GetPublicSecretKey(context.Background(), &pb.GetPublicSecretKeyRequest{
			EnvironmentType: environmentType,
			ServiceName:     deployConfig.Spec.ServiceName,
		}, utils.GetGrpcTrailer())
		if err != nil {
			return err
		}
		if verbose {
			log.Println("Retrieved public key!")
		}

		secret, err := getSecretValue()
		if err != nil {
			return err
		}
		if verbose {
			log.Println("Encrypting secret...")
		}
		err = secrets.StoreSecret(&deployConfig.Spec, publicKeyReply.PublicKey, secretKey, secret, environmentType)
		if err != nil {
			return err
		}

		if verbose {
			log.Println("Secret successfully encrypted!")
		}
		return config.SetNucleusConfig(deployConfig)
	},
}

func getSecretValue() (string, error) {
	isPiped, err := isPipedInput()

	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(os.Stdin)

	var secret string

	if !isPiped {
		fmt.Println("Enter secret followed by [Enter]:")
		fmt.Print("> ")
	}

	secret, err = reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	trimmedSecret := strings.TrimSpace(secret)

	if len(trimmedSecret) == 0 {
		return "", errors.New("must provide value to set secret")
	}

	return trimmedSecret, nil
}

func isPipedInput() (bool, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false, err
	}

	return (fi.Mode() & os.ModeCharDevice) == 0, nil
}

func init() {
	secretCmd.AddCommand(setCmd)

	setCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	setCmd.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// setCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

	// setCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
