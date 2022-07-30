package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	ga "github.com/mhelmich/go-archiver"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/secrets"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploys your service to Nucleus and returns an endpoint that you can use to communicate with your newly deployed service.",
	Long:  `Deploys your service to Nucleus and returns an endpoint that you can use to communicate with your newly deployed service.`,

	RunE: func(cmd *cobra.Command, args []string) error {
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

		if utils.IsValidEnvironmentType(environmentType) {
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

		conn, err := utils.NewApiConnectionByEnv(utils.GetEnv())
		if err != nil {
			return err
		}
		defer conn.Close()

		cliClient := pb.NewCliServiceClient(conn)

		ctx := context.Background()
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
		err = deploy(ctx, cliClient, req)
		if err != nil {
			return err
		}
		return setAuthzPolicy(
			ctx,
			cliClient,
			environmentType,
			serviceName,
			deployConfig.Spec.AllowedServices,
			deployConfig.Spec.DisallowedServices,
		)
	},
}

func setAuthzPolicy(
	ctx context.Context,
	cliClient pb.CliServiceClient,
	environmentType string,
	serviceName string,
	allowList []string,
	denyList []string,
) error {
	_, err := cliClient.SetServiceMtlsPolicy(ctx, &pb.SetServiceMtlsPolicyRequest{
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
}

func deploy(ctx context.Context, cliClient pb.CliServiceClient, req deployRequest) error {
	fmt.Printf("\nGetting deployment ready: \n↪Service: %s \n↪Environment: %s \n↪Project Directory: %s \n\n", req.serviceName, req.environmentType, req.folderPath)

	s1 := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
	s1.Start()

	// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
	var trailer metadata.MD
	_, err := cliClient.CreateEnvironment(ctx, &pb.CreateEnvironmentRequest{
		EnvironmentType: req.environmentType,
	},
		grpc.Trailer(&trailer),
	)
	if err != nil {
		return err
	}
	s1.Stop()

	if verbose {
		if len(trailer["x-request-id"]) == 1 {
			fmt.Printf("request id: %s\n", trailer["x-request-id"][0])
		}
	}

	deployRequest := pb.DeployRequest{
		EnvironmentType: req.environmentType,
		ServiceName:     req.serviceName,
		ServiceType:     req.serviceType,
		IsPrivate:       req.isPrivateService,
		Vars:            req.envVars,
		Secrets:         req.envSecrets,
	}

	if req.serviceType == "docker" {
		if req.image == "" {
			return fmt.Errorf("must provide image if service type is 'docker'")
		}
		deployRequest.Image = req.image
	} else {
		s1.Start()
		uploadKey, err := bundleAndUploadCode(ctx, cliClient, req.folderPath, req.environmentType, req.serviceName, &trailer)
		if err != nil {
			s1.Stop()
			return err
		}
		deployRequest.URL = uploadKey
		deployRequest.BuildCommand = req.buildCommand
		deployRequest.StartCommand = req.startCommand
	}

	stream, err := cliClient.Deploy(ctx, &deployRequest)
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

		servUrl := update.GetURL()
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

func getTotalTasks(taskCount *pb.DeploymentTaskStatusCount) int {
	if taskCount == nil {
		return 0
	}
	return int(taskCount.Completed) + int(taskCount.Failed) + int(taskCount.Incomplete) + int(taskCount.Skipped)
}

func bundleAndUploadCode(ctx context.Context, cliClient pb.CliServiceClient, folderPath string, environmentType string, serviceName string, trailer *metadata.MD) (string, error) {
	fd, err := ioutil.TempFile("", "nucleus-cli-")
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

	signedURL, err := cliClient.GetServiceUploadUrl(ctx, &pb.GetServiceUploadUrlRequest{
		EnvironmentType: environmentType,
		ServiceName:     serviceName,
	},
		grpc.Trailer(trailer),
	)
	if err != nil {
		return "", err
	}

	err = uploadArchive(signedURL.URL, fd)
	if err != nil {
		return "", err
	}
	return signedURL.UploadKey, nil
}

func uploadArchive(signedURL string, r io.Reader) error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
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
