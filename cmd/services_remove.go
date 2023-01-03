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
	"os"
	"strings"

	"github.com/nucleuscloud/cli/internal/config"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
)

var servicesRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "Remove a service from your environment.",
	Long:    "Completely remove a service from your environment. This operation is destructive!",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		environmentName, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if environmentName == "" {
			return fmt.Errorf("must provide environment name")
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
					fmt.Fprintln(os.Stderr, fmt.Errorf("Did not provide service name and could not find nucleus config"))
					return err
				}
				serviceName = cfg.Spec.ServiceName
			}
		}

		if !utils.IsValidName(serviceName) {
			return utils.ErrInvalidServiceName
		}

		fmt.Printf("Service to delete: \n↪Environment: %s\n↪Service: %s\n", environmentName, serviceName)

		err = utils.PromptToProceed(cmd, environmentName, "yes")
		if err != nil {
			return err
		}

		return removeService(ctx, environmentName, serviceName)
	},
}

func init() {
	servicesCmd.AddCommand(servicesRemoveCmd)

	servicesRemoveCmd.Flags().StringP("env", "e", "", "set the nucleus environment")
	servicesRemoveCmd.Flags().StringP("service", "s", "", "set the service name, if not provided will pull from nucleus.yaml (if there is one)")
	servicesRemoveCmd.Flags().BoolP("yes", "y", false, "automatically proceed with removal")
}

func removeService(ctx context.Context, environmentName string, serviceName string) error {
	conn, err := utils.NewApiConnectionByEnv(ctx, utils.GetEnv())
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	_, err = cliClient.RemoveService(ctx, &svcmgmtv1alpha1.RemoveServiceRequest{
		EnvironmentName: strings.TrimSpace(environmentName),
		ServiceName:     serviceName,
	})
	if err != nil {
		return err
	}
	return nil
}
