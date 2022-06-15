package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nucleuscloud/cli/pkg/config"
	"github.com/spf13/cobra"
)

var createServiceCmd = &cobra.Command{
	Use:   "create",
	Short: "creates a yaml file that describes the service",
	Long:  `creates a yaml file that describes the service.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		fmt.Println("This utility will walk you through creating a Nucleus service.\n\nIt creates a declarative configuration file that you can apply using Nucleus deploy once you're ready to deploy your service.\n\nSee `nucleus create help` for definitive documentation on these fields and exactly what they do.\n\nPress ^C at any time to quit.")

		defaultSpec, err := getDefaultSpec()

		if err != nil {
			return err
		}

		servName := cliPrompt("Service name: "+"("+defaultSpec.ServiceName+")", defaultSpec.ServiceName)
		serType := cliPrompt("Service runtime (fastapi,nodejs):", "")

		if serType != "fastapi" && serType != "nodejs" {
			return errors.New("unsupported service type")
		}

		nucleusConfig := config.NucleusConfig{
			CliVersion: "nucleus-cli/v1alpha1",
			Spec: config.SpecStruct{
				ServiceName:    servName,
				ServiceRunTime: serType,
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

func init() {
	rootCmd.AddCommand(createServiceCmd)
}
