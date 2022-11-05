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

	"github.com/fatih/color"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

var servicesListCmd = &cobra.Command{
	Use: "list",
	Aliases: []string{
		"ls",
	},
	Short: "List out available services in your environment.",
	Long:  "Call this command to list out the available services for a specific environment type",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		return listServices(ctx, environmentType)
	},
}

func init() {
	servicesCmd.AddCommand(servicesListCmd)

	servicesListCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
}

func listServices(ctx context.Context, environmentType string) error {
	conn, err := utils.NewApiConnectionByEnv(ctx, utils.GetEnv())
	if err != nil {
		return err
	}
	defer conn.Close()

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("Name", "Status", "Visibility", "Url")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	serviceList, err := cliClient.GetServices(ctx, &svcmgmtv1alpha1.GetServicesRequest{
		EnvironmentType: strings.TrimSpace(environmentType),
	})
	if err != nil {
		return err
	}
	fmt.Printf("Services in environment: %s\n", environmentType)
	for svcName, svcInfo := range serviceList.Services {
		tbl.AddRow(
			svcName,
			getIsActiveLabel(svcInfo.IsActive),
			getVisibilityLabel(svcInfo.IsPrivate),
			getUrlLabel(svcInfo.IsPrivate, svcInfo.Url),
		)
	}
	tbl.Print()
	return nil
}

func getIsActiveLabel(isActive bool) string {
	if isActive {
		return "Active"
	}
	return "Inactive"
}

func getVisibilityLabel(isPrivate bool) string {
	if isPrivate {
		return "Private"
	}
	return "Public"
}

func getUrlLabel(isPrivate bool, url string) string {
	if isPrivate {
		return ""
	}
	return url
}
