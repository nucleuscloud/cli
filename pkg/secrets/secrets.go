package secrets

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/mhelmich/keycloak"
	"gopkg.in/yaml.v2"
)

const (
	secretsPath = "./nucleus-secrets.yaml"
)

type NucleusSecrets struct {
	Secrets map[string]map[string]string `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

func UpsertNucleusSecrets() error {
	_, err := ioutil.ReadFile(secretsPath)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// File doesn't exist yet, let's create it
	err = ioutil.WriteFile(secretsPath, []byte{}, 0644)
	if err != nil {
		return err
	}
	return nil
}

func getSecrets() (*NucleusSecrets, error) {
	file, err := ioutil.ReadFile(secretsPath)
	if err != nil {
		return nil, err
	}

	root := NucleusSecrets{}
	err = yaml.Unmarshal(file, &root)
	if err != nil {
		return nil, err
	}

	return &root, nil
}

func GetSecretsByEnvType(envType string) (map[string]string, error) {
	envType = strings.ToLower(envType)

	root, err := getSecrets()
	if err != nil {
		return nil, err
	}

	if root.Secrets == nil || root.Secrets[envType] == nil {
		return make(map[string]string), nil
	}
	return root.Secrets[envType], nil
}

func StoreSecret(publicKey string, secretKey string, secretValue string, envType string) error {
	envType = strings.ToLower(envType)

	root, err := getSecrets()
	if err != nil {
		return err
	}

	if root.Secrets == nil {
		root.Secrets = make(map[string]map[string]string)
	}

	if root.Secrets[envType] == nil {
		root.Secrets[envType] = make(map[string]string)
	}

	root.Secrets[envType][secretKey] = secretValue

	newBlob, err := yaml.Marshal(root)
	if err != nil {
		return err
	}

	store, err := keycloak.GetStoreFromBytes(newBlob, keycloak.YAML)
	if err != nil {
		return err
	}

	err = store.EncryptSubtree(publicKey, "secrets", envType)
	if err != nil {
		return err
	}

	err = store.ToFile(secretsPath)
	if err != nil {
		return err
	}

	return nil
}
