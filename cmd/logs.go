package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/pkg/auth"
	"github.com/nucleuscloud/cli/pkg/config"
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

		if isValidEnvironmentType(environmentType) {
			return errors.New("invalid value for environment")
		}

		serviceName := strings.TrimSpace(sn)
		if !isValidName(serviceName) {
			return ErrInvalidName
		}

		return logs(environmentType, serviceName)
	},
}

func logs(environmentType string, serviceName string) error {
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
	logs, err := cliClient.Logs(context.Background(), &pb.LogsRequest{
		EnvironmentType: environmentType,
		ServiceName:     serviceName,
	},
		grpc.Trailer(&trailer),
	)
	if err != nil && err != io.EOF {
		return err
	}

	for i := 0; i < len(logs.Log); i++ {
		fmt.Println(string(logs.Log[i]))
	}
	return nil
}

func init() {
	rootCmd.AddCommand(logsCommand)
	logsCommand.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	logsCommand.Flags().StringP("tail", "t", "", "live log tail")
	logsCommand.Flags().StringP("service", "s", "", "service name")
}
