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
	"github.com/nucleuscloud/cli/internal/progress"
	"github.com/nucleuscloud/cli/internal/secrets"
	"github.com/nucleuscloud/cli/internal/utils"
)

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

		environmentType, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}
		if environmentType == "" {
			return fmt.Errorf("must provide environment type")
		}

		serviceName := deployConfig.Spec.ServiceName
		if serviceName == "" {
			return fmt.Errorf("service name not provided")
		}

		serviceType := deployConfig.Spec.ServiceRunTime
		if serviceType == "" {
			return fmt.Errorf("service type not provided")
		}

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

		envSecrets := secrets.GetSecretsByEnvType(&deployConfig.Spec, environmentType)
		if err != nil {
			return err
		}

		conn, err := utils.NewApiConnectionByEnv(ctx, utils.GetEnv())
		if err != nil {
			return err
		}
		defer conn.Close()

		svcClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)

		req := deployRequest{
			cliVersion:       deployConfig.CliVersion,
			environmentType:  environmentType,
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
			environmentType,
			serviceName,
			deployConfig.Spec.AllowedServices,
			deployConfig.Spec.DisallowedServices,
		)
	},
}

func setAuthzPolicy(
	ctx context.Context,
	svcClient svcmgmtv1alpha1.ServiceMgmtServiceClient,
	environmentType string,
	serviceName string,
	allowList []string,
	denyList []string,
) error {
	_, err := svcClient.SetServiceMtlsPolicy(ctx, &svcmgmtv1alpha1.SetServiceMtlsPolicyRequest{
		EnvironmentType:    environmentType,
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
	environmentType  string
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
		green("↪"), req.environmentType,
		green("↪"), req.folderPath,
	)

	deployRequest := svcmgmtv1alpha1.DeployServiceRequest{
		CliVersion:      req.cliVersion,
		EnvironmentType: req.environmentType,
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
		uploadKey, err := bundleAndUploadCode(ctx, svcClient, req.folderPath, req.environmentType, req.serviceName)
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

	mainBar := getProgressBar(progressContainer, "Deploying your service...", 100)
	printPlainOutput := getPlainOutput()

	for {
		response, err := stream.Recv()
		if err != nil {
			mainBar.Abort(true)
			return err
		}

		if response.GetServiceUrl() != "" {
			mainBar.SetCurrent(100)
			progressContainer.Wait()
			fmt.Printf("\nService is deployed at: %s\n", green(response.GetServiceUrl()))
			break
		}

		deployStatus := response.GetDeployStatus()
		if deployStatus == nil {
			continue
		}

		if progressType == progress.PlainProgress {
			// plain output
			printPlainOutput(deployStatus)
		} else {
			mainBar.SetCurrent(int64(getCompletionPercentage(deployStatus)))
		}
	}

	return nil
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

	return func(deployStatus *svcmgmtv1alpha1.DeployStatus) {
		plainOutput(deployStatus)
	}
}

func plainOutput(deployUpdate *svcmgmtv1alpha1.DeployStatus) {
	fmt.Println("====================")
	if deployUpdate.Succeeded != nil {
		fmt.Println("Status Message", deployUpdate.Succeeded.Message)
	}
	for _, taskStatus := range deployUpdate.DeployTaskStatus {
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
		EnvironmentType: environmentName,
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
