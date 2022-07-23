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

	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
)

var servicesUnPauseCmd = &cobra.Command{
	Use:   "unpause",
	Short: "Un-pause a service in your environment.",
	Long:  "Call this command to un-pause a service. This will make a service active and accessible.",
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
			err := utils.CheckProdOk(cmd, environmentType, "yes")
			if err != nil {
				return err
			}
		}

		return setServicePause(environmentType, serviceName, false)
	},
}

func init() {
	servicesCmd.AddCommand(servicesUnPauseCmd)

	servicesUnPauseCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	servicesUnPauseCmd.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")
	servicesUnPauseCmd.Flags().StringP("service", "s", "", "set the service name, if not provided will pull from nucleus.yaml (if there is one)")
}
