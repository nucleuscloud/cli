package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nucleuscloud/cli/internal/config"
	clienv "github.com/nucleuscloud/cli/internal/env"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
)

var logsCommand = &cobra.Command{
	Use:   "logs",
	Short: "Returns logs for a given service.",
	Long:  `Returns logs for a given service.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		environmentName, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}
		if environmentName == "" {
			return fmt.Errorf("must provide environment name")
		}

		sn, err := cmd.Flags().GetString("service")
		if err != nil {
			return err
		}

		window, err := cmd.Flags().GetString("window")
		if err != nil {
			return err
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

		podName, err := cmd.Flags().GetString("pod")
		if err != nil {
			return err
		}

		maxLines, err := cmd.Flags().GetInt64("max-lines")
		if err != nil {
			return err
		}

		// Set this after ensuring flags are correct
		cmd.SilenceUsage = true

		shouldTail := tail || follow
		var maxLogLines *int64
		if maxLines > 0 {
			maxLogLines = &maxLines
		}
		var parsedPodName *string
		if len(podName) > 0 {
			parsedPodName = &podName
		}
		return getLogs(ctx, environmentName, serviceName, parsedPodName, window, shouldTail, maxLogLines)
	},
}

func getLogs(ctx context.Context, envName string, serviceName string, podName *string, window string, shouldTail bool, maxLines *int64) error {
	conn, err := utils.NewApiConnectionByEnv(ctx, clienv.GetEnv())
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
	logStream, err := cliClient.GetServiceLogs(ctx, &svcmgmtv1alpha1.GetServiceLogsRequest{
		EnvironmentName: envName,
		ServiceName:     serviceName,
		Window:          getLogWindow(window),
		ShouldTail:      shouldTail,
		PodName:         podName,
		MaxLogLines:     maxLines,
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
				fmt.Fprintln(os.Stderr, err2)
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
	logsCommand.Flags().StringP("env", "e", "", "set the nucleus environment")
	logsCommand.Flags().BoolP("tail", "t", false, "live log tail")
	logsCommand.Flags().BoolP("follow", "f", false, "live log tail")
	logsCommand.Flags().StringP("service", "s", "", "service name")
	logsCommand.Flags().StringP("window", "w", "", "logging window allowed values: [15min, 1h, 1d]")
	logsCommand.Flags().StringP("pod", "p", "", "specific pod to pull logs from")
	logsCommand.Flags().Int64("max-lines", 0, "will return only the max number of lines. 0 means all")
}
