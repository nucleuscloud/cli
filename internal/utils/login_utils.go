package utils

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/nucleuscloud/cli/internal/auth"
	"github.com/nucleuscloud/cli/internal/config"
	clienv "github.com/nucleuscloud/cli/internal/env"
	authv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/auth/v1alpha1"
	mgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/mgmt/v1alpha1"
	"github.com/spf13/viper"
	"github.com/toqueteos/webbrowser"
)

type oauthCallbackResponse struct {
	code  string
	state string
}

func getHttpSrvHost() string {
	host := viper.GetString("LOGIN_HOST")
	if host == "" {
		return "127.0.0.1"
	}
	return host
}

func getHttpRedirectHost() string {
	host := viper.GetString("LOGIN_REDIRECT_HOST")
	if host == "" {
		return "127.0.0.1"
	}
	return host
}

func getHttpSrvPort() uint32 {
	port := viper.GetUint32("LOGIN_PORT")
	if port == 0 {
		return 4242
	}
	return port
}

func getRedirectUriBaseUrl() string {
	return fmt.Sprintf("%s:%d", getHttpRedirectHost(), getHttpSrvPort())
}

func getHttpSrvBaseUrl() string {
	return fmt.Sprintf("%s:%d", getHttpSrvHost(), getHttpSrvPort())
}

const (
	callbackPath    = "/api/auth/callback"
	orgCallbackPath = "/api/auth/org/callback"
)

var (
	redirectUri    = fmt.Sprintf("http://%s%s", getRedirectUriBaseUrl(), callbackPath)
	redirectOrgUri = fmt.Sprintf("http://%s%s", getRedirectUriBaseUrl(), orgCallbackPath)
)

func OAuthLogin(ctx context.Context) error {
	authClient, err := auth.NewAuthClientByEnv(clienv.GetEnv())
	if err != nil {
		return err
	}

	orgCodeChan := make(chan oauthCallbackResponse)
	errChan := make(chan error)

	state := uuid.NewString()
	orgState := uuid.NewString()

	http.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		resAuthCode := r.URL.Query().Get("code")
		resAuthState := r.URL.Query().Get("state")
		errorCode := r.URL.Query().Get("error")
		errorMsg := r.URL.Query().Get("error_description")
		if errorCode != "" || errorMsg != "" {
			err := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        errorCode,
				ErrorDescription: errorMsg,
			})
			if err != nil {
				errChan <- err
				return
			}
			errChan <- fmt.Errorf("unabe to finish login flow")
			return
		}
		if resAuthCode == "" || resAuthState == "" {
			err := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        "BadRequest",
				ErrorDescription: "Missing required query parameters to finish logging in.",
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			errChan <- fmt.Errorf("received invalid callback response")
			return
		}
		if state != resAuthState {
			err := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        "BadRequest",
				ErrorDescription: "Received invalid state in response",
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			errChan <- fmt.Errorf("received invalid state in response")
			return
		}

		accessTokenRes, err := getAccessToken(ctx, resAuthCode, resAuthState, redirectUri, clienv.GetEnv())
		if err != nil {
			renderErr := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        "Internal",
				ErrorDescription: "Unable to get access token to continue logging in",
			})
			if renderErr != nil {
				fmt.Fprintln(os.Stderr, renderErr)
			}
			errChan <- err
			return
		}

		orgIds, err := getUsersOrganizations(ctx, accessTokenRes.AccessToken)
		if err != nil {
			renderErr := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        "Internal",
				ErrorDescription: "Unable to retrieve your organizations.",
			})
			if renderErr != nil {
				fmt.Fprintln(os.Stderr, renderErr)
			}
			errChan <- err
			return
		}
		if len(orgIds) > 0 {
			orgId := orgIds[0]
			authorizeUrl := authClient.GetAuthorizeUrl(Scopes, orgState, redirectOrgUri, &orgId)
			http.Redirect(w, r, authorizeUrl, 301)
		} else {
			err := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        "Internal",
				ErrorDescription: "Must have an organization to login to CLI. Login through the https://nucleuscloud.com and create an organization to continue.",
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			errChan <- fmt.Errorf("must have an organization in order to login to CLI")
			return
		}
	})

	http.HandleFunc(orgCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		resAuthCode := r.URL.Query().Get("code")
		resAuthState := r.URL.Query().Get("state")
		errorCode := r.URL.Query().Get("error")
		errorMsg := r.URL.Query().Get("error_description")
		if errorCode != "" || errorMsg != "" {
			err := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        errorCode,
				ErrorDescription: errorMsg,
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			errChan <- fmt.Errorf("unabe to finish login flow")
			return
		}
		if resAuthCode == "" || resAuthState == "" {
			err := RenderLoginErrorPage(w, LoginPageErrorData{
				Title:            "Login Failed",
				ErrorCode:        "BadRequest",
				ErrorDescription: "Missing required query parameters to finish logging in.",
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			errChan <- fmt.Errorf("received invalid callback response")
			return
		}
		err := RenderLoginSuccessPage(w, LoginPageData{Title: "Success"})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		orgCodeChan <- oauthCallbackResponse{resAuthCode, resAuthState}
	})

	go func() {
		httpErr := http.ListenAndServe(getHttpSrvBaseUrl(), nil)
		if httpErr != nil {
			errChan <- httpErr
		}
	}()

	authorizeUrl := authClient.GetAuthorizeUrl(Scopes, state, redirectUri, nil)

	err = webbrowser.Open(authorizeUrl)
	if err != nil {
		fmt.Println("There was an issue opening the web browser, proceed to the following url to finish logging in to Nucleus", authorizeUrl)
	}

	select {
	case err := <-errChan:
		close(errChan)
		close(orgCodeChan)
		return err
	case response := <-orgCodeChan:
		close(orgCodeChan)
		if orgState != response.state {
			return fmt.Errorf("state received from response was not what was sent")
		}
		accessTokenRes, err := getAccessToken(ctx, response.code, response.state, redirectOrgUri, clienv.GetEnv())
		if err != nil {
			return err
		}
		conn, err := NewAuthenticatedConnection(accessTokenRes.AccessToken)
		if err != nil {
			return err
		}
		defer conn.Close()

		nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
		_, err = nucleusClient.GetUser(ctx, &mgmtv1alpha1.GetUserRequest{})
		if err != nil {
			return err
		}
		err = config.SetNucleusAuthFile(config.NucleusAuthConfig{
			AccessToken:  accessTokenRes.AccessToken,
			RefreshToken: accessTokenRes.RefreshToken,
			IdToken:      accessTokenRes.IdToken,
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func getAccessToken(
	ctx context.Context,
	code string,
	state string,
	redirectUri string,
	envType clienv.NucleusEnv,
) (*authv1alpha1.GetAccessTokenResponse, error) {
	conn, err := NewAnonymousConnection()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create anonymous connection")
		return nil, err
	}

	apiCfg := GetApiConnectionConfigByEnv(envType)
	nucleusAuthClient := authv1alpha1.NewAuthServiceClient(conn)
	tokenResponse, err := nucleusAuthClient.GetAccessToken(ctx, &authv1alpha1.GetAccessTokenRequest{
		ClientId:    apiCfg.AuthClientId,
		Code:        code,
		RedirectUri: redirectUri,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get access token from nucleus client")
		return nil, err
	}
	err = conn.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return tokenResponse, nil
}

func getUsersOrganizations(
	ctx context.Context,
	accessToken string,
) ([]string, error) {

	conn, err := NewAuthenticatedConnection(accessToken)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
	orgRes, err := nucleusClient.GetUserOrganizations(ctx, &mgmtv1alpha1.GetUserOrganizationsRequest{})
	if err != nil {
		return nil, err
	}

	return orgRes.OrgIds, nil
}

func ClientLogin(ctx context.Context, clientId string, clientSecret string) error {
	conn, err := NewAnonymousConnection()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create anonymous connection")
		return err
	}

	nucleusAuthClient := authv1alpha1.NewAuthServiceClient(conn)
	tokenResponse, err := nucleusAuthClient.GetServiceAccountAccessToken(ctx, &authv1alpha1.GetServiceAccountAccessTokenRequest{
		ClientId:     clientId,
		ClientSecret: clientSecret,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get client access token from nucleus client")
		return err
	}

	err = config.SetNucleusAuthFile(config.NucleusAuthConfig{
		AccessToken: tokenResponse.AccessToken,
	})
	if err != nil {
		return err
	}
	conn.Close()

	conn, err = NewAuthenticatedConnection(tokenResponse.AccessToken)
	if err != nil {
		return err
	}
	defer conn.Close()

	nucleusMgmtClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
	_, err = nucleusMgmtClient.GetAccountByServiceAccountClientId(ctx, &mgmtv1alpha1.GetAccountByServiceAccountClientIdRequest{})
	return err

}
