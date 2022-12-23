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

	"github.com/fatih/color"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

type EnvironmentConfig struct {
	EnvironmentId        string
	EnvironmentNamespace string
	EnvironmentName      string
	EnvironmentRegion    string
	EnvironmentCluster   string
	EnvironmentProvider  string
	ServiceCount         int32
	ClusterConfigId      string
}

var environmentsListCmd = &cobra.Command{
	Use: "list",
	Aliases: []string{
		"ls",
	},
	Short: "List out available environments in your account.",
	Long:  "Call this command to list out all of the environments in your account",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		return listEnvironments(ctx)
	},
}

func init() {
	environmentsCmd.AddCommand(environmentsListCmd)
}

func listEnvironments(ctx context.Context) error {
	conn, err := utils.NewApiConnectionByEnv(ctx, utils.GetEnv())
	if err != nil {
		return err
	}
	defer conn.Close()

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Name", "Region", "Cluster", "Services")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	envList, err := cliClient.GetEnvironments(ctx, &svcmgmtv1alpha1.GetEnvironmentsRequest{})
	if err != nil {
		return err
	}

	clusterConfigs, err := cliClient.GetProviderClusterConfigs(ctx, &svcmgmtv1alpha1.GetProviderClusterConfigsRequest{})
	if err != nil {
		return err
	}

	var envConfig []*EnvironmentConfig

	for _, configs := range clusterConfigs.ClusterConfigs {
		envs, err := cliClient.GetEnvironmentsByProviderClusterId(ctx, &svcmgmtv1alpha1.GetEnvironmentsByProviderClusterIdRequest{ProviderClusterConfigId: configs.Id})
		if err != nil {
			return err
		}

		for _, envs := range envs.Environments {
			count, err := getServicesCount(ctx, envs.EnvironmentName, cliClient)
			if err != nil {
				return err
			}
			envConfig = append(envConfig, &EnvironmentConfig{
				EnvironmentId:        envs.EnvironmentId,
				EnvironmentName:      envs.EnvironmentName,
				EnvironmentRegion:    configs.ProviderRegionName,
				EnvironmentProvider:  "aws",
				EnvironmentCluster:   configs.ClusterName,
				EnvironmentNamespace: envs.EnvironmentNamespace,
				ServiceCount:         count,
				ClusterConfigId:      configs.Id,
			})
		}
	}

	for _, env := range envList.Environments {

		tbl.AddRow(
			env.EnvironmentName,
			getRegion(env.EnvironmentName, envConfig),
			getClusterName(env.EnvironmentName, envConfig),
			filterEnvforServicesCount(env.EnvironmentName, envConfig),
		)
	}
	tbl.Print()
	return nil
}

func getRegion(envName string, config []*EnvironmentConfig) string {
	for _, envConfig := range config {
		if envConfig.EnvironmentName == envName {
			return envConfig.EnvironmentRegion
		}
	}
	return ""
}

func getClusterName(envName string, config []*EnvironmentConfig) string {
	for _, envConfig := range config {
		if envConfig.EnvironmentName == envName {
			return envConfig.EnvironmentCluster
		}
	}
	return ""
}

func filterEnvforServicesCount(envName string, config []*EnvironmentConfig) int32 {
	for _, envConfig := range config {
		if envConfig.EnvironmentName == envName {
			return envConfig.ServiceCount
		}
	}
	return 0
}

func getServicesCount(ctx context.Context, envName string, client svcmgmtv1alpha1.ServiceMgmtServiceClient) (int32, error) {

	svcs, err := client.GetServices(ctx, &svcmgmtv1alpha1.GetServicesRequest{EnvironmentType: envName})
	if err != nil {
		return 0, err
	}
	return int32(len(svcs.Services)), nil
}
