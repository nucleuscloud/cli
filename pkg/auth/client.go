package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

var (
	ErrAccessDenied = errors.New("access denied")
	ErrExpiredToken = errors.New("expired token")
	ErrUnknownToken = errors.New("unable to authenticate")
)

type AuthClientInterface interface {
	GetDeviceCode(scopes []string) (*AuthDeviceResponse, error)
	PollDeviceAccessToken(deviceResponse *AuthDeviceResponse) (*AuthTokenResponseData, error)
	ValidateToken(ctx context.Context, accessToken string) error
}

// Implements AuthClientInterface
type authClient struct {
	clientId string
	audience string

	loginUrl string
	tokenUrl string

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

type AuthTokenResponse struct {
	Result *AuthTokenResponseData
	Error  *AuthTokenErrorData
}

type AuthTokenResponseData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IdToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type AuthTokenErrorData struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
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
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
		return nil, err
	}
	return &authClient{
		clientId: clientId,
		audience: audience,

		loginUrl: fmt.Sprintf("%s/oauth/device/code", tenantUrl),
		tokenUrl: fmt.Sprintf("%s/oauth/token", tenantUrl),

		jwtValidator: jwtValidator,
	}, nil
}

func (c *authClient) GetDeviceCode(scopes []string) (*AuthDeviceResponse, error) {
	payload := strings.NewReader(fmt.Sprintf("client_id=%s&scope=%s&audience=%s", c.clientId, strings.Join(scopes, " "), c.audience))
	req, err := http.NewRequest("POST", c.loginUrl, payload)

	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := getHttpClient().Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	var deviceResponse AuthDeviceResponse

	err = json.Unmarshal(body, &deviceResponse)
	if err != nil {
		return nil, err
	}

	return &deviceResponse, nil
}

func (c *authClient) PollDeviceAccessToken(deviceResponse *AuthDeviceResponse) (*AuthTokenResponseData, error) {
	checkInterval := time.Duration(deviceResponse.Interval) * time.Second
	expiresAt := time.Now().Add(time.Duration(deviceResponse.ExpiresIn) * time.Second)

	for {
		time.Sleep(checkInterval)

		if time.Now().After(expiresAt) {
			return nil, errors.New("authenticated timed out")
		}

		resp, err := c.getTokenResponse(deviceResponse.DeviceCode)

		if err != nil {
			return nil, err
		} else if resp.Error != nil {
			if resp.Error.Error == "authorization_pending" {
				continue
			} else if resp.Error.Error == "slow_down" {
				// We can do better here.
				// Their docs say this:
				// To avoid receiving this error due to network latency, you should start counting each interval after receipt of the last polling request's response.
				time.Sleep(checkInterval)
			} else if resp.Error.Error == "expired_token" {
				return nil, ErrExpiredToken
			} else if resp.Error.Error == "access_denied" {
				return nil, ErrAccessDenied
			} else {
				// unknown error
				return nil, ErrUnknownToken
			}
		} else {
			return resp.Result, nil
		}
	}
}

func (c *authClient) getTokenResponse(deviceCode string) (*AuthTokenResponse, error) {
	payload := strings.NewReader(fmt.Sprintf("grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=%s&client_id=%s", deviceCode, c.clientId))
	req, err := http.NewRequest("POST", c.tokenUrl, payload)

	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := getHttpClient().Do(req)

	if err != nil {
		fmt.Println("Hit this error block")
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		// handle errors here
		fmt.Println("Received this error: ", err)
		fmt.Println("Got this response: ", body)
		return nil, err
	}

	var tokenResponse AuthTokenResponseData
	err = json.Unmarshal(body, &tokenResponse)

	if err != nil {
		return nil, err
	}

	if tokenResponse.AccessToken == "" {
		var errorResponse AuthTokenErrorData
		err = json.Unmarshal(body, &errorResponse)
		if err != nil {
			return nil, err
		}
		return &AuthTokenResponse{
			Result: nil,
			Error:  &errorResponse,
		}, nil
	}

	return &AuthTokenResponse{
		Result: &tokenResponse,
		Error:  nil,
	}, nil
}

type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

func (c *authClient) ValidateToken(ctx context.Context, accessToken string) error {
	_, err := c.jwtValidator.ValidateToken(ctx, accessToken)
	return err
}

func getHttpClient() *http.Client {
	client := &http.Client{Timeout: 10 * time.Second}
	return client
}
