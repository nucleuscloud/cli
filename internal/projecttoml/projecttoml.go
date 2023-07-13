package projecttoml

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type ProjectToml struct {
	Io    Io
	Build Build
}

type Io struct {
	Buildpacks Buildpacks
}
type Buildpacks struct {
	Build Build
}
type Build struct {
	Env []map[string]string
}

const (
	projectTomlPath = "./project.toml"
)

func DoesProjectFileExist() bool {
	_, err := os.Stat(projectTomlPath)
	return !errors.Is(err, os.ErrNotExist)
}

func GetBuildEnvVars(project *ProjectToml) (map[string]string, error) {
	buildEvs := map[string]string{}
	if project == nil {
		return buildEvs, nil
	}

	for _, env := range project.Io.Buildpacks.Build.Env {
		name, ok := env["name"]
		if !ok {
			return nil, fmt.Errorf("io.buildpacks.build.env missing 'name' key")
		}
		value, ok := env["value"]
		if !ok {
			return nil, fmt.Errorf("io.buildpacks.build.env.[%s] missing 'value' key", name)
		}
		buildEvs[name] = value
	}
	for _, env := range project.Build.Env {
		name, ok := env["name"]
		if !ok {
			return nil, fmt.Errorf("build.env missing 'name' key")

		}
		value, ok := env["value"]
		if !ok {
			return nil, fmt.Errorf("build.env.[%s] missing 'value' key", name)
		}
		buildEvs[name] = value
	}

	return buildEvs, nil
}

func GetProjectFile() (*ProjectToml, error) {
	file, err := os.ReadFile(projectTomlPath)
	if err != nil {
		return nil, err
	}

	data := &ProjectToml{}
	err = toml.Unmarshal(file, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
