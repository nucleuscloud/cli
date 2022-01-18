package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var createServiceCmd = &cobra.Command{
	Use:   "create",
	Short: "creates a yaml file that describes the service",
	Long:  `creates a yaml file that describes the service.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		fmt.Println("This utility will walk you through creating a Haiku service.\n\nIt creates a declarative configuration file that you can apply using Haiku deploy once you're ready to deploy your service.\n\nSee `haiku create help` for definitive documentation on these fields and exactly what they do.\n\nPress ^C at any time to quit.\n\n")

		envName := cliPrompt("Environment name: ")
		servName := cliPrompt("Service name: ")
		dirName := cliPrompt("Folder name: ")
		serType := cliPrompt("Service runtime: ")

		configfileName := envName + "_" + servName + "_" + "config.yaml"
		yamlData, err := createYamlConfig("create", envName, servName, dirName, serType)
		if err != nil {
			panic("Unable to write data into the file")
		}
		err = ioutil.WriteFile(configfileName, yamlData, 0644)
		if err != nil {
			panic("Unable to write data into the file")
		}

		return nil
	},
}

func cliPrompt(label string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, label+" ")
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}

func createYamlConfig(action string, envName string, servName string, dirPath string, runtime string) ([]byte, error) {

	type SpecStruct struct {
		EnvironmentName string `yaml:"environmentName"`
		ServiceName     string `yaml:"serviceName"`
		DirectoryPath   string `yaml:"directoryPath"`
		ServiceRunTime  string `yaml:"serviceRuntime"`
	}

	type ConfigYaml struct {
		CliVersion string     `yaml:"cliVersion"`
		Action     string     `yaml:"action"`
		Spec       SpecStruct `yaml:"spec"`
	}

	y := ConfigYaml{
		CliVersion: "haiku-cli/v1",
		Action:     action,
		Spec: SpecStruct{
			EnvironmentName: envName,
			ServiceName:     servName,
			DirectoryPath:   dirPath,
			ServiceRunTime:  runtime,
		},
	}

	yamlData, err := yaml.Marshal(&y)

	if err != nil {
		fmt.Printf("Error while Marshaling. %v", err)
	}

	return yamlData, err
}

func init() {
	rootCmd.AddCommand(createServiceCmd)
}
