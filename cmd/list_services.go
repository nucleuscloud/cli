package cmd

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var listServicesCommand = &cobra.Command{
	Use:   "listServices",
	Short: "",
	Long:  ``,

	RunE: func(cmd *cobra.Command, args []string) error {
		environmentName, err := cmd.Flags().GetString(environmentFlag[0])
		if err != nil {
			return err
		}
		if environmentName == "" {
			return errors.New("environment name not provided")
		}

		environmentName = strings.TrimSpace(environmentName)
		if !isValidName(environmentName) {
			return ErrInvalidName
		}

		return listServices(environmentName)
	},
}

func listServices(environmentName string) error {
	conn, err := newConnection()
	if err != nil {
		return err
	}

	defer conn.Close()
	cliClient := pb.NewCliServiceClient(conn)
	var trailer metadata.MD
	serviceList, err := cliClient.ListServices(context.Background(), &pb.ListServicesRequest{
		EnvironmentName: strings.TrimSpace(environmentName),
	}, grpc.Trailer(&trailer))
	if err != nil {
		return err
	}

	log.Printf("services in %s:", environmentName)
	for _, svcName := range serviceList.ServiceNames {
		log.Printf("%s", svcName)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(listServicesCommand)
	stringP(listServicesCommand, environmentFlag)
}
