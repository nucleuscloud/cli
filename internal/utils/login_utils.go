package utils

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/nucleuscloud/cli/internal/auth"
	"github.com/nucleuscloud/cli/internal/config"
	clienv "github.com/nucleuscloud/cli/internal/env"
	mgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/mgmt/v1alpha1"
	"github.com/toqueteos/webbrowser"
)

type oauthCallbackResponse struct {
	code  string
	state string
}

const (
	httpSrvBaseUrl  = "localhost:4242"
	callbackPath    = "/api/auth/callback"
	orgCallbackPath = "/api/auth/org/callback"
)

var (
	redirectUri    = fmt.Sprintf("http://%s%s", httpSrvBaseUrl, callbackPath)
	redirectOrgUri = fmt.Sprintf("http://%s%s", httpSrvBaseUrl, orgCallbackPath)
)

func OAuthLogin(ctx context.Context) error {
	authClient, err := auth.NewAuthClientByEnv(clienv.GetEnv())
	if err != nil {
		return err
	}

	codeChan := make(chan oauthCallbackResponse)
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
				errChan <- err
				return
			}
			errChan <- fmt.Errorf("received invalid callback response")
			return
		}
		//////
		accessTokenRes, err := getAccessToken(ctx, resAuthCode, resAuthState, redirectUri, clienv.GetEnv())
		if err != nil {
			errChan <- err
			return
		}

		conn, err := NewAuthenticatedConnection(accessTokenRes.AccessToken)
		if err != nil {
			errChan <- err
			return
		}
		defer conn.Close()

		nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
		_, err = nucleusClient.SetUser(ctx, &mgmtv1alpha1.SetUserRequest{})
		if err != nil {
			errChan <- err
			return
		}

		orgRes, err := nucleusClient.GetUserOrganizations(ctx, &mgmtv1alpha1.GetUserOrganizationsRequest{})
		if err != nil {
			errChan <- err
			return
		}
		fmt.Println("Found Org Ids: ", strings.Join(orgRes.OrgIds, ", "))

		if len(orgRes.OrgIds) > 0 {
			orgId := orgRes.OrgIds[0]
			authorizeUrl := authClient.GetAuthorizeUrl(Scopes, orgState, redirectOrgUri, &orgId)
			http.Redirect(w, r, authorizeUrl, 301)
			return
		} else {
			errChan <- fmt.Errorf("must have an organization in order to login to CLI")
			// todo
			return
		}

		//////
		// err = RenderLoginSuccessPage(w, LoginPageData{Title: "Success"})
		// if err != nil {
		// 	errChan <- fmt.Errorf("unable to write to login page")
		// 	return
		// }
		// codeChan <- oauthCallbackResponse{resAuthCode, resAuthState}
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
				errChan <- err
				return
			}
			errChan <- fmt.Errorf("received invalid callback response")
			return
		}
		err := RenderLoginSuccessPage(w, LoginPageData{Title: "Success"})
		if err != nil {
			errChan <- fmt.Errorf("unable to write to login page")
		}
		orgCodeChan <- oauthCallbackResponse{resAuthCode, resAuthState}
	})

	go func() {
		httpErr := http.ListenAndServe(httpSrvBaseUrl, nil)
		if httpErr != nil {
			errChan <- httpErr
		}
	}()

	authorizeUrl := authClient.GetAuthorizeUrl(Scopes, state, redirectUri, nil)
	err = webbrowser.Open(authorizeUrl)
	if err != nil {
		return err
	}

	// select {
	// case response := <-codeChan:
	// 	close(codeChan)
	// 	if state != response.state {
	// 		return fmt.Errorf("State received from response was not what was sent")
	// 	}
	// 	accessTokenRes, err := getAccessToken(ctx, response.code, response.state, redirectUri, clienv.GetEnv())
	// 	if err != nil {
	// 		return err
	// 	}

	// 	conn, err := NewAuthenticatedConnection(accessTokenRes.AccessToken)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	defer conn.Close()

	// 	nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
	// 	_, err = nucleusClient.SetUser(ctx, &mgmtv1alpha1.SetUserRequest{})
	// 	if err != nil {
	// 		return err
	// 	}

	// 	orgRes, err := nucleusClient.GetUserOrganizations(ctx, &mgmtv1alpha1.GetUserOrganizationsRequest{})
	// 	if err != nil {
	// 		return err
	// 	}
	// 	fmt.Println("Found Org Ids: ", strings.Join(orgRes.OrgIds, ", "))

	// 	if len(orgRes.OrgIds) > 0 {
	// 		orgId := orgRes.OrgIds[0]
	// 		authorizeUrl := authClient.GetAuthorizeUrl(Scopes, orgState, redirectOrgUri, &orgId)
	// 		err = webbrowser.Open(authorizeUrl)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	} else {
	// 		return fmt.Errorf("must have an organization in order to login to CLI")
	// 	}

	// case err := <-errChan:
	// 	close(errChan)
	// 	close(codeChan)
	// 	close(orgCodeChan)
	// 	return err
	// }

	select {
	case err := <-errChan:
		close(errChan)
		close(codeChan)
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
		_, err = nucleusClient.SetUser(ctx, &mgmtv1alpha1.SetUserRequest{})
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
) (*mgmtv1alpha1.GetAccessTokenResponse, error) {
	conn, err := NewAnonymousConnection()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create anonymous connection")
		return nil, err
	}

	apiCfg := GetApiConnectionConfigByEnv(envType)
	nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
	tokenResponse, err := nucleusClient.GetAccessToken(ctx, &mgmtv1alpha1.GetAccessTokenRequest{
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

	// err = config.SetNucleusAuthFile(config.NucleusAuthConfig{
	// 	AccessToken:  tokenResponse.AccessToken,
	// 	RefreshToken: tokenResponse.RefreshToken,
	// 	IdToken:      tokenResponse.IdToken,
	// })
	// if err != nil {
	// 	return err
	// }
	// conn.Close()

	// conn, err = NewAuthenticatedConnection(tokenResponse.AccessToken)
	// if err != nil {
	// 	return err
	// }
	// defer conn.Close()

	// nucleusClient = mgmtv1alpha1.NewMgmtServiceClient(conn)
	// _, err = nucleusClient.SetUser(ctx, &mgmtv1alpha1.SetUserRequest{})

	// orgRes, err := nucleusClient.GetUserOrganizations(ctx, &mgmtv1alpha1.GetUserOrganizationsRequest{})
	// if err != nil {
	// 	return err
	// }
	// fmt.Println("Found Org Ids: ", strings.Join(orgRes.OrgIds, ", "))

	// if len(orgRes.OrgIds) > 0 {
	// 	orgId := orgRes.OrgIds[0]
	// 	authorizeUrl := authClient.GetAuthorizeUrl(Scopes, state, redirectUri, &orgId)
	// 	err = webbrowser.Open(authorizeUrl)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
}

func ClientLogin(ctx context.Context, clientId string, clientSecret string) error {
	conn, err := NewAnonymousConnection()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create anonymous connection")
		return err
	}

	nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
	tokenResponse, err := nucleusClient.GetServiceAccountAccessToken(ctx, &mgmtv1alpha1.GetServiceAccountAccessTokenRequest{
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

	nucleusClient = mgmtv1alpha1.NewMgmtServiceClient(conn)
	_, err = nucleusClient.GetAccountByServiceAccountClientId(ctx, &mgmtv1alpha1.GetAccountByServiceAccountClientIdRequest{})
	return err

}
