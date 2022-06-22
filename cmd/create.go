package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/pkg/auth"
	"github.com/nucleuscloud/cli/pkg/config"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var ServiceCommands = struct {
	BuildCommand string
	StartCommand string
	ServiceName  string
	ServiceType  string
}{}

var createServiceCmd = &cobra.Command{
	Use:   "create",
	Short: "creates a yaml file that describes the service",
	Long:  `creates a yaml file that describes the service.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		fmt.Println("This utility will walk you through creating a Nucleus service.\n\nIt creates a declarative configuration file that you can apply using Nucleus deploy once you're ready to deploy your service.\n\nSee `nucleus create help` for definitive documentation on these fields and exactly what they do.\n\nPress ^C at any time to quit.")

		fmt.Print("\n")

		defaultSpec, err := getDefaultSpec()
		if err != nil {
			return err
		}

		var serviceQuestions = []*survey.Question{
			{
				Name: "serviceName",
				Prompt: &survey.Input{
					Message: "Service name: " + "(" + defaultSpec.ServiceName + ")",
				},
				Transform: survey.Title,
				Validate: func(val interface{}) error {
					str := val.(string)
					if str != "" {
						lowerStr := strings.ToLower(str)
						if !isValidName(lowerStr) {
							return fmt.Errorf("Your service's custom name can only contain alphanumeric characters and hyphens.")
						}
					}
					return nil
				},
			},
			{
				Name: "serviceType",
				Prompt: &survey.Select{
					Message: "Select your service's runtime:",
					Options: []string{
						"nodejs",
						"fastapi",
						"golang",
					},
				},
				Validate: survey.Required,
			},
		}

		// ask the question
		err = survey.Ask(serviceQuestions, &ServiceCommands, survey.WithIcons(func(icons *survey.IconSet) {
			icons.Question.Text = ">"
			icons.Question.Format = "white"
		}))

		if err != nil {
			return err
		}

		if ServiceCommands.ServiceName == "" {
			ServiceCommands.ServiceName = defaultSpec.ServiceName
		}

		//refactor these clients into a utils file later
		authClient, err := auth.NewAuthClient(auth0BaseUrl, auth0ClientId, apiAudience)
		if err != nil {
			return err
		}
		unAuthConn, err := newConnection()
		if err != nil {
			return err
		}
		unAuthCliClient := pb.NewCliServiceClient(unAuthConn)
		accessToken, err := config.GetValidAccessTokenFromConfig(authClient, unAuthCliClient)
		unAuthConn.Close()
		if err != nil {
			return err
		}

		conn, err := newAuthenticatedConnection(accessToken)
		if err != nil {
			return err
		}

		defer conn.Close()
		//retrieve the default build and start commands based on runtime
		cliClient := pb.NewCliServiceClient(conn)
		// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
		var trailer metadata.MD
		defaultBuildStartCommands, err := cliClient.BuildStartCommands(context.Background(), &pb.DefaultBuildStartCommandsRequest{
			Runtime: ServiceCommands.ServiceType,
		},
			grpc.Trailer(&trailer),
		)
		if err != nil {
			return err
		}

		bc := defaultBuildStartCommands.BuildCommand
		sc := defaultBuildStartCommands.StartCommand

		err = runtimeQuestions(bc, sc)
		if err != nil {
			return err
		}

		nucleusConfig := config.NucleusConfig{
			CliVersion: "nucleus-cli/v1alpha1",
			Spec: config.SpecStruct{
				ServiceName:    strings.ToLower(ServiceCommands.ServiceName),
				ServiceRunTime: strings.ToLower(ServiceCommands.ServiceType),
				BuildCommand:   strings.ToLower(ServiceCommands.BuildCommand),
				StartCommand:   strings.ToLower(ServiceCommands.StartCommand),
			},
		}
		err = config.SetNucleusConfig(&nucleusConfig)

		if err != nil {
			return errors.New("unable to write data into the file")
		}

		return nil
	},
}

func getDefaultSpec() (*config.SpecStruct, error) {
	spec := config.SpecStruct{}
	defaultServiceName, err := getDefaultServiceName()

	if err != nil {
		return nil, err
	}

	spec.ServiceName = defaultServiceName

	return &spec, nil
}

func getDefaultServiceName() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defaultDir := filepath.Base(wd)
	return defaultDir, nil
}

func cliPrompt(label string, defaultEnv string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, label+" ")
	s, _ = r.ReadString('\n')
	if s == "\n" {
		s = defaultEnv
	}
	return strings.TrimSpace(s)
}

func runtimeQuestions(bc string, sc string) error {

	var commands = []*survey.Question{
		{
			Name: "buildCommand",
			Prompt: &survey.Input{
				Message: "Press enter for default build command -> " + bc + ", or type in custom build command:",
			},
			Transform: survey.Title,
		},
		{
			Name: "startCommand",
			Prompt: &survey.Input{
				Message: "Press enter for default start command -> " + sc + ", or type in custom start command:",
			},
			Transform: survey.Title,
		},
	}

	err := survey.Ask(commands, &ServiceCommands, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = ">"
		icons.Question.Format = "white"
	}))
	return err
}

func init() {
	rootCmd.AddCommand(createServiceCmd)
}
