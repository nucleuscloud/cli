package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	ga "github.com/mhelmich/go-archiver"
	"github.com/nucleuscloud/cli/internal/config"
	"github.com/nucleuscloud/cli/internal/secrets"
	"github.com/nucleuscloud/cli/internal/utils"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
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

		if !utils.IsValidEnvironmentType(environmentType) {
			return fmt.Errorf("invalid value for environment")
		}

		if environmentType == "prod" {
			err := utils.PromptToProceed(cmd, environmentType, "yes")
			if err != nil {
				return err
			}
		}

		serviceName := deployConfig.Spec.ServiceName
		if serviceName == "" {
			return fmt.Errorf("service name not provided")
		}

		serviceType := deployConfig.Spec.ServiceRunTime
		if serviceType == "" {
			return fmt.Errorf("service type not provided")
		}

		// convert fastapi to python
		if serviceType == "fastapi" {
			serviceType = "python"
			deployConfig.Spec.ServiceRunTime = "python"
			err = config.SetNucleusConfig(deployConfig)
			if err != nil {
				return fmt.Errorf("unable to convert fastapi project to python")
			}
		}

		if serviceType == "python" {
			err = ensureProcfileExists()
			if err != nil {
				return err
			}
		}

		buildCommand := deployConfig.Spec.BuildCommand
		startCommand := deployConfig.Spec.StartCommand

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
			environmentType:  environmentType,
			serviceName:      serviceName,
			serviceType:      serviceType,
			image:            deployConfig.Spec.Image,
			folderPath:       directoryName,
			buildCommand:     buildCommand,
			startCommand:     startCommand,
			isPrivateService: deployConfig.Spec.IsPrivate,
			envVars:          deployConfig.Spec.Vars,
			envSecrets:       envSecrets,
		}
		err = deploy(ctx, svcClient, req)
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
	environmentType  string
	serviceName      string
	serviceType      string
	image            string
	folderPath       string
	buildCommand     string
	startCommand     string
	isPrivateService bool
	envVars          map[string]string
	envSecrets       map[string]string
	resources        config.ResourceRequirements
}

func deploy(
	ctx context.Context,
	svcClient svcmgmtv1alpha1.ServiceMgmtServiceClient,
	req deployRequest,
) error {
	fmt.Printf("\nGetting deployment ready: \n↪Service: %s \n↪Environment: %s \n↪Project Directory: %s \n\n", req.serviceName, req.environmentType, req.folderPath)

	s1 := spinner.New(spinner.CharSets[26], 100*time.Millisecond)

	deployRequest := svcmgmtv1alpha1.DeployServiceRequest{
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
		s1.Start()
		uploadKey, err := bundleAndUploadCode(ctx, svcClient, req.folderPath, req.environmentType, req.serviceName)
		if err != nil {
			s1.Stop()
			return err
		}
		deployRequest.UploadedCodeUri = uploadKey
		deployRequest.BuildCommand = req.buildCommand
		deployRequest.StartCommand = req.startCommand
	}

	stream, err := svcClient.DeployService(ctx, &deployRequest)
	if err != nil {
		s1.Stop()
		return err
	}
	s1.Stop()
	p := mpb.New(mpb.WithWidth(64))
	bar := getProgressBar(p, "Deploying service...", 0)
	var currCompleted int64 = 0
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			bar.Abort(true)
			break
		} else if err != nil {
			bar.Abort(true)
			log.Fatalf("server side error: %s", err.Error())
		}

		deployUpdate := update.GetDeploymentUpdate()
		if deployUpdate != nil {
			if deployUpdate.GetIsFailure() {
				bar.Abort(true)
				return fmt.Errorf(deployUpdate.GetMessage())
			}
			taskCount := deployUpdate.GetTaskStatusCount()
			totalTasks := getTotalTasks(taskCount)
			if taskCount != nil && totalTasks > 0 {
				if bar.Current() == 0 {
					bar.SetTotal(int64(totalTasks), false)
				}
				if taskCount.GetCompleted() != currCompleted {
					bar.IncrInt64(taskCount.GetCompleted() - currCompleted)
					currCompleted = taskCount.GetCompleted()
				}
			}
			continue
		}
		// should have to do a final increment because once all 4 tasks are completed we just return the url
		bar.Increment()
		// For some reason the bar never completes without this call.
		bar.EnableTriggerComplete()
		p.Wait()

		servUrl := update.GetServiceUrl()
		if servUrl == "" {
			fmt.Printf("Unable to retrieve URL..please try again")
		} else {
			fmt.Printf("\nService is deployed at: %s\n", servUrl)
		}
		break
	}
	p.Wait()

	return nil
}

func getTotalTasks(taskCount *svcmgmtv1alpha1.DeployServiceTaskStatusCount) int {
	if taskCount == nil {
		return 0
	}
	return int(taskCount.Completed) + int(taskCount.Failed) + int(taskCount.Incomplete) + int(taskCount.Skipped)
}

func bundleAndUploadCode(
	ctx context.Context,
	svcClient svcmgmtv1alpha1.ServiceMgmtServiceClient,
	folderPath string,
	environmentType string,
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
		EnvironmentType: environmentType,
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

	deployCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	deployCmd.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")
}

func getProgressBar(progress *mpb.Progress, name string, total int) *mpb.Bar {
	return progress.New(int64(total),
		// BarFillerBuilder with custom style
		mpb.BarStyle().Lbound("╢").Filler("▌").Tip("▌").Padding("░").Rbound("╟"),
		mpb.PrependDecorators(
			decor.Name(name, decor.WC{W: len(name) + 1, C: decor.DidentRight}),
			decor.OnComplete(
				decor.Spinner([]string{}),
				"",
			),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)
}
