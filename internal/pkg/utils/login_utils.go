package utils

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/nucleuscloud/api/pkg/api/v1/pb"
	"github.com/nucleuscloud/cli/internal/pkg/auth"
	"github.com/nucleuscloud/cli/internal/pkg/config"
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

func LoginManaged(verbose bool) error {
	authClient, err := auth.NewAuthClientByEnv(GetEnv(), false)
	if err != nil {
		return err
	}

	deviceResponse, err := authClient.GetDeviceCode(Scopes)
	if err != nil {
		return err
	}

	fmt.Println("Your activation code is: ", deviceResponse.UserCode)

	err = webbrowser.Open(deviceResponse.VerificationURIComplete)
	if err != nil {
		fmt.Println("There was an issue opening the web browser, proceed to the following URL to continue logging in: ", deviceResponse.VerificationURIComplete)
	}

	tokenResponse, err := authClient.PollDeviceAccessToken(deviceResponse)

	if err != nil {
		// handle expired token error by re-prompting
		fmt.Println("There was an error. Please try logging in again")
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

	conn, err := NewAuthenticatedConnection(tokenResponse.AccessToken, false)
	if err != nil {
		return err
	}

	defer conn.Close()

	if verbose {
		fmt.Println("Attempting to register user in Nucleus system...")
	}

	nucleusClient := pb.NewCliServiceClient(conn)
	_, err = nucleusClient.ResolveUser(context.Background(), &pb.ResolveUserRequest{})
	if err != nil {
		return err
	}
	fmt.Println("User successfully resolved in Nucleus system!")
	return nil
}

func LoginOnPrem() error {
	ctx := context.Background()
	authClient, err := auth.NewAuthClientByEnv(GetEnv(), true)
	if err != nil {
		return err
	}

	codeChan := make(chan callbackResponse)
	errChan := make(chan error)

	http.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		resAuthCode := r.URL.Query().Get("code")
		resAuthState := r.URL.Query().Get("state")
		if resAuthCode == "" || resAuthState == "" {
			fmt.Fprintf(w, "Missing required query parameters to finish logging in.")
			errChan <- fmt.Errorf("Received invalid callback response")
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
		return getAccessTokenAndSetUser(ctx, response.code, response.state, redirectUri, GetEnv(), true)
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
	isOnPrem bool,
) error {
	conn, err := NewAnonymousConnection(true)
	if err != nil {
		return err
	}

	apiCfg := GetApiConnectionConfigByEnv(envType, isOnPrem)
	nucleusClient := mgmtv1alpha1.NewMgmtServiceClient(conn)
	tokenResponse, err := nucleusClient.GetAccessToken(ctx, &mgmtv1alpha1.GetAccessTokenRequest{
		ClientId:    apiCfg.AuthClientId,
		Code:        code,
		RedirectUri: redirectUri,
	})
	if err != nil {
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

	conn, err = NewAuthenticatedConnection(tokenResponse.AccessToken, isOnPrem)
	if err != nil {
		return err
	}
	defer conn.Close()

	nucleusClient = mgmtv1alpha1.NewMgmtServiceClient(conn)
	_, err = nucleusClient.SetUser(ctx, &mgmtv1alpha1.SetUserRequest{})
	return err
}
