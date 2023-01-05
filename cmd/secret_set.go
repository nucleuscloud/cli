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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nucleuscloud/cli/internal/config"
	clienv "github.com/nucleuscloud/cli/internal/env"
	"github.com/nucleuscloud/cli/internal/secrets"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set <secret-name>",
	Short: "Encrypts a secret and stores it for use in your nucleus manifest file.",
	Long:  "Encrypts a secret and stores it for use in your nucleus manifest file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) == 0 {
			return errors.New("must provide secret-name to set secret")
		}

		// todo: probably need to validate this key
		secretKey := args[0]

		deployConfig, err := config.GetNucleusConfig()
		if err != nil {
			return err
		}

		if !utils.IsValidName(deployConfig.Spec.ServiceName) {
			return utils.ErrInvalidServiceName
		}

		environmentName, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if environmentName == "" {
			return fmt.Errorf("must provide environment name")
		}

		// Set this after ensuring flags are correct
		cmd.SilenceUsage = true

		secretResult, err := getSecretValue()
		if err != nil {
			return err
		}

		conn, err := utils.NewApiConnectionByEnv(ctx, clienv.GetEnv())
		if err != nil {
			return err
		}
		defer conn.Close()

		svcClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)

		if verbose {
			fmt.Println("Attempting to retrieve public key for encrypting secrets...")
		}

		publicKeyReply, err := svcClient.GetPublicSecretKey(ctx, &svcmgmtv1alpha1.GetPublicSecretKeyRequest{
			EnvironmentName: environmentName,
			ServiceName:     deployConfig.Spec.ServiceName,
		})
		if err != nil {
			return err
		}
		// todo: this key should be cached
		publicKey := publicKeyReply.PublicKey
		if verbose {
			fmt.Println("Retrieved public key!")
		}

		if verbose {
			fmt.Println("Encrypting secret...")
		}
		err = secrets.StoreSecret(&deployConfig.Spec, publicKey, secretKey, secretResult.value, environmentName)
		if err != nil {
			return err
		}

		if verbose {
			fmt.Println("Secret successfully encrypted!")
		}
		return config.SetNucleusConfig(deployConfig)
	},
}

type SecretResult struct {
	value   string
	isPiped bool
}

func getSecretValue() (*SecretResult, error) {
	secretValue := ""

	piped, err := isPipedInput()
	if err != nil {
		return nil, err
	}

	if piped {
		_, err = fmt.Scanf("%s", &secretValue)
		if err != nil {
			return nil, err
		}
		secretValue = strings.TrimSpace(secretValue)
		if secretValue == "" {
			return nil, fmt.Errorf("secret length must be greater than 0")
		}
		return &SecretResult{
			value:   secretValue,
			isPiped: true,
		}, nil
	}

	err = survey.AskOne(&survey.Input{
		Message: "Enter secret followed by [Enter]:",
	}, &secretValue)
	if err != nil {
		return nil, err
	}
	if secretValue == "" {
		return nil, fmt.Errorf("secret length must be greater than 0")
	}
	return &SecretResult{
		value:   secretValue,
		isPiped: false,
	}, nil
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

	setCmd.Flags().StringP("env", "e", "", "set the nucleus environment")
}
