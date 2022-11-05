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

	"github.com/nucleuscloud/cli/internal/config"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
)

var servicesStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "Stop a service in your environment.",
	Aliases: []string{"pause"},
	Long:    "Call this command to stop a service. This will shut it down and no longer make it accessible.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if environmentType == "" {
			return fmt.Errorf("must provide environment type")
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

		return setServicePause(ctx, environmentType, serviceName, true)
	},
}

func init() {
	servicesCmd.AddCommand(servicesStopCmd)

	servicesStopCmd.Flags().StringP("env", "e", "", "set the nucleus environment")
	servicesStopCmd.Flags().StringP("service", "s", "", "set the service name, if not provided will pull from nucleus.yaml (if there is one)")
}

func setServicePause(ctx context.Context, environmentType string, serviceName string, isPaused bool) error {
	conn, err := utils.NewApiConnectionByEnv(ctx, utils.GetEnv())
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	_, err = cliClient.SetServiceActiveStatus(ctx, &svcmgmtv1alpha1.SetServiceActiveStatusRequest{
		EnvironmentType: strings.TrimSpace(environmentType),
		ServiceName:     serviceName,
		IsActive:        !isPaused,
	})
	if err != nil {
		return err
	}
	return nil
}
