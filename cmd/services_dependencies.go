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

var servicesDependenciesCmd = &cobra.Command{
	Use: "dependency",
	Aliases: []string{
		"deps",
	},
	Short: "Add a service dependency.",
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

		servList, err := listRawServices(ctx, environmentName)
		if err != nil {
			return err
		}

		serviceQuestions := []*survey.Question{
			{
				Name: "serviceType",
				Prompt: &survey.Select{
					Message: "Select the service dependency: ",
					Options: servList,
				},
				Validate: survey.Required,
			},
		}

		// ask the question
		var serviceDeps string
		err = survey.Ask(serviceQuestions, &serviceDeps, surveyIcons)
		if err != nil {
			return err
		}

		err = storeServiceDependency(serviceDeps)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	servicesCmd.AddCommand(servicesDependenciesCmd)

	servicesDependenciesCmd.Flags().StringP("env", "e", "", "set the nucleus environment")
}

func listRawServices(ctx context.Context, environmentName string) ([]string, error) {
	conn, err := utils.NewApiConnectionByEnv(ctx, clienv.GetEnv())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	serviceList, err := cliClient.GetServices(ctx, &svcmgmtv1alpha1.GetServicesRequest{
		EnvironmentName: strings.TrimSpace(environmentName),
	})
	if err != nil {
		return nil, err
	}

	var simpleServiceList []string //just the service names

	for _, k := range serviceList.Services {
		simpleServiceList = append(simpleServiceList, k.ServiceCustomerConfig.ServiceName)
	}

	sort.Slice(simpleServiceList, func(i, j int) bool {
		return simpleServiceList[i] < simpleServiceList[j]
	})

	return simpleServiceList, nil
}

func storeServiceDependency(val string) error {
	nucleusConfig, err := config.GetNucleusConfig()
	if err != nil {
		return err
	}

	var allowedServices []string

	allowedServices = nucleusConfig.Spec.AllowedServices

	allowedServices = append(allowedServices, val)

	nucleusConfig.Spec.AllowedServices = allowedServices

	err = config.SetNucleusConfig(nucleusConfig)

	if err != nil {
		return err
	}
	return nil
}
