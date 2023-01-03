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

func PromptToProceed(cmd *cobra.Command, environmentName string, yesPromptFlagName string) error {
	yesPrompt, err := cmd.Flags().GetBool(yesPromptFlagName)
	if err != nil {
		return err
	}
	if !yesPrompt {
		shouldProceed := false
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Are you sure you want to invoke this command in %s?", environmentName),
		}, &shouldProceed)
		if err != nil {
			return err
		}

		if !shouldProceed {
			return fmt.Errorf("exiting %s deployment", environmentName)
		}
	}
	return nil
}

// Runtimes
var supportedRuntimes = []string{
	"go",
	"nodejs",
	"docker",
	"python",
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
