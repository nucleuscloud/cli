package cmd

import (
	"errors"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type SpecStruct struct {
	EnvironmentName string              `yaml:"environmentName"`
	ServiceName     string              `yaml:"serviceName"`
	ServiceRunTime  string              `yaml:"serviceRuntime"`
	IsPrivate       bool                `yaml:"isPrivate"`
	Env             map[string][]string `yaml:"env"`
	Secrets         map[string][]string `yaml:"secrets"`
}

type ConfigYaml struct {
	CliVersion string     `yaml:"cliVersion"`
	Spec       SpecStruct `yaml:"spec"`
}

func getHaikuConfig() (*ConfigYaml, error) {
	// TODO(marco): make it so that parent dirs are recursively searched
	yamlFile, err := ioutil.ReadFile("./haiku.yaml")
	if err != nil {
		return nil, err
	}

	yamlData := ConfigYaml{}
	err = yaml.Unmarshal(yamlFile, &yamlData)

	if err != nil {
		return nil, err
	}

	return &yamlData, nil
}

func upsertHaikuSecrets() error {
	_, err := ioutil.ReadFile("./haiku-secrets.yaml")

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// File doesn't exist yet, let's create it
	err = ioutil.WriteFile("./haiku-secrets.yaml", []byte{}, 0644)
	if err != nil {
		return err
	}
	return nil
}
