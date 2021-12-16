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

// dockerCmd represents the docker command
var dockerLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := cmd.Flags().GetString("server")

		if err != nil {
			return err
		}

		if server == "" {
			return errors.New("server url not provided")
		}

		username, err := cmd.Flags().GetString("username")

		if err != nil {
			return err
		}

		if username == "" {
			return errors.New("username not provided")
		}

		password, err := cmd.Flags().GetString("password")

		if err != nil {
			return err
		}

		if password == "" {
			return errors.New("password not provided")
		}

		email, err := cmd.Flags().GetString("email")

		if err != nil {
			return err
		}

		if email == "" {
			return errors.New("email not provided")
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
		reply, err := client.DockerLogin(context.Background(), &pb.DockerLoginRequest{
			Server:   server,
			Username: username,
			Password: password,
			Email:    email,
		},
			grpc.Trailer(&trailer),
		)

		if err != nil {
			return err
		}

		fmt.Println(reply.ID)
		if len(trailer["x-request-id"]) == 1 {
			fmt.Println(trailer["x-request-id"][0])
		}
		return nil
	},
}

func init() {
	dockerCmd.AddCommand(dockerLoginCmd)

	dockerCmd.Flags().String("server", "", "Server url to the docker registry")
	dockerCmd.Flags().StringP("username", "u", "", "docker registry username")
	dockerCmd.Flags().StringP("password", "p", "", "docker registry password")
	dockerCmd.Flags().StringP("email", "e", "", "docker registry password")
}
