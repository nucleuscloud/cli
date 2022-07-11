package utils

import (
	"fmt"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	ErrInvalidServiceName = fmt.Errorf("invalid name")
	validNameMatcher      = regexp.MustCompile("^[a-z][a-z1-9-]*$").MatchString
)

// Auth Vars
var (
	Auth0ClientId string = "STljLBgOpW4fuwyKT30YWBsvnxyVAZkr"
	Auth0BaseUrl  string = "https://auth.stage.usenucleus.cloud"
	ApiAudience   string = "https://api.usenucleus.cloud"

	Scopes []string = []string{
		"openid",
		"profile",
		"offline_access",

		// custom
		"deploy:service",
		"read:service",
	}
)

func IsValidName(s string) bool {
	return validNameMatcher(s)
}

func GetGrpcTrailer() grpc.CallOption {
	// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
	var trailer metadata.MD
	return grpc.Trailer(&trailer)
}

func IsValidEnvironmentType(environmentType string) bool {
	return environmentType != "dev" && environmentType != "stage" && environmentType != "prod"
}

func CheckProdOk(cmd *cobra.Command, environmentType string, yesPromptFlagName string) error {
	yesPrompt, err := cmd.Flags().GetBool(yesPromptFlagName)
	if err != nil {
		return err
	}
	if !yesPrompt {
		shouldProceed := false
		err = survey.AskOne(&survey.Confirm{
			Message: "Are you sure you want to invoke this command in production?",
		}, &shouldProceed)
		if err != nil {
			return err
		}

		if !shouldProceed {
			return fmt.Errorf("exiting production deployment")
		}
	}
	return nil
}

// Runtimes
var supportedRuntimes = []string{
	"fastapi",
	"go",
	"nodejs",
	"docker",
	//"python",
}

func GetSupportedRuntimes() []string {
	return supportedRuntimes
}

func isValidRuntime(runtime string) bool {
	for _, current := range supportedRuntimes {
		if runtime == current {
			return true
		}
	}
	return false
}
