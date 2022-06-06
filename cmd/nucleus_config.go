package cmd

import (
	"errors"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type SpecStruct struct {
	EnvironmentName string            `yaml:"environmentName"`
	ServiceName     string            `yaml:"serviceName"`
	ServiceRunTime  string            `yaml:"serviceRuntime"`
	IsPrivate       bool              `yaml:"isPrivate"`
	Vars            map[string]string `yaml:"vars,omitempty"`
}

type ConfigYaml struct {
	CliVersion string     `yaml:"cliVersion"`
	Spec       SpecStruct `yaml:"spec"`
}

func getNucleusConfig() (*ConfigYaml, error) {
	// TODO(marco): make it so that parent dirs are recursively searched
	yamlFile, err := ioutil.ReadFile("./nucleus.yaml")
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

func setNucleusConfig(config *ConfigYaml) error {
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("./nucleus.yaml", yamlData, 0644)
	if err != nil {
		return errors.New("Unable to write data into the config file")
	}
	return nil
}

func upsertNucleusSecrets() error {
	_, err := ioutil.ReadFile("/nucleus-secrets.yaml")

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// File doesn't exist yet, let's create it
	err = ioutil.WriteFile("./nucleus-secrets.yaml", []byte{}, 0644)
	if err != nil {
		return err
	}
	return nil
}
