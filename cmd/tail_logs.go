package cmd

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	"github.com/haikuapp/api/pkg/api/v1/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var tailLogsCommand = &cobra.Command{
	Use:   "tailService",
	Short: "Tails logs for a given service.",
	Long:  `Tails logs for a given service.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		deployConfig, err := getHaikuConfig()
		if err != nil {
			return err
		}

		environmentName := deployConfig.Spec.EnvironmentName
		if environmentName == "" {
			return errors.New("environment name not provided")
		}

		serviceName := deployConfig.Spec.ServiceName
		if serviceName == "" {
			return errors.New("service name not provided")
		}

		environmentName = strings.TrimSpace(environmentName)
		serviceName = strings.TrimSpace(serviceName)
		if !isValidName(environmentName) || !isValidName(serviceName) {
			return ErrInvalidName
		}

		return tailLoop(environmentName, serviceName)
	},
}

func tailLoop(environmentName string, serviceName string) error {
	var ts string
	var err error
	for {
		ts, err = tailLogs(environmentName, serviceName, ts)
		if err != nil && err != io.EOF {
			return err
		}
		// log.Printf("EOF ts: %s", ts)
		time.Sleep(1 * time.Second)
	}
}

func tailLogs(environmentName string, serviceName string, timestamp string) (string, error) {
	conn, err := newConnection()
	if err != nil {
		return "", err
	}

	defer conn.Close()
	cliClient := pb.NewCliServiceClient(conn)
	var trailer metadata.MD
	stream, err := cliClient.TailLogs(context.Background(), &pb.TailLogsRequest{
		EnvironmentName: environmentName,
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
}
