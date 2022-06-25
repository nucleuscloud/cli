package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
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

		if utils.IsValidEnvironmentType(environmentType) {
			return errors.New("invalid value for environment")
		}

		return listServices(environmentType)
	},
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

	fmt.Printf("services in %s:\n", environmentType)
	for _, svcName := range serviceList.ServiceNames {
		fmt.Printf("%s\n", svcName)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(listServicesCommand)

	listServicesCommand.Flags().StringP("env", "e", "prod", "set the nucleus environment")

}
