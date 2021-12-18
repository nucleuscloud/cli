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
	"fmt"

	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
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
		getParamString(cmd, dockerServerFlag)
		server, err := getParamString(cmd, dockerServerFlag)
		if err != nil {
			return err
		}

		username, err := getParamString(cmd, dockerUsernameFlag)
		if err != nil {
			return err
		}

		password, err := getParamString(cmd, dockerPasswordFlag)
		if err != nil {
			return err
		}

		email, err := getParamString(cmd, dockerEmailFlag)
		if err != nil {
			return err
		}

		environmentName, err := getParamString(cmd, environmentFlag)
		if err != nil {
			return err
		}

		conn, err := newConnection()
		if err != nil {
			return err
		}

		defer conn.Close()
		client := pb.NewCliServiceClient(conn)
		// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
		var trailer metadata.MD
		reply, err := client.DockerLogin(context.Background(), &pb.DockerLoginRequest{
			Server:          server,
			Username:        username,
			Password:        password,
			Email:           email,
			EnvironmentName: environmentName,
		},
			grpc.Trailer(&trailer),
		)
		if err != nil {
			return err
		}

		fmt.Printf("k8s id: %s\n", reply.ID)
		if verbose {
			if len(trailer["x-request-id"]) == 1 {
				fmt.Printf("request id: %s\n", trailer["x-request-id"][0])
			}
		}
		return nil
	},
}

func init() {
	dockerCmd.AddCommand(dockerLoginCmd)
	stringP(dockerLoginCmd, dockerServerFlag)
	stringP(dockerLoginCmd, dockerUsernameFlag)
	stringP(dockerLoginCmd, dockerPasswordFlag)
	stringP(dockerLoginCmd, dockerEmailFlag)
	stringP(dockerLoginCmd, environmentFlag)
}
