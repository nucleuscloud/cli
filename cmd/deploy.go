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

	ga "github.com/mhelmich/go-archiver"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploys your service to Nucleus and returns an endpoint to call your service",
	Long:  `Creates an environment for your service with the given environmentName and a service with the given serviceName. Deploys your service and returns back a URL where your service is available. `,

	RunE: func(cmd *cobra.Command, args []string) error {
		deployConfig, err := getNucleusConfig()
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

		serviceType := deployConfig.Spec.ServiceRunTime
		if serviceType == "" {
			return errors.New("service type not provided")
		}

		directoryName, err := os.Getwd()
		if err != nil {
			return err
		}

		return deploy(environmentName, serviceName, serviceType, directoryName, deployConfig.Spec.IsPrivate, deployConfig.Spec.Vars)
	},
}

func deploy(environmentName string, serviceName string, serviceType string, folderPath string, isPrivateService bool, envVars map[string]string) error {
	log.Printf("Getting ready to deploy service: -%s- in environment: -%s- from directory: -%s- \n", serviceName, environmentName, folderPath)
	fd, err := ioutil.TempFile("", "nucleus-cli-")
	if err != nil {
		return err
	}

	if verbose {
		log.Printf("archiving directory into temp file: %s", fd.Name())
	}

	conn, err := newAuthenticatedConnection()
	if err != nil {
		return err
	}

	defer conn.Close()

	cliClient := pb.NewCliServiceClient(conn)
	// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
	var trailer metadata.MD
	reply, err := cliClient.CreateEnvironment(context.Background(), &pb.CreateEnvironmentRequest{
		EnvironmentName: environmentName,
	},
		grpc.Trailer(&trailer),
	)
	if err != nil {
		return err
	}

	fmt.Printf("environment successfully created with k8s id: %s\n", reply.ID)
	if verbose {
		if len(trailer["x-request-id"]) == 1 {
			fmt.Printf("request id: %s\n", trailer["x-request-id"][0])
		}
	}

	log.Printf("archiving...")
	gitignorePath := filepath.Join(folderPath, ".gitignore")
	_, err = os.Stat(gitignorePath)
	if errors.Is(err, os.ErrNotExist) {
		err = ga.GzipCompress(folderPath, fd, ga.IgnoreDotGit())
	} else {
		err = ga.GzipCompress(folderPath, fd, ga.ArchiveGitRepo())
	}
	if err != nil {
		return err
	}

	// flush buffer to disk
	err = fd.Sync()
	if err != nil {
		return err
	}

	// set file reader back to the beginning of the file
	_, err = fd.Seek(0, 0)
	if err != nil {
		return err
	}

	log.Printf("getting upload url...")
	ctx := context.Background()
	signedURL, err := cliClient.GetServiceUploadUrl(ctx, &pb.GetServiceUploadUrlRequest{
		EnvironmentName: environmentName,
		ServiceName:     serviceName,
	},
		grpc.Trailer(&trailer),
	)
	if err != nil {
		return err
	}

	log.Printf("uploading archive...")
	err = uploadArchive(signedURL.URL, fd)
	if err != nil {
		return err
	}

	log.Printf("triggering pipeline...")
	stream, err := cliClient.Deploy(ctx, &pb.DeployRequest{
		EnvironmentName: environmentName,
		ServiceName:     serviceName,
		URL:             signedURL.UploadKey,
		ServiceType:     serviceType,
		IsPrivate:       isPrivateService,
		Vars:            envVars,
	})
	if err != nil {
		return err
	}

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("server side error: %s", err.Error())
		}

		if msg := update.GetDeploymentUpdate(); msg != nil {
			log.Printf("%s", msg.Message)
			continue
		}

		log.Printf("service deployed under: %s\n", update.GetURL())
		break
	}

	return nil
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
}
