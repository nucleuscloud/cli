package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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
	Long: `A longer description that spans multiple lines and likely contains examples
	and usage of using your command. For example:

	Cobra is a CLI library for Go that empowers applications.
	This application is a tool to generate the needed files
	to quickly create a Cobra application.`,

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

		folderName, err := cmd.Flags().GetString(folderUploadFlag[0])
		if err != nil {
			return err
		} else if folderName == "" {
			return errors.New("folder name not provided")
		}

		return deploy(environmentName, serviceName, folderName)
	},
}

func deploy(environmentName string, serviceName string, folderPath string) error {
	log.Printf("called deploy with: %s %s %s\n", environmentName, serviceName, folderPath)
	fd, err := ioutil.TempFile("", "haiku-cli-")
	if err != nil {
		return err
	}

	log.Printf("archiving...")
	err = ga.GzipCompress(folderPath, fd, ga.ArchiveGitRepo())
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
	conn, err := newConnection()
	if err != nil {
		return err
	}

	defer conn.Close()
	cliClient := pb.NewCliServiceClient(conn)
	ctx := context.Background()
	var trailer metadata.MD
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
	stream, err := cliClient.DeployUrl(ctx, &pb.DeployUrlRequest{
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
	stringP(deployCmd, folderUploadFlag)
	stringP(deployCmd, environmentFlag)
}
