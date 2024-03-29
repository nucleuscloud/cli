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
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nucleuscloud/cli/internal/config"
	clienv "github.com/nucleuscloud/cli/internal/env"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
)

var servicesDependenciesAllowCmd = &cobra.Command{
	Use: "allow",
	Aliases: []string{
		"a",
	},
	Short: "Add a service dependency to the service's allow list.",
	Long:  "Call this command to add a service dependency to this service in order to authorize inter-service communication",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		environmentName, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if environmentName == "" {
			return fmt.Errorf("must provide environment name")
		}

		// Set this after ensuring flags are correct
		cmd.SilenceUsage = true

		servList, err := getServiceNamesByEnvironment(ctx, environmentName)
		if err != nil {
			return err
		}
		nucleusConfig, err := config.GetNucleusConfig()
		if err != nil {
			return err
		}

		serviceExcludeSet := map[string]struct{}{
			nucleusConfig.Spec.ServiceName: {},
		}
		for _, svc := range nucleusConfig.Spec.AllowedServices {
			serviceExcludeSet[svc] = struct{}{}
		}

		filteredServices := []string{}
		for _, svc := range servList {
			if _, ok := serviceExcludeSet[svc]; !ok {
				filteredServices = append(filteredServices, svc)
			}
		}

		var depName string
		err = survey.AskOne(&survey.Select{
			Message: "Select the service dependency",
			Options: filteredServices,
		}, &depName, surveyIcons)
		if err != nil {
			return err
		}

		return storeServiceDependency(depName)
	},
}

func init() {
	servicesDependenciesCmd.AddCommand(servicesDependenciesAllowCmd)

	servicesDependenciesAllowCmd.Flags().StringP("env", "e", "", "set the nucleus environment")
}

func getServiceNamesByEnvironment(ctx context.Context, environmentName string) ([]string, error) {
	conn, err := utils.NewApiConnectionByEnv(ctx, clienv.GetEnv())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	servicesResp, err := cliClient.GetServices(ctx, &svcmgmtv1alpha1.GetServicesRequest{
		EnvironmentName: strings.TrimSpace(environmentName),
	})
	if err != nil {
		return nil, err
	}

	serviceNames := []string{}
	for _, service := range servicesResp.Services {
		if service.ServiceCustomerConfig.ServiceName != "" {
			serviceNames = append(serviceNames, service.ServiceCustomerConfig.ServiceName)
		}
	}

	sort.Slice(serviceNames, func(i, j int) bool {
		return serviceNames[i] < serviceNames[j]
	})

	return serviceNames, nil
}

func storeServiceDependency(val string) error {
	nucleusConfig, err := config.GetNucleusConfig()
	if err != nil {
		return err
	}

	for _, allowedSvc := range nucleusConfig.Spec.AllowedServices {
		if allowedSvc == val {
			fmt.Println("This service is already in the allowed services list")
			return nil
		}
	}

	nucleusConfig.Spec.AllowedServices = append(nucleusConfig.Spec.AllowedServices, val)
	return config.SetNucleusConfig(nucleusConfig)
}
