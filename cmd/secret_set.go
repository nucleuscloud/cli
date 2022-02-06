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
	"os"
	"strings"

	"github.com/haikuapp/api/pkg/api/v1/pb"
	"github.com/mhelmich/keycloak"
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

		secretKey := args[0]

		conn, err := newConnection()
		if err != nil {
			return err
		}

		defer conn.Close()

		haikuClient := pb.NewCliServiceClient(conn)

		deployConfig, err := getHaikuConfig()
		if err != nil {
			return err
		}

		fmt.Println("Attempting to retrieve public key")

		publicKeyReply, err := haikuClient.GetPublicSecretKey(context.Background(), &pb.GetPublicSecretKeyRequest{
			EnvironmentName: deployConfig.Spec.EnvironmentName,
			ServiceName:     deployConfig.Spec.ServiceName,
		}, getGrpcTrailer())
		if err != nil {
			return err
		}

		publicKey := string(publicKeyReply.PublicKey)

		secret, err := getSecretValue()
		if err != nil {
			return err
		}

		fmt.Println(secret)

		// todo: maybe make this configurable
		err = storeSecret("./haiku-secrets.yaml", publicKey, secretKey, secret)
		if err != nil {
			return err
		}

		fmt.Println("Secret successfully encrypted")
		return nil
	},
}

func storeSecret(fileName string, publicKey string, secretKey string, secretValue string) error {
	store, err := keycloak.GetStoreForFile(fileName)
	if err != nil {
		return err
	}

	err = store.EncryptSubtree(publicKey, secretKey, secretValue)
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

	if isPiped {
		secret, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
	} else {
		fmt.Print("> ")
		secret, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// setCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

	// setCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
