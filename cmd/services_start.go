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

	"github.com/nucleuscloud/cli/internal/config"
	"github.com/nucleuscloud/cli/internal/utils"
	"github.com/spf13/cobra"
)

var servicesStartCmd = &cobra.Command{
	Use:     "start",
	Short:   "start a service in your environment.",
	Aliases: []string{"unpause"},
	Long:    "Call this command to start a service. This will make a service active and accessible.",
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

		return setServicePause(ctx, environmentType, serviceName, false)
	},
}

func init() {
	servicesCmd.AddCommand(servicesStartCmd)

	servicesStartCmd.Flags().StringP("env", "e", "", "set the nucleus environment")
	servicesStartCmd.Flags().StringP("service", "s", "", "set the service name, if not provided will pull from nucleus.yaml (if there is one)")
}
