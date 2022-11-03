package procfile

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Procfile struct {
	Web string `yaml:"web"`
}

const (
	procfilePath = "./Procfile"
)

func DoesProcfileExist() bool {
	_, err := os.Stat(procfilePath)
	return !errors.Is(err, os.ErrNotExist)
}

func GetProcfile() (*Procfile, error) {
	file, err := os.ReadFile(procfilePath)
	if err != nil {
		return nil, err
	}

	data := &Procfile{}
	err = yaml.Unmarshal(file, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func SetProcfile(file *Procfile) error {
	data, err := yaml.Marshal(&file)
	if err != nil {
		return err
	}

	err = os.WriteFile(procfilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("unable to write data into procfile")
	}
	return nil
}
