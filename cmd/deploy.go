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

		serviceType := deployConfig.Spec.ServiceRunTime
		if serviceType == "" {
			return errors.New("service type not provided")
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
		return deploy(environmentType, serviceName, serviceType, directoryName, buildCommand, startCommand, false, deployConfig.Spec.Vars, envSecrets)
	},
}

func deploy(environmentType string, serviceName string, serviceType string, folderPath string, buildCommand string, startCommand string, isPrivateService bool, envVars map[string]string, envSecrets map[string]string) error {
	fmt.Printf("\nGetting deployment ready: \n↪Service: %s \n↪Environment: %s \n↪Project Directory: %s \n\n", serviceName, environmentType, folderPath)

	s1 := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
	s1.Start()

	conn, err := utils.NewApiConnection(utils.ApiConnectionConfig{
		AuthBaseUrl:  utils.Auth0BaseUrl,
		AuthClientId: utils.Auth0ClientId,
		ApiAudience:  utils.ApiAudience,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	cliClient := pb.NewCliServiceClient(conn)
	// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
	var trailer metadata.MD
	_, err = cliClient.CreateEnvironment(context.Background(), &pb.CreateEnvironmentRequest{
		EnvironmentType: environmentType,
	},
		grpc.Trailer(&trailer),
	)
	if err != nil {
		return err
	}

	if verbose {
		if len(trailer["x-request-id"]) == 1 {
			fmt.Printf("request id: %s\n", trailer["x-request-id"][0])
		}
	}

	ctx := context.Background()
	uploadKey, err := bundleAndUploadCode(ctx, cliClient, folderPath, environmentType, serviceName, &trailer)
	if err != nil {
		return err
	}

	stream, err := cliClient.Deploy(ctx, &pb.DeployRequest{
		EnvironmentType: environmentType,
		ServiceName:     serviceName,
		URL:             uploadKey,
		ServiceType:     serviceType,
		BuildCommand:    buildCommand,
		StartCommand:    startCommand,
		IsPrivate:       isPrivateService,
		Vars:            envVars,
		Secrets:         envSecrets,
	})
	if err != nil {
		return err
	}
	s1.Stop()

	//staging the deploy, total time is about 2100
	// progressBar("Initializing deployment... ", 700)
	// fmt.Print("\n")
	// progressBar("Deploying service ...      ", 700)
	// fmt.Print("\n")
	// progressBar("Finalizing deployment...   ", 700)
	// fmt.Print("\n")

	servUrl := ""

	p := mpb.New(mpb.WithWidth(64))
	defer p.Wait()
	total := 100

	bar := getProgressBar(p, "Deploying service ...", total)

	// max := time.Duration(700) * time.Millisecond //this value should be 2x what you think you need since the rand.Intn function takes a random sampling which comes out to about 50% of the value you set
	// for i := 0; i < total; i++ {
	// 	if shouldAbort() {
	// 		bar.Abort(false)
	// 	}
	// 	time.Sleep(time.Duration(rand.Intn(10)+1) * max / 10)
	// 	bar.Increment()
	// }
	// // wait for our bar to complete and flush
	// p.Wait()
	s2 := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
	s2.Start()
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			// bar.Abort(true)
			s2.Stop()
			log.Fatalf("server side error: %s", err.Error())
		}

		// bar.Increment()
		deployUpdate := update.GetDeploymentUpdate()
		if deployUpdate != nil {
			bar.Increment()
			continue
		}

		servUrl = update.GetURL()
		s2.Stop()
		fmt.Printf("Service is deployed at: %s\n", servUrl)
		break
	}
	return nil
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

// func progressBar(message string, length int32, shouldAbort func() bool) {
// 	p := mpb.New(mpb.WithWidth(64))
// 	total := 100

// 	bar := getProgressBar(p, message, total)

// 	max := time.Duration(length) * time.Millisecond //this value should be 2x what you think you need since the rand.Intn function takes a random sampling which comes out to about 50% of the value you set
// 	for i := 0; i < total; i++ {
// 		if shouldAbort() {
// 			bar.Abort(false)
// 		}
// 		time.Sleep(time.Duration(rand.Intn(10)+1) * max / 10)
// 		bar.Increment()
// 	}
// 	// wait for our bar to complete and flush
// 	p.Wait()
// }

func getProgressBar(progress *mpb.Progress, name string, total int) *mpb.Bar {
	return progress.New(int64(total),
		// BarFillerBuilder with custom style
		mpb.BarStyle().Lbound("╢").Filler("▌").Tip("▌").Padding("░").Rbound("╟"),
		mpb.PrependDecorators(
			// display our name with one space on the right
			decor.Name(name, decor.WC{W: len(name) + 1, C: decor.DidentRight}),
			// replace ETA decorator with "done" message, OnComplete event
			decor.OnComplete(
				decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 4}), "Done",
			),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringP("env", "e", "prod", "set the nucleus environment")
	deployCmd.Flags().BoolP("yes", "y", false, "automatically answer yes to the prod prompt")
}
