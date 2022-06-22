package config

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/pkg/auth"
	"gopkg.in/yaml.v2"
)

type NucleusConfig struct {
	CliVersion string     `yaml:"cliVersion"`
	Spec       SpecStruct `yaml:"spec"`
}

type SpecStruct struct {
	ServiceName    string            `yaml:"serviceName"`
	ServiceRunTime string            `yaml:"serviceRuntime"`
	BuildCommand   string            `yaml:"buildCommand,omitempty"`
	StartCommand   string            `yaml:"startCommand,omitempty"`
	IsPrivate      bool              `yaml:"isPrivate"`
	Vars           map[string]string `yaml:"vars,omitempty"`
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
	ErrMustLogin = errors.New("error retrieving auth information. Try logging in via 'nucleus login'")
)

// Retrieves the nucleus config defined by the user
func GetNucleusConfig() (*NucleusConfig, error) {
	// TODO(marco): make it so that parent dirs are recursively searched
	yamlFile, err := ioutil.ReadFile(nucleusConfigPath)
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

	err = ioutil.WriteFile(nucleusConfigPath, yamlData, 0644)
	if err != nil {
		return errors.New("Unable to write data into the config file")
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

	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Println("Auth file doesnt exist. User has not logged in.\n", err)
		return nil, ErrMustLogin
	}

	var auth *NucleusAuthConfig
	err = yaml.Unmarshal(file, &auth)
	if err != nil {
		fmt.Println("Auth config is not in correct format.\n", err)
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

func ClearNucleusAuthFile() error {
	dirPath, err := GetOrCreateNucleusFolder()
	if err != nil {
		return err
	}

	if dirPath == "" {
		return errors.New("Could not find the correct nucleus dir to store configs")
	}
	fileName := dirPath + "/auth.yaml"
	return os.Remove(fileName)
}

// Retrieves the access token from the config and validates it.
func GetValidAccessTokenFromConfig(authClient auth.AuthClientInterface, nucleusClient pb.CliServiceClient) (string, error) {
	config, err := GetNucleusAuthConfig()
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	err = authClient.ValidateToken(ctx, config.AccessToken)
	if err != nil {
		log.Println("Access token is no longer valid. Attempting to refresh...")
		if config.RefreshToken != "" {
			reply, err := nucleusClient.RefreshAccessToken(ctx, &pb.RefreshAccessTokenRequest{
				RefreshToken: config.RefreshToken,
			})
			// refreshResponse, err := authClient.GetRefreshedAccessToken(config.RefreshToken)
			if err != nil {
				err = ClearNucleusAuthFile()
				if err != nil {
					return "", err
				}
				return "", errors.New("unable to refresh token, please try logging in again.")
			}
			var newRefreshToken string
			if reply.RefreshToken != "" {
				newRefreshToken = reply.RefreshToken
			} else {
				newRefreshToken = config.RefreshToken
			}
			err = SetNucleusAuthFile(NucleusAuthConfig{
				AccessToken:  reply.AccessToken,
				RefreshToken: newRefreshToken,
				IdToken:      reply.IdToken,
			})
			if err != nil {
				log.Println("Successfully refreshed token, but was unable to update nucleus auth file")
				return "", err
			}
			return reply.AccessToken, authClient.ValidateToken(ctx, reply.AccessToken)
		}
	}
	return config.AccessToken, authClient.ValidateToken(ctx, config.AccessToken)
}
