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
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var servicesPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause a service in your environment.",
	Long:  "Call this command to pause a service. This will shut it down and no longer make it accessible.",
	RunE: func(cmd *cobra.Command, args []string) error {
		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if utils.IsValidEnvironmentType(environmentType) {
			return fmt.Errorf("invalid value for environment")
		}

		serviceName, err := cmd.Flags().GetString("service")
		if err != nil {
			return err
		}
		serviceName = strings.TrimSpace(serviceName)
		if serviceName == "" {
			if config.DoesNucleusConfigExist() {
				cfg, err := config.GetNucleusConfig()
				if err != nil {
					fmt.Println("Did not provide service name and could not find nucleus config")
					return err
				}
				serviceName = cfg.Spec.ServiceName
			}
		}

		if !utils.IsValidName(serviceName) {
			return utils.ErrInvalidServiceName
		}

		if environmentType == "prod" {
			err := utils.PromptToProceed(cmd, environmentType, "yes")
			if err != nil {
				return err
			}
		}

		return setServicePause(environmentType, serviceName, true)
	},
}

func init() {
	servicesCmd.AddCommand(servicesPauseCmd)

	servicesPauseCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	servicesPauseCmd.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")
	servicesPauseCmd.Flags().StringP("service", "s", "", "set the service name, if not provided will pull from nucleus.yaml (if there is one)")
}

func setServicePause(environmentType string, serviceName string, isPaused bool) error {
	conn, err := utils.NewApiConnectionByEnv(utils.GetEnv())
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := pb.NewCliServiceClient(conn)
	var trailer metadata.MD
	_, err = cliClient.SetServicePauseStatus(context.Background(), &pb.SetServicePauseStatusRequest{
		EnvironmentType: strings.TrimSpace(environmentType),
		ServiceName:     serviceName,
		IsPaused:        isPaused,
	}, grpc.Trailer(&trailer))
	if err != nil {
		return err
	}
	return nil
}
