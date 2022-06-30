package utils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

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
		shouldProceed := CliPrompt("\nAre you sure you want to deploy this in production? (y/n)", "n")
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

func CliPrompt(label string, defaultEnv string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, label+" ")
	s, _ = r.ReadString('\n')
	if s == "\n" {
		s = defaultEnv
	}
	return strings.TrimSpace(s)
}
