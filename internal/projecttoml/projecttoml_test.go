package projecttoml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
