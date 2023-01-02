package utils

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/nucleuscloud/cli/internal/auth"
	"github.com/nucleuscloud/cli/internal/config"
	mgmtv1alpha1 "github.com/nucleuscloud/mgmt-api/gen/proto/go/mgmt/v1alpha1"
	"github.com/toqueteos/webbrowser"
)

type callbackResponse struct {
	code  string
	state string
}

const (
	httpSrvBaseUrl = "localhost:4242"
	callbackPath   = "/api/auth/callback"
)

var (
	redirectUri = fmt.Sprintf("http://%s%s", httpSrvBaseUrl, callbackPath)
)

func Login(ctx context.Context) error {
	authClient, err := auth.NewAuthClientByEnv(GetEnv())
	if err != nil {
		return err
	}

	codeChan := make(chan callbackResponse)
	errChan := make(chan error)

	http.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		resAuthCode := r.URL.Query().Get("code")
		resAuthState := r.URL.Query().Get("state")
		errorCode := r.URL.Query().Get("error")
		errorMsg := r.URL.Query().Get("error_description")
		if errorCode != "" || errorMsg != "" {
			fmt.Fprintf(w, "Error Code: %s\nError Description: %s\n", errorCode, errorMsg)
			errChan <- fmt.Errorf("unabe to finish login flow")
			return
		}
		if resAuthCode == "" || resAuthState == "" {
			fmt.Fprintf(w, "Missing required query parameters to finish logging in.")
			errChan <- fmt.Errorf("received invalid callback response")
			return
		}
		fmt.Fprintf(w, "Login success! You may now return to your CLI window.")
		codeChan <- callbackResponse{resAuthCode, resAuthState}
	})

	go func() {
		httpErr := http.ListenAndServe(httpSrvBaseUrl, nil)
		if httpErr != nil {
			errChan <- httpErr
		}
	}()

	state := uuid.NewString()

	authorizeUrl := authClient.GetAuthorizeUrl(Scopes, state, redirectUri)
	err = webbrowser.Open(authorizeUrl)
	if err != nil {
		return err
	}

	select {
	case response := <-codeChan:
		close(errChan)
		close(codeChan)
		if state != response.state {
			return fmt.Errorf("State received from response was not what was sent")
		}
		return getAccessTokenAndSetUser(ctx, response.code, response.state, redirectUri, GetEnv())
	case err := <-errChan:
		close(errChan)
		close(codeChan)
		return err
	}
}

func getAccessTokenAndSetUser(
	ctx context.Context,
	code string,
	state string,
	redirectUri string,
	envType string,
) error {
	conn, err := NewAnonymousConnection()
	if err != nil {
		fmt.Println("failed to create anonymous connection")
		return err
	}

	apiCfg := GetApiConnectionConfigByEnv(envType)
	nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
	tokenResponse, err := nucleusClient.GetAccessToken(ctx, &mgmtv1alpha1.GetAccessTokenRequest{
		ClientId:    apiCfg.AuthClientId,
		Code:        code,
		RedirectUri: redirectUri,
	})
	if err != nil {
		fmt.Println("failed to get access token from nucleus client")
		return err
	}

	err = config.SetNucleusAuthFile(config.NucleusAuthConfig{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		IdToken:      tokenResponse.IdToken,
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
	_, err = nucleusClient.SetUser(ctx, &mgmtv1alpha1.SetUserRequest{})
	return err
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
