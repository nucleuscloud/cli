package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"github.com/nucleuscloud/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type serviceCommands struct {
	BuildCommand string
	StartCommand string
	ServiceName  string
	ServiceType  string
	IsPrivate    bool
}

var createServiceCmd = &cobra.Command{
	Use: "create",
	Aliases: []string{
		"init",
	},
	Short: "Creates a yaml file that describes the service",
	Long:  `Utility command that walks you through the creation of the Nucleus manifest file. This allows you to call nucleus deploy, among other commands, and gives you definitive documentation of the representation of your service.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("This utility will walk you through creating a Nucleus service.\n\nIt creates a declarative configuration file that you can apply using Nucleus deploy once you're ready to deploy your service.\n\nSee `nucleus create help` for definitive documentation on these fields and exactly what they do.\n\nPress ^C at any time to quit.\n\n")

		defaultSpec, err := getDefaultSpec()
		if err != nil {
			return err
		}

		serviceQuestions := []*survey.Question{
			{
				Name: "serviceName",
				Prompt: &survey.Input{
					Message: "Service name:",
					Default: defaultSpec.ServiceName,
				},
				Transform: survey.ToLower,
				Validate: func(val interface{}) error {
					str := val.(string)
					if !utils.IsValidName(str) {
						return fmt.Errorf("The name you provided contains invalid characters. It can only contain alphanumeric characters and hyphens.")
					}
					return nil
				},
			},
			{
				Name: "serviceType",
				Prompt: &survey.Select{
					Message: "Select your service's runtime:",
					Options: utils.GetSupportedRuntimes(),
				},
				Validate: survey.Required,
			},
			{
				Name: "isPrivate",
				Prompt: &survey.Confirm{
					Message: "Is your service private?",
					Default: false,
				},
			},
		}

		// ask the question
		var svcCommands serviceCommands
		err = survey.Ask(serviceQuestions, &svcCommands, survey.WithIcons(func(icons *survey.IconSet) {
			icons.Question.Text = ">"
			icons.Question.Format = "white"
		}))
		if err != nil {
			return err
		}

		conn, err := utils.NewApiConnection(utils.ApiConnectionConfig{
			AuthBaseUrl:  utils.Auth0BaseUrl,
			AuthClientId: utils.Auth0ClientId,
			ApiAudience:  utils.ApiAudience,
		})
		if err != nil {
			return err
		}
		defer conn.Close()
		//retrieve the default build and start commands based on runtime
		cliClient := pb.NewCliServiceClient(conn)
		// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
		var trailer metadata.MD
		defaultBuildStartCommands, err := cliClient.BuildStartCommands(context.Background(), &pb.DefaultBuildStartCommandsRequest{
			Runtime: svcCommands.ServiceType,
		},
			grpc.Trailer(&trailer),
		)
		if err != nil {
			return err
		}

		bc := defaultBuildStartCommands.BuildCommand
		sc := defaultBuildStartCommands.StartCommand

		err = runtimeQuestions(&svcCommands, bc, sc)
		if err != nil {
			return err
		}

		nucleusConfig := config.NucleusConfig{
			CliVersion: "nucleus-cli/v1alpha1",
			Spec: config.SpecStruct{
				ServiceName:    svcCommands.ServiceName,
				ServiceRunTime: svcCommands.ServiceType,
				BuildCommand:   svcCommands.BuildCommand,
				StartCommand:   svcCommands.StartCommand,
				IsPrivate:      svcCommands.IsPrivate,
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
	defaultDir := strings.ReplaceAll(strings.ToValidUTF8(strings.ToLower(filepath.Base(wd)), ""), "_", "-")
	return defaultDir, nil
}

func runtimeQuestions(svcCommands *serviceCommands, bc string, sc string) error {
	commands := []*survey.Question{
		{
			Name: "buildCommand",
			Prompt: &survey.Input{
				Message: "Build command:",
				Default: bc,
			},
			Transform: survey.ToLower,
		},
		{
			Name: "startCommand",
			Prompt: &survey.Input{
				Message: "Start command:",
				Default: sc,
			},
			Transform: survey.ToLower,
		},
	}

	err := survey.Ask(commands, svcCommands, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = ">"
		icons.Question.Format = "white"
	}))
	return err
}

func init() {
	rootCmd.AddCommand(createServiceCmd)
}
