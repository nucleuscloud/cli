package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	ga "github.com/mhelmich/go-archiver"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/nucleuscloud/cli/internal/config"
	clienv "github.com/nucleuscloud/cli/internal/env"
	"github.com/nucleuscloud/cli/internal/progress"
	"github.com/nucleuscloud/cli/internal/secrets"
	"github.com/nucleuscloud/cli/internal/utils"
)

type ProgressBar struct {
	abort      bool
	currentInt int64
}

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys your service to Nucleus and returns an endpoint that you can use to communicate with your newly deployed service.",
	Long:  `Deploys your service to Nucleus and returns an endpoint that you can use to communicate with your newly deployed service.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		deployConfig, err := config.GetNucleusConfig()
		if err != nil {
			return err
		}

		if !utils.IsValidName(deployConfig.Spec.ServiceName) {
			return utils.ErrInvalidServiceName
		}

		environmentName, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}
		if environmentName == "" {
			return fmt.Errorf("must provide environment name")
		}

		serviceName := deployConfig.Spec.ServiceName
		if serviceName == "" {
			return fmt.Errorf("service name not provided")
		}

		serviceType := deployConfig.Spec.ServiceRunTime
		if serviceType == "" {
			return fmt.Errorf("service type not provided")
		}
		if !utils.IsValidRuntime(serviceType) {
			return fmt.Errorf("must provide valid service runtime")
		}

		// Set this after ensuring flags are correct
		cmd.SilenceUsage = true

		if serviceType == "python" {
			err = ensureProcfileExists()
			if err != nil {
				return err
			}
		}

		progressType, err := progress.ValidateAndRetrieveProgressFlag(cmd)
		if err != nil {
			return err
		}

		directoryName, err := os.Getwd()
		if err != nil {
			return err
		}

		envSecrets := secrets.GetSecretsByEnvName(&deployConfig.Spec, environmentName)
		if err != nil {
			return err
		}

		conn, err := utils.NewApiConnectionByEnv(ctx, clienv.GetEnv())
		if err != nil {
			return err
		}
		defer conn.Close()

		svcClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)

		req := deployRequest{
			cliVersion:       deployConfig.CliVersion,
			environmentName:  environmentName,
			serviceName:      serviceName,
			serviceType:      serviceType,
			image:            deployConfig.Spec.Image,
			folderPath:       directoryName,
			isPrivateService: deployConfig.Spec.IsPrivate,
			envVars:          deployConfig.Spec.Vars,
			envSecrets:       envSecrets,
		}
		err = deploy(ctx, svcClient, req, progressType)
		if err != nil {
			return err
		}
		return setAuthzPolicy(
			ctx,
			svcClient,
			environmentName,
			serviceName,
			deployConfig.Spec.AllowedServices,
			deployConfig.Spec.DisallowedServices,
		)
	},
}

func setAuthzPolicy(
	ctx context.Context,
	svcClient svcmgmtv1alpha1.ServiceMgmtServiceClient,
	environmentName string,
	serviceName string,
	allowList []string,
	denyList []string,
) error {
	_, err := svcClient.SetServiceMtlsPolicy(ctx, &svcmgmtv1alpha1.SetServiceMtlsPolicyRequest{
		EnvironmentName:    environmentName,
		ServiceName:        serviceName,
		AllowedServices:    allowList,
		DisallowedServices: denyList,
	})
	if err != nil {
		return err
	}
	return nil
}

type deployRequest struct {
	cliVersion       string
	environmentName  string
	serviceName      string
	serviceType      string
	image            string
	folderPath       string
	isPrivateService bool
	envVars          map[string]string
	envSecrets       map[string]string
	resources        config.ResourceRequirements
}

func deploy(
	ctx context.Context,
	svcClient svcmgmtv1alpha1.ServiceMgmtServiceClient,
	req deployRequest,
	progressType progress.ProgressType,
) error {
	green := progress.SProgressPrint(progressType, color.FgGreen)
	fmt.Printf("\nGetting deployment ready: \n%sService: %s \n%sEnvironment: %s \n%sProject Directory: %s \n\n",
		green("↪"), req.serviceName,
		green("↪"), req.environmentName,
		green("↪"), req.folderPath,
	)

	deployRequest := svcmgmtv1alpha1.DeployServiceRequest{
		CliVersion:      req.cliVersion,
		EnvironmentName: req.environmentName,
		ServiceName:     req.serviceName,
		ServiceType:     req.serviceType,
		IsPrivate:       req.isPrivateService,
		EnvVars:         req.envVars,
		Secrets:         req.envSecrets,
		Resources: &svcmgmtv1alpha1.ResourceRequirements{
			Minimum: &svcmgmtv1alpha1.ResourceList{
				Cpu:    req.resources.Minimum.Cpu,
				Memory: req.resources.Minimum.Memory,
			},
			Maximum: &svcmgmtv1alpha1.ResourceList{
				Cpu:    req.resources.Maximum.Cpu,
				Memory: req.resources.Maximum.Memory,
			},
		},
	}

	if req.serviceType == "docker" {
		if req.image == "" {
			return fmt.Errorf("must provide image if service type is 'docker'")
		}
		deployRequest.DockerImage = req.image
	} else {
		uploadSpinner := spinner.New(spinner.CharSets[35], 100*time.Millisecond)
		uploadSpinner.Suffix = "  Bundling and uploading code..."
		if progressType == progress.TtyProgress {
			uploadSpinner.Start()
		} else {
			fmt.Println("Bundling and uploading code...")
		}
		uploadKey, err := bundleAndUploadCode(ctx, svcClient, req.folderPath, req.environmentName, req.serviceName)
		uploadSpinner.Stop()
		if err != nil {
			return err
		}
		deployRequest.UploadedCodeUri = uploadKey
	}

	deployInitSpinner := spinner.New(spinner.CharSets[35], 100*time.Millisecond)
	deployInitSpinner.Suffix = "  Initiating deployment request"
	if progressType == progress.TtyProgress {
		deployInitSpinner.Start()
	}
	stream, err := svcClient.DeployService(ctx, &deployRequest)
	deployInitSpinner.Stop()
	if err != nil {
		return err
	}

	termWidth := progress.GetProgressBarWidth(50)
	progressContainer := mpb.NewWithContext(
		ctx,
		mpb.WithWidth(termWidth),
	)

	var mainBar *mpb.Bar
	if progressType != progress.PlainProgress {
		mainBar = getProgressBar(progressContainer, "Deploying your service...", 100)
	}
	printPlainOutput := getPlainOutput()

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			handleMainBar(mainBar, progressType, &ProgressBar{abort: true})
			return err
		}

		if response.GetServiceUrl() != "" {
			handleMainBar(mainBar, progressType, &ProgressBar{currentInt: 100})
			progressContainer.Wait()
			fmt.Printf("\nService is deployed at: %s\n", green(response.GetServiceUrl()))
			break
		}

		deployStatus := response.GetDeployStatus()
		if deployStatus == nil {
			continue
		}

		if didPipelineFail(deployStatus) {
			handleMainBar(mainBar, progressType, &ProgressBar{abort: true})
			progressContainer.Wait()
			printPlainOutput(deployStatus)
			if didPipelineGetCancelled(deployStatus) {
				return nil
			}
			err = streamPodErrorLogs(ctx, svcClient, req.environmentName, req.serviceName, deployStatus)
			if err != nil {
				return err
			}
			return fmt.Errorf("pipeline failed with error")
		}

		if progressType == progress.PlainProgress {
			// plain output
			printPlainOutput(deployStatus)
		} else {
			handleMainBar(mainBar, progressType, &ProgressBar{currentInt: int64(getCompletionPercentage(deployStatus))})
		}
	}

	return nil
}

// This doesnt handle the option where a task is gracefully shutdown
func didPipelineGetCancelled(
	deployStatus *svcmgmtv1alpha1.DeployStatus,
) bool {
	if deployStatus == nil {
		return false
	}
	if deployStatus.Succeeded != nil && deployStatus.Succeeded.Reason == "Cancelled" {
		return true
	}
	for _, taskStatus := range deployStatus.DeployTaskStatus {
		if taskStatus.CompletionTime == nil {
			continue
		}
		for _, step := range taskStatus.Steps {
			terminatedState := step.GetTerminated()
			if terminatedState == nil {
				continue
			}
			if terminatedState.Reason == "TaskRunCancelled" {
				return true
			}
		}
	}
	return false
}

func streamPodErrorLogs(
	ctx context.Context,
	client svcmgmtv1alpha1.ServiceMgmtServiceClient,
	envName string,
	serviceName string,
	deployStatus *svcmgmtv1alpha1.DeployStatus,
) error {
	if deployStatus == nil {
		return fmt.Errorf("deploystatus was nil")
	}

	logrequest, err := getLogRequestFromDeployStatus(deployStatus)
	if err != nil {
		return err
	}

	fmt.Println("====================")
	fmt.Printf("Printing logs for Task '%s' at Step '%s'\n\n", logrequest.TaskName, logrequest.TaskStep)
	fmt.Println("====================")

	stream, err := client.GetDeployLogs(ctx, &svcmgmtv1alpha1.GetDeployLogsRequest{
		EnvironmentName: envName,
		ServiceName:     serviceName,
		LogRequest:      logrequest,
	})
	if err != nil {
		return err
	}
	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				fmt.Println("====================")
				return nil
			}
			return err
		}
		fmt.Println(response.LogLine)
	}
}

func getLogRequestFromDeployStatus(
	deployStatus *svcmgmtv1alpha1.DeployStatus,
) (*svcmgmtv1alpha1.PipelineRunLogRequest, error) {
	if deployStatus == nil {
		return nil, fmt.Errorf("deploy status was nil")
	}

	taskStatus := getLastTaskStatus(deployStatus.DeployTaskStatus)
	if taskStatus == nil {
		return nil, fmt.Errorf("unable to find task status with error state to print logs for")
	}

	for _, step := range taskStatus.Steps {
		terminatedState := step.GetTerminated()
		if terminatedState == nil {
			continue
		}
		if terminatedState.Reason == "Error" {
			return &svcmgmtv1alpha1.PipelineRunLogRequest{
				PipelineRun: deployStatus.PipelineRun,
				TaskName:    taskStatus.Name,
				TaskStep:    step.Name,
			}, nil
		}
	}
	return nil, fmt.Errorf("unable to find task step with error state to print logs for")
}

func getLastTaskStatus(statuses []*svcmgmtv1alpha1.DeployTaskStatus) *svcmgmtv1alpha1.DeployTaskStatus {
	if len(statuses) == 0 {
		return nil
	}

	var lastTaskStatus *svcmgmtv1alpha1.DeployTaskStatus

	for _, taskStatus := range statuses {
		if taskStatus.CompletionTime == nil {
			continue
		}

		if lastTaskStatus == nil {
			lastTaskStatus = taskStatus
		} else {
			lastCompletionTime, err := time.Parse(time.RFC3339, *lastTaskStatus.CompletionTime)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			completionTime, err := time.Parse(time.RFC3339, *taskStatus.CompletionTime)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return nil
			}
			if completionTime.After(lastCompletionTime) {
				lastTaskStatus = taskStatus
			}
		}
	}
	return lastTaskStatus
}

func handleMainBar(bar *mpb.Bar, progressType progress.ProgressType, barSettings *ProgressBar) {
	if progressType == progress.PlainProgress {
		return
	}
	if barSettings.abort {
		bar.Abort(true)
		return
	}
	bar.SetCurrent(barSettings.currentInt)
}

func didPipelineFail(deployStatus *svcmgmtv1alpha1.DeployStatus) bool {
	return deployStatus != nil && deployStatus.Succeeded != nil && deployStatus.Succeeded.Status == "False"
}

func getCompletionPercentage(deployStatus *svcmgmtv1alpha1.DeployStatus) float64 {
	maxComplete := 0.

	numStages := float64(len(deployStatus.DeployTaskStatus))

	for tsIdx, taskStatus := range deployStatus.DeployTaskStatus {
		floorVal := float64(tsIdx) / numStages
		ceilVal := float64((tsIdx + 1)) / numStages
		if taskStatus.StartTime == nil {
			continue
		}

		numSteps := len(taskStatus.Steps)
		for stepIdx, step := range taskStatus.Steps {
			currstep := stepIdx + 1
			subCeil := getPercentageComplete(ceilVal, floorVal, currstep, numSteps)
			totalSubSteps := 3 // waiting, running, terminated

			if step.GetWaiting() != nil {
				maxComplete = getPercentageComplete(subCeil, floorVal, 1, totalSubSteps)
			}
			if step.GetRunning() != nil {
				maxComplete = getPercentageComplete(subCeil, floorVal, 2, totalSubSteps)
			}
			if step.GetTerminated() != nil {
				// equivalent to: getPercentageComplete(subCeil, floorVal, 3, 3)
				maxComplete = subCeil
			}
		}
	}
	return maxComplete * 100
}

func getPercentageComplete(ceilVal float64, floorVal float64, currStep int, numSteps int) float64 {
	return ((ceilVal-floorVal)*(float64(currStep)/float64(numSteps)) + floorVal)
}

func getPlainOutput() func(deployStatus *svcmgmtv1alpha1.DeployStatus) {
	printedTasks := map[string]int{}

	return func(deployStatus *svcmgmtv1alpha1.DeployStatus) {
		plainOutput(deployStatus, printedTasks)
	}
}

func plainOutput(deployUpdate *svcmgmtv1alpha1.DeployStatus, printedTasks map[string]int) {
	for _, taskStatus := range deployUpdate.DeployTaskStatus {
		if shouldPrint(taskStatus, printedTasks) {
			fmt.Println("====================")
			if deployUpdate.Succeeded != nil {
				fmt.Println("Status Message", deployUpdate.Succeeded.Message)
			}
			fmt.Printf("    Task Name: '%s'\n    Start Time: '%s'\n    Complete Time: '%s'\n    Num Steps: '%d'\n", taskStatus.Name, taskStatus.GetStartTime(), taskStatus.GetCompletionTime(), len(taskStatus.Steps))
			for _, step := range taskStatus.Steps {
				fmt.Printf("        Task Step: '%s'\n", step.Name)
				waiting := step.GetWaiting()
				terminating := step.GetTerminated()
				running := step.GetRunning()
				if waiting != nil {
					fmt.Printf("            Waiting: Reason: '%s' Message: '%s'\n", waiting.Reason, waiting.Message)
				} else if running != nil {
					fmt.Printf("            Running: '%s'\n", running.StartedAt)
				} else if terminating != nil {
					fmt.Printf("            Terminated: Reason: '%s' Message: '%s' Started At: '%s' Finished At: '%s'\n", terminating.Reason, terminating.Message, terminating.StartedAt, terminating.FinishedAt)
				}
			}
		}
	}
}

func shouldPrint(taskStatus *svcmgmtv1alpha1.DeployTaskStatus, printedTasks map[string]int) bool {
	if taskStatus.GetCompletionTime() != "" {
		taskName := fmt.Sprintf("%s-complete", taskStatus.Name)
		if _, ok := printedTasks[taskName]; ok {
			return false
		}
		printedTasks[taskName] = 1
		return true
	}
	if len(taskStatus.Steps) != 0 {
		taskName := fmt.Sprintf("%s-incomplete", taskStatus.Name)
		if _, ok := printedTasks[taskName]; ok {
			return false
		}
		printedTasks[taskName] = 1
		return true
	}
	return false
}

func bundleAndUploadCode(
	ctx context.Context,
	svcClient svcmgmtv1alpha1.ServiceMgmtServiceClient,
	folderPath string,
	environmentName string,
	serviceName string,
) (string, error) {
	fd, err := os.CreateTemp("", "nucleus-cli-")
	if err != nil {
		return "", err
	}

	if verbose {
		fmt.Printf("archiving directory into temp file: %s", fd.Name())
	}

	gitignorePath := filepath.Join(folderPath, ".gitignore")
	_, err = os.Stat(gitignorePath)
	if errors.Is(err, os.ErrNotExist) {
		err = ga.GzipCompress(folderPath, fd, ga.IgnoreDotGit())
	} else {
		err = ga.GzipCompress(folderPath, fd, ga.ArchiveGitRepo())
	}
	if err != nil {
		return "", err
	}

	// flush buffer to disk
	err = fd.Sync()
	if err != nil {
		return "", err
	}

	// set file reader back to the beginning of the file
	_, err = fd.Seek(0, 0)
	if err != nil {
		return "", err
	}

	signedResponse, err := svcClient.GetServiceUploadUrl(ctx, &svcmgmtv1alpha1.GetServiceUploadUrlRequest{
		EnvironmentName: environmentName,
		ServiceName:     serviceName,
	})
	if err != nil {
		return "", err
	}

	err = uploadArchive(signedResponse.Url, fd)
	if err != nil {
		return "", err
	}
	return signedResponse.UploadKey, nil
}

func uploadArchive(signedURL string, r io.Reader) error {
	httpClient := &http.Client{
		Timeout: 120 * time.Second,
	}

	// TODO: Figure out how to send this as a stream instead of reading into memory.
	// S3 seems to not like when we send this as a stream
	// https://stackoverflow.com/questions/67896779/streaming-uploading-to-s3-using-presigned-url
	// https://github.com/aws/aws-sdk-js/issues/1603
	buf := &bytes.Buffer{}
	nRead, err := io.Copy(buf, r)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, signedURL, buf)
	if err != nil {
		return err
	}

	req.Header.Set("content-type", "application/gzip")
	req.Header.Set("content-length", strconv.FormatInt(nRead, 10))
	rsp, err := httpClient.Do(req)
	if err != nil {
		return err
	} else if rsp.StatusCode != http.StatusOK {
		fmt.Print(rsp)
		return fmt.Errorf("upload didn't work: %s", rsp.Status)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringP("env", "e", "", "set the nucleus environment")
	progress.AttachProgressFlag(deployCmd)
}

func getProgressBar(progress *mpb.Progress, name string, total int) *mpb.Bar {
	spinnerStyle := []string{
		color.GreenString("⠋"),
		color.GreenString("⠙"),
		color.GreenString("⠹"),
		color.GreenString("⠸"),
		color.GreenString("⠼"),
		color.GreenString("⠴"),
		color.GreenString("⠦"),
		color.GreenString("⠧"),
		color.GreenString("⠇"),
		color.GreenString("⠏"),
	}
	return progress.New(int64(total),
		// BarFillerBuilder with custom style
		mpb.BarStyle().Lbound("╢").Filler("▌").Tip("▌").Padding("░").Rbound("╟"),
		mpb.PrependDecorators(
			decor.Name(name, decor.WC{W: len(name) + 1, C: decor.DidentRight}),
			decor.OnComplete(
				decor.Spinner(spinnerStyle),
				"",
			),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)
}
