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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/secrets"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set <secret-name>",
	Short: "Encrypts a secret and stores it for use in your nucleus manifest file.",
	Long:  "Encrypts a secret and stores it for use in your nucleus manifest file.",
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

		if !utils.IsValidName(deployConfig.Spec.ServiceName) {
			return utils.ErrInvalidServiceName
		}

		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if utils.IsValidEnvironmentType(environmentType) {
			return errors.New("invalid value for environment")
		}

		secretResult, err := getSecretValue()
		if err != nil {
			return err
		}

		if environmentType == "prod" {
			if secretResult.isPiped {
				yesPrompt, err := cmd.Flags().GetBool("yes")
				if err != nil {
					return err
				}
				if !yesPrompt {
					return fmt.Errorf("must provide -y when piping in secret value to production environment")
				}
			} else {
				err := utils.PromptToProceed(cmd, environmentType, "yes")
				if err != nil {
					return err
				}
			}
		}

		conn, err := utils.NewApiConnectionByEnv(utils.GetEnv(), onPrem)
		if err != nil {
			return err
		}
		defer conn.Close()

		cliClient := pb.NewCliServiceClient(conn)
		svcClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)

		if verbose {
			fmt.Println("Attempting to retrieve public key for encrypting secrets...")
		}

		// todo: cache this key
		var publicKey []byte
		if onPrem {
			publicKeyReply, err := svcClient.GetPublicSecretKey(context.Background(), &svcmgmtv1alpha1.GetPublicSecretKeyRequest{
				EnvironmentType: environmentType,
				ServiceName:     deployConfig.Spec.ServiceName,
			})
			if err != nil {
				return err
			}
			publicKey = publicKeyReply.PublicKey
		} else {
			publicKeyReply, err := cliClient.GetPublicSecretKey(context.Background(), &pb.GetPublicSecretKeyRequest{
				EnvironmentType: environmentType,
				ServiceName:     deployConfig.Spec.ServiceName,
			}, utils.GetGrpcTrailer())
			if err != nil {
				return err
			}
			publicKey = publicKeyReply.PublicKey
		}
		if verbose {
			fmt.Println("Retrieved public key!")
		}

		if verbose {
			fmt.Println("Encrypting secret...")
		}
		err = secrets.StoreSecret(&deployConfig.Spec, publicKey, secretKey, secretResult.value, environmentType)
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
