package cmd

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"

	"github.com/nucleuscloud/cli/pkg/auth"
	"gopkg.in/yaml.v2"
)

type SpecStruct struct {
	ServiceName    string            `yaml:"serviceName"`
	ServiceRunTime string            `yaml:"serviceRuntime"`
	IsPrivate      bool              `yaml:"isPrivate"`
	Vars           map[string]string `yaml:"vars,omitempty"`
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

func clearNucleusAuthFile() error {
	dirPath, err := upsertNucleusFolder()
	if err != nil {
		return err
	}

	if dirPath == "" {
		return errors.New("Could not find the correct nucleus dir to store configs")
	}
	fileName := dirPath + "/auth.yaml"
	return os.Remove(fileName)
}

/**
 * Retrieves the access token from the config and validates it.
 */
func getValidAccessTokenFromConfig(authClient auth.AuthClientInterface) (string, error) {
	config, err := getNucleusAuthConfig()
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	err = authClient.ValidateToken(ctx, config.AccessToken)
	if err != nil {
		log.Println("Access token is no longer valid. Attempting to refresh...")
		if config.RefreshToken != "" {
			refreshResponse, err := authClient.GetRefreshedAccessToken(config.RefreshToken)
			if err != nil {
				err = clearNucleusAuthFile()
				if err != nil {
					return "", err
				}
				return "", errors.New("unable to refresh token, please try logging in again.")
			}
			err = setNucleusAuthFile(NucleusAuth{
				AccessToken:  refreshResponse.AccessToken,
				RefreshToken: config.RefreshToken,
				IdToken:      refreshResponse.IdToken,
			})
			if err != nil {
				log.Println("Successfully refreshed token, but was unable to update nucleus auth file")
				return "", err
			}
			return refreshResponse.AccessToken, authClient.ValidateToken(ctx, refreshResponse.AccessToken)
		}
	}
	return config.AccessToken, authClient.ValidateToken(ctx, config.AccessToken)
}
