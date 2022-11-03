package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

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

		if !utils.IsValidEnvironmentType(environmentType) {
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
		return getLogs(environmentType, serviceName, window, shouldTail)
	},
}

func getLogs(envType string, serviceName string, window string, shouldTail bool) error {
	conn, err := utils.NewApiConnectionByEnv(utils.GetEnv())
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
	for {
		msg, err := logStream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			err2 := logStream.CloseSend()
			if err2 != nil {
				fmt.Println(err2)
			}
			return err
		}
		fmt.Println(msg.Log)
	}
	return logStream.CloseSend()
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

func init() {
	rootCmd.AddCommand(logsCommand)
	logsCommand.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	logsCommand.Flags().BoolP("tail", "t", false, "live log tail")
	logsCommand.Flags().BoolP("follow", "f", false, "live log tail")
	logsCommand.Flags().StringP("service", "s", "", "service name")
	logsCommand.Flags().StringP("window", "w", "", "logging window allowed values: [15min, 1h, 1d]")
}
