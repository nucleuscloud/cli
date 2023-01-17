package clienv

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/viper"

	"github.com/nucleuscloud/cli/internal/term"
)

type NucleusEnv string

const (
	nucleusDebugEnvKey = "NUCLEUS_DEBUG_ENV"

	ProdEnv  NucleusEnv = "prod"
	StageEnv NucleusEnv = "stage"
	DevEnv   NucleusEnv = "dev"
)

var (
	nucleusEnvMap = map[string]NucleusEnv{
		string(ProdEnv):  ProdEnv,
		string(StageEnv): StageEnv,
		string(DevEnv):   DevEnv,
	}

	hasLoggedAboutEnvType = false
)

func GetEnv() NucleusEnv {
	val := viper.GetString(nucleusDebugEnvKey)
	if val == "" {
		return ProdEnv
	}
	nucleusEnv, ok := parseEnvString(val)
	if !ok {
		panic(fmt.Errorf("%s can only be one of %s", nucleusDebugEnvKey, strings.Join(getAllowedEnvs(), ",")))
	}
	if !hasLoggedAboutEnvType {
		green := term.GetColoredSprintFunc(color.FgGreen)
		fmt.Println(green(nucleusDebugEnvKey, "=", val))
		hasLoggedAboutEnvType = true
	}
	return nucleusEnv
}

func IsDevEnv() bool {
	return GetEnv() == DevEnv
}
func IsStageEnv() bool {
	return GetEnv() == StageEnv
}
func IsProdEnv() bool {
	return GetEnv() == ProdEnv
}

func parseEnvString(str string) (NucleusEnv, bool) {
	v, ok := nucleusEnvMap[strings.ToLower(str)]
	return v, ok
}
func getAllowedEnvs() []string {
	envs := []string{}
	for nenv := range nucleusEnvMap {
		envs = append(envs, nenv)
	}
	return envs
}
