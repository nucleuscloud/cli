package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/nucleuscloud/cli/internal/auth"
	"github.com/nucleuscloud/cli/internal/config"
	mgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/mgmt/v1alpha1"
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
		return "mgmt-api-nucleus.svcs.stage.nucleuscloud.com:443"
	}
	return "mgmt-api-nucleus.svcs.nucleuscloud.com:443"
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

func NewAnonymousConnection() (*grpc.ClientConn, error) {
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

func GetApiConnectionConfigByEnv(envType string) *ApiConnectionConfig {
	switch envType {
	case "prod", "":
		return &ApiConnectionConfig{
			AuthBaseUrl:  auth.Auth0ProdBaseUrl,
			AuthClientId: auth.Auth0ProdClientId,
			ApiAudience:  auth.ApiAudience,
		}
	case "dev", "stage":
		return &ApiConnectionConfig{
			AuthBaseUrl:  auth.Auth0StageBaseUrl,
			AuthClientId: auth.Auth0StageClientId,
			ApiAudience:  auth.ApiAudience,
		}
	}
	return nil
}

func NewApiConnectionByEnv(ctx context.Context, envType string) (*grpc.ClientConn, error) {
	cfg := GetApiConnectionConfigByEnv(envType)
	if cfg == nil {
		return nil, fmt.Errorf("must provide valid env type")
	}
	return NewApiConnection(ctx, cfg)
}

func NewApiConnection(ctx context.Context, cfg *ApiConnectionConfig) (*grpc.ClientConn, error) {
	authClient, err := auth.NewAuthClient(cfg.AuthBaseUrl, cfg.AuthClientId, cfg.ApiAudience)
	if err != nil {
		return nil, err
	}
	unAuthConn, err := NewAnonymousConnection()
	if err != nil {
		return nil, err
	}
	unAuthCliClient := mgmtv1alpha1.NewMgmtServiceClient(unAuthConn)
	accessToken, err := getValidAccessTokenFromConfig(ctx, authClient, unAuthCliClient, cfg.AuthClientId)
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

// Retrieves the access token from the config and validates it.
func getValidAccessTokenFromConfig(
	ctx context.Context,
	authClient auth.AuthClientInterface,
	mgmtClient mgmtv1alpha1.MgmtServiceClient,
	clientId string,
) (string, error) {
	cfg, err := config.GetNucleusAuthConfig()
	if err != nil {
		return "", err
	}
	err = authClient.ValidateToken(ctx, cfg.AccessToken)
	if err != nil {
		fmt.Println("Access token is no longer valid. Attempting to refresh...")
		if cfg.RefreshToken != "" {
			res, err := getRefreshResponse(ctx, mgmtClient, clientId, cfg.RefreshToken)
			if err != nil {
				err2 := config.ClearNucleusAuthFile()
				if err2 != nil {
					fmt.Println("unable to remove nucleus auth file", err2)
				}
				fmt.Println(err)
				return "", fmt.Errorf("unable to refresh token, please try logging in again.")
			}
			var newRefreshToken string
			if res.RefreshToken != "" {
				newRefreshToken = res.RefreshToken
			} else {
				newRefreshToken = cfg.RefreshToken
			}
			err = config.SetNucleusAuthFile(config.NucleusAuthConfig{
				AccessToken:  res.AccessToken,
				RefreshToken: newRefreshToken,
				IdToken:      res.IdToken,
			})
			if err != nil {
				fmt.Println("Successfully refreshed token, but was unable to update nucleus auth file")
				return "", err
			}
			return res.AccessToken, authClient.ValidateToken(ctx, res.AccessToken)
		}
	}
	return cfg.AccessToken, authClient.ValidateToken(ctx, cfg.AccessToken)
}

type refreshResponse struct {
	AccessToken  string
	RefreshToken string
	IdToken      string
}

func getRefreshResponse(
	ctx context.Context,
	mgmtClient mgmtv1alpha1.MgmtServiceClient,
	clientId string,
	refreshToken string,
) (*refreshResponse, error) {
	reply, err := mgmtClient.GetNewAccessToken(ctx, &mgmtv1alpha1.GetNewAccessTokenRequest{
		ClientId:     clientId,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, err
	}
	return &refreshResponse{
		AccessToken:  reply.AccessToken,
		RefreshToken: reply.RefreshToken,
		IdToken:      reply.IdToken,
	}, nil
}
