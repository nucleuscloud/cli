package cmd

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/pkg/auth"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var listServicesCommand = &cobra.Command{
	Use:   "listServices",
	Short: "Lists all services in a given namespace.",
	Long:  `Lists all services in a given namespace`,

	RunE: func(cmd *cobra.Command, args []string) error {
		deployConfig, err := getNucleusConfig()
		if err != nil {
			return err
		}

		environmentName := deployConfig.Spec.EnvironmentName
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
	authClient, err := auth.NewAuthClient(auth0BaseUrl, auth0ClientId, auth0ClientSecret, apiAudience)
	if err != nil {
		return err
	}
	conn, err := newAuthenticatedConnection(authClient)
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
}
