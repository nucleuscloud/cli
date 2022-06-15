package config

import (
	"errors"
	"io/ioutil"
	"os"
)

func UpsertNucleusSecrets() error {
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
