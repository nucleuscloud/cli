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
	"errors"
	"fmt"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var servicesListCmd = &cobra.Command{
	Use: "list",
	Aliases: []string{
		"ls",
	},
	Short: "List out available services in your environment.",
	Long:  "Call this command to list out the available services for a specific environment type",
	RunE: func(cmd *cobra.Command, args []string) error {
		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if utils.IsValidEnvironmentType(environmentType) {
			return errors.New("invalid value for environment")
		}

		return listServices(environmentType)
	},
}

func init() {
	servicesCmd.AddCommand(servicesListCmd)

	servicesListCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
}

func listServices(environmentType string) error {
	conn, err := utils.NewApiConnection(utils.ApiConnectionConfig{
		AuthBaseUrl:  utils.Auth0BaseUrl,
		AuthClientId: utils.Auth0ClientId,
		ApiAudience:  utils.ApiAudience,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := pb.NewCliServiceClient(conn)
	var trailer metadata.MD
	serviceList, err := cliClient.ListServices(context.Background(), &pb.ListServicesRequest{
		EnvironmentType: strings.TrimSpace(environmentType),
	}, grpc.Trailer(&trailer))
	if err != nil {
		return err
	}

	fmt.Printf("Services in environment: %s\n", environmentType)
	for _, svcName := range serviceList.ServiceNames {
		fmt.Printf("* %s\n", svcName)
	}
	return nil
}
