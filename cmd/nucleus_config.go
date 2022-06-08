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

func upsertNucleusFolder() (string, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	fullName := dirname + "/.nucleus"

	_, err = os.Stat(fullName)
	if os.IsNotExist(err) {
		err = os.Mkdir(fullName, 0755)
		if err != nil {
			if os.IsExist(err) {
				return fullName, nil
			}
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	return fullName, nil
}

type NucleusAuth struct {
	AccessToken  string `yaml:"accessToken"`
	RefreshToken string `yaml:"refreshToken,omitempty"`
	IdToken      string `yaml:"idToken,omitempty"`
}

func getNucleusAuthConfig() (*NucleusAuth, error) {
	dirPath, err := upsertNucleusFolder()
	if err != nil {
		return nil, err
	}

	fileName := dirPath + "/auth.yaml"

	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var auth *NucleusAuth
	err = yaml.Unmarshal(file, &auth)
	if err != nil {
		return nil, err
	}
	return auth, nil
}

func setNucleusAuthFile(authConfig NucleusAuth) error {
	dirPath, err := upsertNucleusFolder()
	if err != nil {
		return err
	}

	if dirPath == "" {
		return errors.New("Could not find the correct nucleus dir to store configs")
	}

	fileName := dirPath + "/auth.yaml"

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer file.Close()

	dataToWrite, err := yaml.Marshal(authConfig)
	if err != nil {
		return err
	}

	_, err = file.Write(dataToWrite)
	if err != nil {
		return err
	}
	return nil
}
