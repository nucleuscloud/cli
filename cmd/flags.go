package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

type Param []string

var (
	// haiku init
	environmentFlag = Param{"environment", "e", "", "-e my-environment"}

	// haiku docker login
	dockerServerFlag   = Param{"server", "s", "", "-s docker.io"}
	dockerUsernameFlag = Param{"username", "u", "", "-u julie"}
	dockerPasswordFlag = Param{"password", "p", "", "-p s3cr3t"}
	dockerEmailFlag    = Param{"email", "m", "", "-m julie@haiku.io"}

	// haiku deploy
	imageFlag        = Param{"image", "i", "", "ghcr.io/andrea/my-service:latest"}
	serviceNameFlag  = Param{"service-name", "s", "", "-s my-service"}
	serviceTypeFlag  = Param{"service-type", "t", "", "-t service-type"}
	folderUploadFlag = Param{"folder", "f", ".", "-f folder-to-upload"}

	//haiku listEnv is environmentFlag & serviceFlag
)

func stringP(cmd *cobra.Command, param Param) {
	panicIfWrongSize(param)
	cmd.Flags().StringP(param[0], param[1], param[2], param[3])
}

func panicIfWrongSize(a []string) {
	if len(a) != 4 {
		log.Fatalf("no bueno")
	}
}

func getParamString(cmd *cobra.Command, param Param) (string, error) {
	strParam, err := cmd.Flags().GetString(param[0])
	if err != nil {
		return "", err
	} else if strParam == "" {
		return "", fmt.Errorf("%s not provided", param[0])
	}

	return strParam, nil
}
