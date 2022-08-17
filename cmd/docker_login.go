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

	"github.com/AlecAivazis/survey/v2"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
)

var dockerLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Set docker login credentials for your service environment.",
	Long:  "If your service uses a docker image that resides in a private registry, use this command to let nucleus pull the image.",
	RunE: func(cmd *cobra.Command, args []string) error {
		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if !utils.IsValidEnvironmentType(environmentType) {
			return fmt.Errorf("invalid value for environment")
		}

		username, err := cmd.Flags().GetString("username")
		if err != nil {
			return err
		}
		if username == "" {
			return fmt.Errorf("must provide valid username")
		}

		server, err := cmd.Flags().GetString("server")
		if err != nil {
			return err
		}
		if server == "" {
			return fmt.Errorf("must provide valid server")
		}

		email, err := cmd.Flags().GetString("email")
		if err != nil {
			return err
		}

		passwordResult, err := getPassword()
		if err != nil {
			return err
		}

		if environmentType == "prod" {
			if passwordResult.isPiped {
				yesPrompt, err := cmd.Flags().GetBool("yes")
				if err != nil {
					return err
				}
				if !yesPrompt {
					return fmt.Errorf("must provide -y when piping in password to production environment")
				}
			} else {
				err := utils.PromptToProceed(cmd, environmentType, "yes")
				if err != nil {
					return err
				}
			}
		}

		return dockerLogin(environmentType, server, username, passwordResult.value, email)
	},
}

func init() {
	dockerCmd.AddCommand(dockerLoginCmd)

	dockerLoginCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	dockerLoginCmd.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")
	dockerLoginCmd.Flags().StringP("username", "u", "", "registry username")
	dockerLoginCmd.Flags().StringP("server", "s", "", "registry server")
	dockerLoginCmd.Flags().String("email", "", "registry email")
}

func dockerLogin(environmentType, server, username, password, email string) error {
	conn, err := utils.NewApiConnectionByEnv(utils.GetEnv(), onPrem)
	if err != nil {
		return err
	}
	defer conn.Close()

	if onPrem {
		cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
		_, err = cliClient.SetDockerLogin(context.Background(), &svcmgmtv1alpha1.SetDockerLoginRequest{
			EnvironmentType: environmentType,
			Server:          server,
			Email:           email,
			Username:        username,
			Password:        password,
		})
		if err != nil {
			return err
		}
		return nil
	}

	cliClient := pb.NewCliServiceClient(conn)
	_, err = cliClient.DockerLogin(context.Background(), &pb.DockerLoginRequest{
		EnvType:  environmentType,
		Server:   server,
		Email:    email,
		Username: username,
		Password: password,
	}, utils.GetGrpcTrailer())
	if err != nil {
		return err
	}
	return nil
}

type PasswordResult struct {
	value   string
	isPiped bool
}

func getPassword() (*PasswordResult, error) {
	passwordValue := ""

	piped, err := isPipedInput()
	if err != nil {
		return nil, err
	}

	if piped {
		_, err = fmt.Scanf("%s", &passwordValue)
		if err != nil {
			return nil, err
		}
		if passwordValue == "" {
			return nil, fmt.Errorf("password must have length greater than 0")
		}
		return &PasswordResult{
			value:   passwordValue,
			isPiped: true,
		}, nil
	}
	err = survey.AskOne(&survey.Password{
		Message: "Enter password followed by [Enter]:",
	}, &passwordValue)
	if err != nil {
		return nil, err
	}
	if passwordValue == "" {
		return nil, fmt.Errorf("password must have length greater than 0")
	}
	return &PasswordResult{
		value:   passwordValue,
		isPiped: false,
	}, nil
}
