package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

const (
	Auth0StageClientId = "IHJD9fSlrH4p9WhPYp6uJe0yFNr26ZLy"
	Auth0StageBaseUrl  = "https://auth.stage.nucleuscloud.com"

	Auth0ProdClientId = "6zk97YDDj9YplY9jqOaHmKYojhEXquD8"
	Auth0ProdBaseUrl  = "https://auth.nucleuscloud.com"

	ApiAudience = "https://api.usenucleus.cloud"

	logoutReturnTo = "https://nucleuscloud.com"
)

type AuthClientInterface interface {
	ValidateToken(ctx context.Context, accessToken string) error
	GetLogoutUrl() (string, error)
	GetAuthorizeUrl(scopes []string, state string, redirectUri string) string
}

// Implements AuthClientInterface
type authClient struct {
	clientId string
	audience string

	authorizeUrl string
	logoutUrl    string

	jwtValidator *validator.Validator
}

type AuthDeviceResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type AuthTokenResponseData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IdToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func NewAuthClientByEnv(envType string) (AuthClientInterface, error) {
	switch envType {
	case "prod", "":
		return NewAuthClient(Auth0ProdBaseUrl, Auth0ProdClientId, ApiAudience)
	case "dev", "stage":
		return NewAuthClient(Auth0StageBaseUrl, Auth0StageClientId, ApiAudience)
	}
	return nil, fmt.Errorf("must provide valid env type")
}

func NewAuthClient(tenantUrl, clientId, audience string) (AuthClientInterface, error) {
	issuerUrl, err := url.Parse(tenantUrl + "/")
	if err != nil {
		return nil, err
	}
	provider := jwks.NewCachingProvider(issuerUrl, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerUrl.String(),
		[]string{audience},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &customClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		return nil, err
	}
	return &authClient{
		clientId: clientId,
		audience: audience,

		authorizeUrl: fmt.Sprintf("%s/authorize", tenantUrl),
		logoutUrl:    fmt.Sprintf("%s/v2/logout", tenantUrl),

		jwtValidator: jwtValidator,
	}, nil
}

func (c *authClient) GetAuthorizeUrl(scopes []string, state string, redirectUri string) string {
	params := url.Values{}
	params.Add("audience", c.audience)
	params.Add("scope", strings.Join(scopes, " "))
	params.Add("response_type", "code")
	params.Add("client_id", c.clientId)
	params.Add("redirect_uri", redirectUri)
	params.Add("state", state)

	return fmt.Sprintf("%s?%s", c.authorizeUrl, params.Encode())
}

type customClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c customClaims) Validate(ctx context.Context) error {
	return nil
}

func (c *authClient) ValidateToken(ctx context.Context, accessToken string) error {
	_, err := c.jwtValidator.ValidateToken(ctx, accessToken)
	return err
}

func (c *authClient) GetLogoutUrl() (string, error) {
	base, err := url.Parse(c.logoutUrl)
	if err != nil {
		return "", err
	}

	queryParams := url.Values{}
	queryParams.Add("client_id", c.clientId)
	queryParams.Add("returnTo", logoutReturnTo)
	base.RawQuery = queryParams.Encode()
	return base.String(), nil
}
