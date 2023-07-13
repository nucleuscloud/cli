package projecttoml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	project1FilePath = "./fixtures/project1.toml"
	invalidFilePath  = "./fixtures/idontexist.toml"
)

func TestGetProjectFile(t *testing.T) {
	project, err := GetProjectFile(project1FilePath)
	assert.Nil(t, err)
	assert.Equal(t, project, &ProjectToml{
		Io: Io{
			Buildpacks: Buildpacks{
				Build: Build{
					Env: []map[string]string{
						{"name": "HELLO", "value": "FOO"},
						{"name": "HELLO2", "value": "WORLD2"},
						{"name": "HELLO3", "value": "WORLD3"},
					},
				},
			},
		},
		Build: Build{
			Env: []map[string]string{
				{"name": "build", "value": "value"},
				{"name": "HELLO3", "value": "WORLD3_OVERRIDE"},
			},
		},
	})
}

func TestGetBuildEnvsFromFile(t *testing.T) {
	file, err := GetProjectFile(project1FilePath)
	assert.Nil(t, err)

	vals, err := GetBuildEnvVars(file)
	assert.Nil(t, err)
	assert.Equal(
		t,
		vals,
		map[string]string{
			"HELLO":  "FOO",
			"HELLO2": "WORLD2",
			"build":  "value",
			"HELLO3": "WORLD3_OVERRIDE",
		},
	)
}

func TestDoesProjectFileExist(t *testing.T) {
	assert.True(t, DoesProjectFileExist(project1FilePath))
	assert.False(t, DoesProjectFileExist(invalidFilePath))
}

func TestGetBuildEnvVars(t *testing.T) {
	vals, err := GetBuildEnvVars(buildFakeProjectToml(nil, nil))
	assert.Nil(t, err)
	assert.Empty(t, vals)

	vals, err = GetBuildEnvVars(nil)
	assert.Nil(t, err)
	assert.Empty(t, vals)

	vals, err = GetBuildEnvVars(buildFakeProjectToml([]map[string]string{
		{"name": "foo", "value": "bar"},
	}, nil))
	assert.Nil(t, err)
	assert.Equal(t, vals, map[string]string{"foo": "bar"})

	vals, err = GetBuildEnvVars(buildFakeProjectToml(
		[]map[string]string{
			{"name": "foo", "value": "bar"},
		},
		[]map[string]string{
			{"name": "bar", "value": "baz"},
		},
	))
	assert.Nil(t, err)
	assert.Equal(t, vals, map[string]string{"foo": "bar", "bar": "baz"}, "should combine both env var sets")

	vals, err = GetBuildEnvVars(buildFakeProjectToml(
		[]map[string]string{
			{"name": "foo", "value": "bar"},
		},
		[]map[string]string{
			{"name": "foo", "value": "baz"},
		},
	))
	assert.Nil(t, err)
	assert.Equal(t, vals, map[string]string{"foo": "baz"}, "build envs should override buildpack envs")
}

func buildFakeProjectToml(
	bpEvs []map[string]string,
	buildEvs []map[string]string,
) *ProjectToml {
	return &ProjectToml{
		Io: Io{
			Buildpacks: Buildpacks{
				Build: Build{
					Env: bpEvs,
				},
			},
		},
		Build: Build{
			Env: buildEvs,
		},
	}
}
