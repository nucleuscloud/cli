package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var logsCommand = &cobra.Command{
	Use:   "logs",
	Short: "Returns logs for a given service.",
	Long:  `Returns logs for a given service.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		sn, err := cmd.Flags().GetString("service")
		if err != nil {
			return err
		}

		window, err := cmd.Flags().GetString("window")
		if err != nil {
			return err
		}

		if !allowedWindowValues(window) {
			return errors.New("invalid value for log window - should be one of [15min,1h,1d]")
		}

		if utils.IsValidEnvironmentType(environmentType) {
			return errors.New("invalid value for environment")
		}

		serviceName := strings.TrimSpace(sn)
		if !utils.IsValidName(serviceName) {
			return utils.ErrInvalidName
		}

		return logs(environmentType, serviceName, window)
	},
}

func logs(environmentType string, serviceName string, window string) error {
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
	logs, err := cliClient.Logs(context.Background(), &pb.LogsRequest{
		EnvironmentType: environmentType,
		ServiceName:     serviceName,
		Window:          window,
	},
		grpc.Trailer(&trailer),
	)
	if err != nil && err != io.EOF {
		return err
	}

	fmt.Println("\nGenerating logs for the last " + window + "...\n")

	if len(logs.Log) == 0 {
		fmt.Println("No logs for this time window. If you just deployed your service, try again in a minute or try a bigger time window.")
	}

	for i := 0; i < len(logs.Log); i++ {
		fmt.Println(string(logs.Log[i]))
	}
	return nil
}

func allowedWindowValues(window string) bool {
	switch window {
	case
		"15min",
		"1h",
		"1d":
		return true
	}
	return false
}

func init() {
	rootCmd.AddCommand(logsCommand)
	logsCommand.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	logsCommand.Flags().StringP("tail", "t", "", "live log tail")
	logsCommand.Flags().StringP("service", "s", "", "service name")
	logsCommand.Flags().StringP("window", "w", "", "logging window allowed values: [15min, 1h, 1d]")
}
