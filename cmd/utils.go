package cmd

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	// ErrInvalidName -
	ErrInvalidName   = fmt.Errorf("invalid name")
	validNameMatcher = regexp.MustCompile("^[a-z][a-z1-9-]*$").MatchString
)

// Auth Vars
var (
	auth0ClientId string = "STljLBgOpW4fuwyKT30YWBsvnxyVAZkr"
	auth0BaseUrl  string = "https://auth.stage.usenucleus.cloud"
	apiAudience   string = "https://api.usenucleus.cloud"

	scopes []string = []string{
		"openid",
		"profile",
		"offline_access",

		// custom
		"deploy:service",
		"read:service",
	}
)

func isValidName(s string) bool {
	return validNameMatcher(s)
}

func getGrpcTrailer() grpc.CallOption {
	// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
	var trailer metadata.MD
	return grpc.Trailer(&trailer)
}

func isValidEnvironmentType(environmentType string) bool {
	return environmentType != "dev" && environmentType != "stage" && environmentType != "prod"
}

func checkProdOk(cmd *cobra.Command, environmentType string, yesPromptFlagName string) error {
	yesPrompt, err := cmd.Flags().GetBool(yesPromptFlagName)
	if err != nil {
		return err
	}
	if !yesPrompt {
		shouldProceed := cliPrompt("\nAre you sure you want to deploy this in production? (y/n)", "n")
		if shouldProceed != "y" {
			return errors.New("Exiting production deployment")
		}
	}
	return nil
}

// Runtimes
var supportedRuntimes = []string{
	"fastapi",
	"go",
	"nodejs",
	"python",
}

func isValidRuntime(runtime string) bool {
	for _, current := range supportedRuntimes {
		if runtime == current {
			return true
		}
	}
	return false
}
