package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	nucleusApiUrl = "localhost:50051"
)

func newConnection() (*grpc.ClientConn, error) {
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})
	return grpc.Dial(nucleusApiUrl, grpc.WithTransportCredentials(creds))
}

func newAuthenticatedConnection(accessToken string) (*grpc.ClientConn, error) {
	// systemRoots, err := x509.SystemCertPool()
	// if err != nil {
	// 	return nil, err
	// }

	// creds := credentials.NewTLS(&tls.Config{
	// 	RootCAs: systemRoots,
	// })

	// accessToken, err := getValidAccessTokenFromConfig(authClient, nucleusClient)
	// if err != nil {
	// 	return nil, err
	// }

	return grpc.Dial(
		nucleusApiUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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
	return false
}
