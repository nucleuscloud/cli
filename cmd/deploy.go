/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"log"

	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := cmd.Flags().GetString("project-name")
		if err != nil {
			return err
		}
		if projectName == "" {
			return errors.New("project-name not provided")
		}

		imageName, err := cmd.Flags().GetString("image")
		if err != nil {
			return err
		}
		if imageName == "" {
			return errors.New("image not provided")
		}

		serviceName, err := cmd.Flags().GetString("service-name")
		if err != nil {
			return err
		}
		if serviceName == "" {
			return errors.New("service name not provided")
		}

		creds, err := credentials.NewClientTLSFromFile("service.pem", "")
		if err != nil {
			log.Fatalf("could not process the credentials: %v", err)
		}

		conn, err := grpc.Dial("127.0.0.1:50051", grpc.WithTransportCredentials(creds))

		if err != nil {
			return err
		}

		defer conn.Close()

		client := pb.NewCliServiceClient(conn)
		// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
		var trailer metadata.MD
		reply, err := client.Deploy(context.Background(), &pb.DeployRequest{
			ProjectName: projectName,
			Image:       imageName,
			ServiceName: serviceName,
		},
			grpc.Trailer(&trailer),
		)

		if err != nil {
			return err
		}

		fmt.Printf("k8s id: %s\n", reply.ID)
		if len(trailer["x-request-id"]) == 1 {
			fmt.Printf("request id: %s\n", trailer["x-request-id"][0])
		}

		fmt.Printf("service url: %s\n", reply.URL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringP("project-name", "p", "", "-p my-project-name")
	deployCmd.Flags().StringP("image", "i", "", "-i https://example.com/link-to-docker-image:latest")
	deployCmd.Flags().StringP("service-name", "s", "", "-s my-service")
	deployCmd.Flags().StringP("env", "e", "", "-e dev")
}
