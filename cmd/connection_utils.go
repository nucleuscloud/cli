package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	nucleusApiUrl = "nucleus-api.nucleus-api.svcs.stage.usenucleus.cloud:443"
)

// func newConnection() (*grpc.ClientConn, error) {
// 	systemRoots, err := x509.SystemCertPool()
// 	if err != nil {
// 		return nil, err
// 	}

// 	creds := credentials.NewTLS(&tls.Config{
// 		RootCAs: systemRoots,
// 	})
// 	return grpc.Dial(nucleusApiUrl, grpc.WithTransportCredentials(creds))
// }

func newAuthenticatedConnection() (*grpc.ClientConn, error) {
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})

	accessToken, err := getValidAccessTokenFromConfig()
	if err != nil {
		return nil, err
	}

	return grpc.Dial(
		nucleusApiUrl,
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
	return true
}
