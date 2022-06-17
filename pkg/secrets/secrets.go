package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"os"
	"strings"

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

func StoreSecret(publicKeyBytes []byte, secretKey string, secretValue string, envType string) error {
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

	publicKey, err := parseRsaPublicKey(publicKeyBytes)
	if err != nil {
		return err
	}
	ciphertextBytes, err := encryptWithPublicKey([]byte(secretValue), publicKey)
	if err != nil {
		return err
	}
	root.Secrets[envType][secretKey] = base64.StdEncoding.EncodeToString(ciphertextBytes)

	newBlob, err := yaml.Marshal(root)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(secretsPath, newBlob, 0777)
	if err != nil {
		return err
	}

	return nil
}

func parseRsaPublicKey(pubKey []byte) (*rsa.PublicKey, error) {
	// block, _ := pem.Decode([]byte(pubPEM))
	// if block == nil {
	// 	return nil, errors.New("failed to parse PEM block containing the key")
	// }
	pub, err := x509.ParsePKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, errors.New("Key type is not RSA")
}

func encryptWithPublicKey(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, msg, nil)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}
