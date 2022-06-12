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
	auth0ClientId     string = "pJTegL4TmzS3RqWdcDlEg2bMpU8LlqnX"
	auth0ClientSecret string = "SCYMY6DjjsFGdadfH6pVfzdwUG_b4Bc5ETIeW0JMIhx4asu1DEE22Qq6IvuQq2Ua" // how do we propery store this?
	auth0BaseUrl      string = "https://dev-idh20w22.us.auth0.com"
	apiAudience       string = "https://api.usenucleus.cloud"

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
		shouldDeploy := cliPrompt("are you sure you want to deploy to production? (y/n)", "n")
		if shouldDeploy != "y" {
			return errors.New("exiting as received non yes answer for production deploy")
		}
	}
	return nil
}
