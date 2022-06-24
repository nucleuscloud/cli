package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/nucleuscloud/cli/internal/pkg/config"
)

func getSecretsFromSpec(spec *config.SpecStruct) config.NucleusSecrets {
	if spec == nil {
		spec = &config.SpecStruct{}
	}
	secrets := spec.Secrets
	if secrets == nil {
		secrets = map[string]map[string]string{}
	}
	return secrets
}

func GetSecretsByEnvType(spec *config.SpecStruct, envType string) map[string]string {
	envType = strings.ToLower(envType)

	root := getSecretsFromSpec(spec)

	if root == nil || root[envType] == nil {
		return map[string]string{}
	}
	return root[envType]
}

func StoreSecret(spec *config.SpecStruct, publicKeyBytes []byte, secretKey string, secretValue string, envType string) error {
	envType = strings.ToLower(envType)

	if spec.Secrets == nil {
		spec.Secrets = map[string]map[string]string{}
	}
	if spec.Secrets[envType] == nil {
		spec.Secrets[envType] = map[string]string{}
	}

	secrets := GetSecretsByEnvType(spec, envType)

	if secrets == nil {
		secrets = map[string]string{}
	}

	publicKey, err := parseRsaPublicKey(publicKeyBytes)
	if err != nil {
		return err
	}

	ciphertextBytes, err := encryptWithPublicKey([]byte(secretValue), publicKey)
	if err != nil {
		return err
	}

	secrets[secretKey] = base64.StdEncoding.EncodeToString(ciphertextBytes)

	spec.Secrets[envType] = secrets
	return nil
}

func parseRsaPublicKey(pubKey []byte) (*rsa.PublicKey, error) {
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
