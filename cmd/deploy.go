package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	ga "github.com/mhelmich/go-archiver"
	"github.com/mhelmich/haiku-api/pkg/api/v1/pb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploys your service to Haiku and returns an endpoint to call your service",
	Long:  `Creates an environment for your service with the given environmentName and a service with the given serviceName. Deploys your service and returns back a URL where your service is available. `,

	RunE: func(cmd *cobra.Command, args []string) error {

		environmentName, err := cmd.Flags().GetString(environmentFlag[0])
		if err != nil {
			return err
		}
		if environmentName == "" {
			return errors.New("environment name not provided")
		}

		serviceName, err := cmd.Flags().GetString(serviceNameFlag[0])
		if err != nil {
			return err
		}
		if serviceName == "" {
			return errors.New("service name not provided")
		}

		directoryName, err := cmd.Flags().GetString(directoryFlag[0])
		if err != nil {
			return err
		} else if directoryName == "" {
			return errors.New("directory name not provided")
		}

		return deploy(environmentName, serviceName, directoryName)
	},
}

func deploy(environmentName string, serviceName string, folderPath string) error {
	log.Printf("Getting ready to deploy service: -%s- in environment: -%s- from directory: -%s- \n", serviceName, environmentName, folderPath)
	fd, err := ioutil.TempFile("", "haiku-cli-")
	if err != nil {
		return err
	}

	conn, err := newConnection()
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

	req, err := http.NewRequest(http.MethodPut, signedURL, r)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/zip")
	rsp, err := httpClient.Do(req)
	if err != nil {
		return err
	} else if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload didn't work: %s", rsp.Status)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(deployCmd)
	stringP(deployCmd, serviceNameFlag)
	stringP(deployCmd, directoryFlag)
	stringP(deployCmd, environmentFlag)
}
