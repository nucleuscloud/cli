package cmd

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/pkg/auth"
	"github.com/nucleuscloud/cli/pkg/config"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var listServicesCommand = &cobra.Command{
	Use:   "listServices",
	Short: "Lists all services in a given namespace.",
	Long:  `Lists all services in a given namespace`,

	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.GetNucleusConfig()
		if err != nil {
			return err
		}

		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		if isValidEnvironmentType(environmentType) {
			return errors.New("invalid value for environment")
		}

		if environmentType == "prod" {
			err := checkProdOk(cmd, environmentType, "yes")
			if err != nil {
				return err
			}
		}

		return listServices(environmentType)
	},
}

func listServices(environmentType string) error {
	authClient, err := auth.NewAuthClient(auth0BaseUrl, auth0ClientId, apiAudience)
	if err != nil {
		return err
	}
	unAuthConn, err := newConnection()
	if err != nil {
		return err
	}
	unAuthCliClient := pb.NewCliServiceClient(unAuthConn)
	accessToken, err := config.GetValidAccessTokenFromConfig(authClient, unAuthCliClient)
	unAuthConn.Close()
	if err != nil {
		return err
	}

	conn, err := newAuthenticatedConnection(accessToken)
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

	log.Printf("services in %s:", environmentType)
	for _, svcName := range serviceList.ServiceNames {
		log.Printf("%s", svcName)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(listServicesCommand)

	listServicesCommand.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	listServicesCommand.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")

}
