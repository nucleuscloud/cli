package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type NucleusConfig struct {
	CliVersion string     `yaml:"cliVersion"`
	Spec       SpecStruct `yaml:"spec"`
}

type NucleusSecrets = map[string]map[string]string

type ResourceRequirements struct {
	Minimum ResourceList `yaml:"minimum,omitempty"`
	Maximum ResourceList `yaml:"maximum,omitempty"`
}

type ResourceList struct {
	Cpu    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

type SpecStruct struct {
	ServiceName        string               `yaml:"serviceName"`
	ServiceRunTime     string               `yaml:"serviceRuntime"`
	Image              string               `yaml:"image,omitempty"`
	IsPrivate          bool                 `yaml:"isPrivate"`
	Vars               map[string]string    `yaml:"vars,omitempty"`
	Secrets            NucleusSecrets       `yaml:"secrets,omitempty"`
	AllowedServices    []string             `yaml:"allowedServices,omitempty"`
	DisallowedServices []string             `yaml:"disallowedServices,omitempty"`
	Resources          ResourceRequirements `yaml:"resources,omitempty"`
}

type NucleusAuthConfig struct {
	AccessToken  string `yaml:"accessToken"`
	RefreshToken string `yaml:"refreshToken,omitempty"`
	IdToken      string `yaml:"idToken,omitempty"`
}

const (
	nucleusConfigPath = "./nucleus.yaml"
	nucleusFolderName = ".nucleus"
)

var (
	ErrMustLogin = fmt.Errorf("error retrieving auth information. Try logging in via 'nucleus login'")
)

func DoesNucleusConfigExist() bool {
	_, err := os.Stat(nucleusConfigPath)
	return !errors.Is(err, os.ErrNotExist)
}

// Retrieves the nucleus config defined by the user
func GetNucleusConfig() (*NucleusConfig, error) {
	yamlFile, err := os.ReadFile(nucleusConfigPath)
	if err != nil {
		return nil, err
	}

	yamlData := NucleusConfig{}
	err = yaml.Unmarshal(yamlFile, &yamlData)

	if err != nil {
		return nil, err
	}

	return &yamlData, nil
}

// Sets the nucleus config defined by the user
func SetNucleusConfig(config *NucleusConfig) error {
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	err = os.WriteFile(nucleusConfigPath, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("Unable to write data into the config file")
	}
	return nil
}

// Get or Creates the Nucleus folder that lives in the homedir and stores persisted settings
func GetOrCreateNucleusFolder() (string, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	fullName := dirname + "/" + nucleusFolderName

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

// Gets the nucleus auth config
func GetNucleusAuthConfig() (*NucleusAuthConfig, error) {
	dirPath, err := GetOrCreateNucleusFolder()
	if err != nil {
		return nil, err
	}

	fileName := dirPath + "/auth.yaml"

	file, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Auth file doesnt exist. User has not logged in.\n", err)
		return nil, ErrMustLogin
	}

	var auth *NucleusAuthConfig
	err = yaml.Unmarshal(file, &auth)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Auth config is not in correct format.\n", err)
		return nil, ErrMustLogin
	}
	return auth, nil
}

func SetNucleusAuthFile(authConfig NucleusAuthConfig) error {
	dirPath, err := GetOrCreateNucleusFolder()
	if err != nil {
		return err
	}

	if dirPath == "" {
		return fmt.Errorf("Could not find the correct nucleus dir to store configs")
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

func ClearNucleusAuthFile() error {
	dirPath, err := GetOrCreateNucleusFolder()
	if err != nil {
		return err
	}

	if dirPath == "" {
		return fmt.Errorf("Could not find the correct nucleus dir to store configs")
	}

	fileName := dirPath + "/auth.yaml"

	err = os.Remove(fileName)
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}
