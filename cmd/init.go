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

	"github.com/mhelmich/haiku-api/pkg/api/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")

		projectName, err := cmd.Flags().GetString("project-name")

		if err != nil {
			panic(err)
		}

		if projectName == "" {
			panic(errors.New("project name not provided"))
		}

		conn, err := grpc.Dial("2.tcp.ngrok.io:10459", grpc.WithInsecure())

		if err != nil {
			panic(err)
		}

		defer conn.Close()

		client := pb.NewCliServiceClient(conn)
		reply, err := client.Init(context.Background(), &pb.InitRequest{
			ProjectName: projectName,
		})

		if err != nil {
			panic(err)
		}

		fmt.Println(reply.ID)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// req := &pb.InitRequest{}

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	initCmd.Flags().StringP("project-name", "p", "", "-p my-project-name")
}
