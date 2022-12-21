package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	svcmgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/servicemgmt/v1alpha1"
	"github.com/spf13/cobra"

	"github.com/nucleuscloud/cli/internal/config"
	"github.com/nucleuscloud/cli/internal/procfile"
	"github.com/nucleuscloud/cli/internal/utils"
)

type serviceCommands struct {
	BuildCommand string
	StartCommand string
	ServiceName  string
	ServiceType  string
	IsPrivate    bool
	DockerImage  string
}

var (
	surveyIcons = survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = ">"
		icons.Question.Format = "white"
	})
)

func isGolang(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	return false
}

func isNodejs(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		return true
	}
	return false
}

func isPython(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		return true
	}
	// poetry and others
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		return true
	}
	return false
}

func isDocker(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err == nil {
		return true
	}
	return false
}

func guessProjectType() string {
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetSupportedRuntimes()[0]

	}
	if isGolang(cwd) {
		return "go"
	}
	if isNodejs(cwd) {
		return "nodejs"
	}
	if isPython(cwd) {
		return "python"
	}
	if isDocker(cwd) {
		return "docker"
	}
	return utils.GetSupportedRuntimes()[0]
}

var createServiceCmd = &cobra.Command{
	Use: "create",
	Aliases: []string{
		"init",
	},
	Short: "Creates a yaml configuration file required for deploying the service",
	Long:  `Utility command that walks you through the creation of the Nucleus manifest file. This allows you to call nucleus deploy, among other commands, and gives you definitive documentation of the representation of your service.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
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
					Default: guessProjectType(),
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
		err = survey.Ask(serviceQuestions, &svcCommands, surveyIcons)
		if err != nil {
			return err
		}

		if svcCommands.ServiceType == "docker" {
			err = survey.Ask([]*survey.Question{
				{
					Name: "dockerImage",
					Prompt: &survey.Input{
						Message: "Docker Image URL:",
					},
					Validate: func(imageUrl interface{}) error {
						imageUrl, ok := imageUrl.(string)
						if !ok {
							return err
						}
						if imageUrl == "" {
							return fmt.Errorf("docker image URL must be specified")
						}
						return nil
					},
				},
			}, &svcCommands, surveyIcons)
			if err != nil {
				return err
			}
		} else if svcCommands.ServiceType != "python" {
			conn, err := utils.NewApiConnectionByEnv(ctx, utils.GetEnv())
			if err != nil {
				return err
			}
			defer conn.Close()
			//retrieve the default build and start commands based on runtime
			var bc string
			var sc string

			cliClient := svcmgmtv1alpha1.NewServiceMgmtServiceClient(conn)
			defaultBuildStartCommands, err := cliClient.GetDefaultBuildStartCommands(ctx, &svcmgmtv1alpha1.GetDefaultBuildStartCommandsRequest{
				Runtime: svcCommands.ServiceType,
			})
			if err != nil {
				return err
			}
			bc = defaultBuildStartCommands.BuildCommand
			sc = defaultBuildStartCommands.StartCommand

			err = runtimeQuestions(&svcCommands, bc, sc)
			if err != nil {
				return err
			}
		} else if svcCommands.ServiceType == "python" {
			err = ensureProcfileExists()
			if err != nil {
				return err
			}
		}

		cliVersion := "nucleus-cli/v1alpha1"

		nucleusConfig := config.NucleusConfig{
			CliVersion: cliVersion,
			Spec: config.SpecStruct{
				ServiceName:    svcCommands.ServiceName,
				ServiceRunTime: svcCommands.ServiceType,
				Image:          svcCommands.DockerImage,
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

func ensureProcfileExists() error {
	// ask about proc file if it doesn't exist
	if !procfile.DoesProcfileExist() {
		var entrypoint string
		err := survey.AskOne(&survey.Input{
			Message: "What is the entrypoint to your web server?",
			Help:    "uvicorn main:app --host 0.0.0.0 --port $PORT",
		}, &entrypoint)
		if err != nil {
			return err
		}
		if entrypoint == "" {
			return fmt.Errorf("entrypoint length must be greater than 0")
		}
		err = procfile.SetProcfile(&procfile.Procfile{Web: entrypoint})
		if err != nil {
			return err
		}
	}
	return nil
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

	err := survey.Ask(commands, svcCommands, surveyIcons)
	return err
}

func init() {
	rootCmd.AddCommand(createServiceCmd)
}
