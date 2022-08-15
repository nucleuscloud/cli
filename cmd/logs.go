package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
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

		if utils.IsValidEnvironmentType(environmentType) {
			return fmt.Errorf("invalid value for environment")
		}

		serviceName := strings.TrimSpace(sn)
		if serviceName == "" {
			if config.DoesNucleusConfigExist() {
				cfg, err := config.GetNucleusConfig()
				if err != nil {
					return err
				}
				serviceName = cfg.Spec.ServiceName
			}
		}

		if !utils.IsValidName(serviceName) {
			return utils.ErrInvalidServiceName
		}

		tail, err := cmd.Flags().GetBool("tail")
		if err != nil {
			return err
		}
		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			return err
		}

		shouldTail := tail || follow
		if onPrem {
			return getOnPremLogs(environmentType, serviceName, window, shouldTail)
		}

		// managed
		if tail {
			return liveTailLogs(environmentType, serviceName)
		} else if allowedWindowValues(window) {
			return staticLogs(environmentType, serviceName, window)
		} else {
			fmt.Println("Pass in a flag to get logs.")
			return nil
		}
	},
}

func getOnPremLogs(envType string, serviceName string, window string, shouldTail bool) error {
	conn, err := utils.NewApiConnectionByEnv(utils.GetEnv(), true)
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	logStream, err := cliClient.GetServiceLogs(context.Background(), &svcmgmtv1alpha1.GetServiceLogsRequest{
		EnvironmentType: envType,
		ServiceName:     serviceName,
		Window:          getLogWindow(window),
		ShouldTail:      shouldTail,
	})
	if err != nil {
		return err
	}
	defer logStream.CloseSend()
	for {
		msg, err := logStream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		fmt.Println(msg.Log)
	}
	return nil
}

func staticLogs(environmentType string, serviceName string, window string) error {
	conn, err := utils.NewApiConnectionByEnv(utils.GetEnv(), false)
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := pb.NewCliServiceClient(conn)
	logs, err := cliClient.Logs(context.Background(), &pb.LogsRequest{
		EnvironmentType: environmentType,
		ServiceName:     serviceName,
		Window:          window,
	})
	if err != nil && err != io.EOF {
		return err
	}

	fmt.Println("\nGenerating logs for the last " + window + "...\n")

	if len(logs.Log) == 0 {
		fmt.Println("No logs for this service at this time window. if you just deployed your service, try again in a minute or try a bigger time window.")
	}

	for i := 0; i < len(logs.Log); i++ {
		fmt.Println(string(logs.Log[i]))
	}
	return nil
}

func getLogWindow(window string) svcmgmtv1alpha1.LogWindow {
	switch window {
	case "15m":
		return svcmgmtv1alpha1.LogWindow_LOG_WINDOW_FIFTEEN_MIN
	case "1h":
		return svcmgmtv1alpha1.LogWindow_LOG_WINDOW_ONE_HOUR
	case "1d":
		return svcmgmtv1alpha1.LogWindow_LOG_WINDOW_ONE_DAY
	default:
		return svcmgmtv1alpha1.LogWindow_LOG_WINDOW_NO_TIME_UNSPECIFIED
	}
}

func liveTailLogs(environmentType string, serviceName string) error {
	conn, err := utils.NewApiConnectionByEnv(utils.GetEnv(), false)
	if err != nil {
		return err
	}

	defer conn.Close()

	var timestamp string
	cliClient := pb.NewCliServiceClient(conn)
	stream, err := cliClient.TailLogs(context.Background(), &pb.TailLogsRequest{
		EnvironmentType: environmentType,
		ServiceName:     serviceName,
		Timestamp:       timestamp,
	})
	if err != nil {
		return err
	}

	fmt.Print("\nStarting live tail, only new logs are published ...\n")

	check := "check"
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if check != resp.LogLine {
			fmt.Println("\n" + resp.LogLine)
		}
		if err != nil {
			log.Fatalf("can not receive %v", err)
			break
		}

		check = resp.LogLine
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
	logsCommand.Flags().BoolP("tail", "t", false, "live log tail")
	logsCommand.Flags().BoolP("follow", "f", false, "live log tail")
	logsCommand.Flags().StringP("service", "s", "", "service name")
	logsCommand.Flags().StringP("window", "w", "", "logging window allowed values: [15min, 1h, 1d]")
}
