package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// initCmd represents the init command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "creates a service given a service typecreates a project folder structure",
	Long:  `Creates an environment for the given service and then creates a folder structure depending on the type of service passed in the <serviceType flag>.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		environmentName, err := cmd.Flags().GetString(environmentFlag[0])
		if err != nil {
			return err
		} else if environmentName == "" {
			return errors.New("environment name is not provided")
		}

		serviceName, err := cmd.Flags().GetString(serviceNameFlag[0])
		if err != nil {
			return err
		} else if serviceName == "" {
			return errors.New("service name not provided")
		}

		serviceType, err := cmd.Flags().GetString(serviceTypeFlag[0])
		if err != nil {
			return err
		} else if serviceType == "" {
			return errors.New("service type not provided")
		}

		conn, err := newConnection()
		if err != nil {
			return err
		}

		defer conn.Close()

		client := pb.NewCliServiceClient(conn)
		// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
		var trailer metadata.MD
		reply, err := client.Init(context.Background(), &pb.InitRequest{
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

		workingDir, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("this is the working directory", workingDir)

		if serviceType == "fastapi" {

			//os.mkdir commands - https://stackoverflow.com/questions/14249467/os-mkdir-and-os-mkdirall-permissions
			//creates an empty folder with the name of the service and read, write, excute priv to the owner, but only read nd execute to everyone else

			dirName := serviceName + "_fastAPI"
			os.Mkdir(dirName, 0755)

			os.Chdir(serviceName)

			//TODO: check to see if fastAPI already exists in the system
			//if not, install fastAPI into the working directory

			mainFile, err := os.Create("sample_python_file.py")
			if err != nil {
				fmt.Println(err)
			}

			mainFile.WriteString("from typing import Optional\n\nfrom fastapi import FastAPI\n\napp = fastapi()\n\n@app.get(\"/\")\ndef read_root():\n\treturn {\"Hello\":\"World\"}\n\n")
			if err != nil {
				fmt.Println(err)
				mainFile.Close()
			}

			reqtFile, err := os.Create("requirements.txt")
			if err != nil {
				fmt.Println("Error:", err)
			}

			reqtFile.WriteString("#Add project requirements below as a new line\n\npip install fastapi\n\n")
			if err != nil {
				fmt.Println(err)
				reqtFile.Close()
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	stringP(createCmd, environmentFlag)
	stringP(createCmd, serviceNameFlag)
	stringP(createCmd, serviceTypeFlag)
}
