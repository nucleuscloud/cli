package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/auth"
	"github.com/nucleuscloud/cli/internal/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	nucleusDebugEnvKey = "NUCLEUS_DEBUG_ENV"
)

var (
	allowedDebugVals = []string{
		"dev",
		"stage",
	}
	hasLoggedAboutEnvType bool = false
)

func GetEnv() string {
	val := os.Getenv(nucleusDebugEnvKey)
	if val == "" {
		return val
	}
	var isValid bool = false
	for _, allowedVal := range allowedDebugVals {
		if allowedVal == val {
			isValid = true
		}
	}
	if !isValid {
		panic(fmt.Errorf("%s can only be one of %s", nucleusDebugEnvKey, strings.Join(allowedDebugVals, ",")))
	}
	if !hasLoggedAboutEnvType {
		fmt.Printf("%s=%s\n", nucleusDebugEnvKey, val)
		hasLoggedAboutEnvType = true
	}
	return val
}

func getApiUrl() string {
	if isDevEnv() {
		return "localhost:50051"
	} else if isStageEnv() {
		return "nucleus-api.nucleus-api.svcs.stage.usenucleus.cloud:443"
	}
	return "nucleus-api.nucleus-api.svcs.prod.usenucleus.cloud:443"
}

func isDevEnv() bool {
	return GetEnv() == "dev"
}
func isStageEnv() bool {
	return GetEnv() == "stage"
}

func getTransportCreds() (credentials.TransportCredentials, error) {
	if isDevEnv() {
		return insecure.NewCredentials(), nil
	}
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})
	return creds, nil
}

func newAnonymousConnection() (*grpc.ClientConn, error) {
	creds, err := getTransportCreds()
	if err != nil {
		return nil, err
	}
	return grpc.Dial(getApiUrl(), grpc.WithTransportCredentials(creds))
}

func NewAuthenticatedConnection(accessToken string) (*grpc.ClientConn, error) {
	creds, err := getTransportCreds()
	if err != nil {
		return nil, err
	}
	return grpc.Dial(
		getApiUrl(),
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(&loginCreds{
			AccessToken: accessToken,
		}),
	)
}

type ApiConnectionConfig struct {
	AuthBaseUrl  string
	AuthClientId string
	ApiAudience  string
}

// Returns a GRPC client that has been authenticated for use with Nucleus API
func NewApiConnection(cfg ApiConnectionConfig) (*grpc.ClientConn, error) {
	//refactor these clients into a utils file later
	authClient, err := auth.NewAuthClient(cfg.AuthBaseUrl, cfg.AuthClientId, cfg.ApiAudience)
	if err != nil {
		return nil, err
	}
	unAuthConn, err := newAnonymousConnection()
	if err != nil {
		return nil, err
	}
	unAuthCliClient := pb.NewCliServiceClient(unAuthConn)
	accessToken, err := config.GetValidAccessTokenFromConfig(authClient, unAuthCliClient)
	defer unAuthConn.Close()
	if err != nil {
		return nil, err
	}

	conn, err := NewAuthenticatedConnection(accessToken)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

type loginCreds struct {
	AccessToken string
}

func (c *loginCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": fmt.Sprintf("bearer %s", c.AccessToken),
	}, nil
}

func (c *loginCreds) RequireTransportSecurity() bool {
	return !isDevEnv()
}
