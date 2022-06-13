package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func getEnv() string {
	return os.Getenv("ENV")
}

func getApiUrl() string {
	if isDevEnv() {
		return "localhost:50051"
	}
	return "nucleus-api.nucleus-api.svcs.stage.usenucleus.cloud:443"
}

func isDevEnv() bool {
	return getEnv() == "dev"
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

func newConnection() (*grpc.ClientConn, error) {
	creds, err := getTransportCreds()
	if err != nil {
		return nil, err
	}
	return grpc.Dial(getApiUrl(), grpc.WithTransportCredentials(creds))
}

func newAuthenticatedConnection(accessToken string) (*grpc.ClientConn, error) {
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
