package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goombaio/namegenerator"
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

		envName := cliPrompt("Environment name: "+"("+defaultSpec.EnvironmentName+")", defaultSpec.EnvironmentName)
		servName := cliPrompt("Service name: "+"("+defaultSpec.ServiceName+")", defaultSpec.ServiceName)
		serType := cliPrompt("Service runtime (fastapi,nodejs):", "")

		if serType != "fastapi" && serType != "nodejs" {
			return errors.New("unsupported service type")
		}

		nucleusConfig := ConfigYaml{
			CliVersion: "nucleus-cli/v1alpha1",
			Spec: SpecStruct{
				EnvironmentName: envName,
				ServiceName:     servName,
				ServiceRunTime:  serType,
			},
		}
		err = setNucleusConfig(&nucleusConfig)

		if err != nil {
			return errors.New("unable to write data into the file")
		}

		return nil
	},
}

func getDefaultSpec() (*SpecStruct, error) {
	spec := SpecStruct{}
	spec.EnvironmentName = getDefaultEnvironmentName()

	defaultServiceName, err := getDefaultServiceName()

	if err != nil {
		return nil, err
	}

	spec.ServiceName = defaultServiceName

	return &spec, nil
}

func getDefaultEnvironmentName() string {
	//there are limited number of unique environment names that we can create here something like 3,700
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)
	return nameGenerator.Generate()
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
