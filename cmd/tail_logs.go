package cmd

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/auth"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var tailLogsCommand = &cobra.Command{
	Use:   "tail",
	Short: "Tails logs for a given service.",
	Long:  `Tails logs for a given service.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		deployConfig, err := config.GetNucleusConfig()
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

		if environmentType == "prod" {
			err := utils.CheckProdOk(cmd, environmentType, "yes")
			if err != nil {
				return err
			}
		}

		serviceName := deployConfig.Spec.ServiceName
		if serviceName == "" {
			return errors.New("service name not provided")
		}

		serviceName = strings.TrimSpace(serviceName)
		if !utils.IsValidName(serviceName) {
			return utils.ErrInvalidName
		}

		return tailLoop(environmentType, serviceName)
	},
}

func tailLoop(environmentType string, serviceName string) error {
	var ts string
	var err error
	for {
		ts, err = tailLogs(environmentType, serviceName, ts)
		if err != nil && err != io.EOF {
			return err
		}
		time.Sleep(3 * time.Second)
	}
}

func tailLogs(environmentType string, serviceName string, timestamp string) (string, error) {
	authClient, err := auth.NewAuthClient(utils.Auth0BaseUrl, utils.Auth0ClientId, utils.ApiAudience)
	if err != nil {
		return "", err
	}
	unAuthConn, err := newConnection()
	if err != nil {
		return "", err
	}
	unAuthCliClient := pb.NewCliServiceClient(unAuthConn)
	accessToken, err := config.GetValidAccessTokenFromConfig(authClient, unAuthCliClient)
	unAuthConn.Close()
	if err != nil {
		return "", err
	}
	conn, err := newAuthenticatedConnection(accessToken)
	if err != nil {
		return "", err
	}

	defer conn.Close()
	cliClient := pb.NewCliServiceClient(conn)
	var trailer metadata.MD
	stream, err := cliClient.TailLogs(context.Background(), &pb.TailLogsRequest{
		EnvironmentType: environmentType,
		ServiceName:     serviceName,
		Timestamp:       timestamp,
	}, grpc.Trailer(&trailer))
	if err != nil {
		return "", err
	}

	var newTimestamp string
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return newTimestamp, err
		} else if err != nil {
			return "", err
		}

		newTimestamp = msg.Timestamp
		if msg.LogLine != "" {
			log.Printf("%s\n", msg.LogLine)
		}
	}
}

func init() {
	rootCmd.AddCommand(tailLogsCommand)

	tailLogsCommand.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	tailLogsCommand.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")

}
