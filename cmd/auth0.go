package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/toqueteos/webbrowser"
)

var (
	auth0ClientId string = "pJTegL4TmzS3RqWdcDlEg2bMpU8LlqnX"
	baseUrl       string = "https://dev-idh20w22.us.auth0.com"
	auth0LoginUrl string = fmt.Sprintf("%s/oauth/device/code", baseUrl)
	auth0TokenUrl string = fmt.Sprintf("%s/oauth/token", baseUrl)
	apiAudience   string = "https://api.usenucleus.cloud"

	accessDeniedError = errors.New("access denied")
	expiredTokenError = errors.New("expired token")
	unknownTokenError = errors.New("unable to authenticate")
)

// loginCmd represents the login command
var auth0Cmd = &cobra.Command{
	Use:   "auth0 login",
	Short: "Logs a user into their Nucleus account.",
	Long:  `Logs a user into their Nucleus account. `,

	RunE: func(cmd *cobra.Command, args []string) error {
		deviceResponse, err := getDeviceCodeResponse()

		if err != nil {
			return err
		}

		// fmt.Println("Visit the following URL to login: ", deviceResponse.VerificationURIComplete)
		cliPrompt("Press [Enter] to continue in the web browser...", "")

		err = webbrowser.Open(deviceResponse.VerificationURIComplete)
		if err != nil {
			fmt.Println("There was an issue opening the web browser, proceed to the following URL to continue logging in: ", deviceResponse.VerificationURIComplete)
		}

		tokenResponse, err := pollToken(deviceResponse)

		if err != nil {
			// handle expired token error by re-prompting
			fmt.Println("There was an error. Please try logging in again")
			return err
		}
		// todo: store token and refresh token
		fmt.Println(tokenResponse.Result.AccessToken)

		return nil

	},
}

func init() {
	rootCmd.AddCommand(auth0Cmd)
}

type Auth0DeviceResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

func getDeviceCodeResponse() (*Auth0DeviceResponse, error) {
	payload := strings.NewReader(fmt.Sprintf("client_id=%s&scope=SCOPE&audience=%s", auth0ClientId, apiAudience))
	req, err := http.NewRequest("POST", auth0LoginUrl, payload)

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

	var deviceResponse Auth0DeviceResponse

	err = json.Unmarshal(body, &deviceResponse)
	if err != nil {
		return nil, err
	}

	return &deviceResponse, nil
}

type Auth0TokenResponse struct {
	Result *Auth0TokenResponseData
	Error  *Auth0TokenErrorData
}

type Auth0TokenResponseData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IdToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type Auth0TokenErrorData struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func pollToken(deviceResponse *Auth0DeviceResponse) (*Auth0TokenResponse, error) {

	checkInterval := time.Duration(deviceResponse.Interval) * time.Second
	expiresAt := time.Now().Add(time.Duration(deviceResponse.ExpiresIn) * time.Second)

	for {
		time.Sleep(checkInterval)

		if time.Now().After(expiresAt) {
			return nil, errors.New("authenticated timed out")
		}

		resp, err := getTokenResponse(deviceResponse.DeviceCode)

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
				return nil, expiredTokenError
			} else if resp.Error.Error == "access_denied" {
				return nil, accessDeniedError
			} else {
				// unknown error
				return nil, unknownTokenError
			}
		} else {
			return resp, nil
		}
	}
}

func getTokenResponse(deviceCode string) (*Auth0TokenResponse, error) {
	payload := strings.NewReader(fmt.Sprintf("grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=%s&client_id=%s", deviceCode, auth0ClientId))
	req, err := http.NewRequest("POST", auth0TokenUrl, payload)

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

	var tokenResponse Auth0TokenResponseData
	err = json.Unmarshal(body, &tokenResponse)

	if err != nil {
		return nil, err
	}

	if tokenResponse.AccessToken == "" {
		var errorResponse Auth0TokenErrorData
		err = json.Unmarshal(body, &errorResponse)
		if err != nil {
			return nil, err
		}
		return &Auth0TokenResponse{
			Result: nil,
			Error:  &errorResponse,
		}, nil
	}

	return &Auth0TokenResponse{
		Result: &tokenResponse,
		Error:  nil,
	}, nil
}

func getHttpClient() *http.Client {
	client := &http.Client{Timeout: 10 * time.Second}
	return client
}
