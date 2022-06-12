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
	"io/ioutil"
	"os"
	"strings"

	"github.com/mhelmich/keycloak"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/pkg/auth"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

		deployConfig, err := getNucleusConfig()
		if err != nil {
			return err
		}

		err = upsertNucleusSecrets()
		if err != nil {
			return err
		}

		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if environmentType != "dev" && environmentType != "stage" && environmentType != "prod" {
			return errors.New("invalid value for environment")
		}

		if environmentType == "prod" {
			yesPrompt, err := cmd.Flags().GetBool("yes")
			if err != nil {
				return err
			}
			if !yesPrompt {
				shouldDeploy := cliPrompt("are you sure you want to deploy to production? (y/n)", "n")
				if shouldDeploy != "y" {
					return errors.New("exiting as received non yes answer for production deploy")
				}
			}
		}

		authClient, err := auth.NewAuthClient(auth0BaseUrl, auth0ClientId, auth0ClientSecret, apiAudience)
		if err != nil {
			return err
		}

		conn, err := newAuthenticatedConnection(authClient)
		if err != nil {
			return err
		}

		defer conn.Close()

		nucleusClient := pb.NewCliServiceClient(conn)

		fmt.Println("Retrieving public key...")

		// todo: cache this key
		publicKeyReply, err := nucleusClient.GetPublicSecretKey(context.Background(), &pb.GetPublicSecretKeyRequest{
			EnvironmentType: environmentType,
			ServiceName:     deployConfig.Spec.ServiceName,
		}, getGrpcTrailer())
		if err != nil {
			return err
		}

		fmt.Println("Retrieved public key!")

		publicKey := string(publicKeyReply.PublicKey)

		secret, err := getSecretValue()
		if err != nil {
			return err
		}

		// todo: maybe make this configurable
		fmt.Println("Encrypting secret...")
		err = storeSecret("./nucleus-secrets.yaml", publicKey, secretKey, secret)
		if err != nil {
			return err
		}

		fmt.Println("Secret encrypted successfully!")
		return nil
	},
}

type NucleusSecrets struct {
	Secrets map[string]string `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

func storeSecret(fileName string, publicKey string, secretKey string, secretValue string) error {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	root := NucleusSecrets{}
	err = yaml.Unmarshal(file, &root)

	if err != nil {
		return err
	}

	if root.Secrets == nil {
		root.Secrets = make(map[string]string)
	}

	root.Secrets[secretKey] = secretValue

	newBlob, err := yaml.Marshal(root)
	if err != nil {
		return err
	}

	store, err := keycloak.GetStoreFromBytes(newBlob, keycloak.YAML)
	if err != nil {
		return err
	}

	err = store.EncryptSubtree(publicKey, "secrets")
	if err != nil {
		return err
	}

	err = store.ToFile(fileName)
	if err != nil {
		return err
	}

	return nil
}

func getSecretValue() (string, error) {
	isPiped, err := isPipedInput()

	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(os.Stdin)

	var secret string

	if !isPiped {
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
