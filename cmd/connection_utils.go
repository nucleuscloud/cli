package cmd

import (
	"crypto/tls"
	"crypto/x509"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newConnection() (*grpc.ClientConn, error) {
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})
	return grpc.Dial("nucleus-api.nucleus-api.svcs.stage.usenucleus.cloud:443", grpc.WithTransportCredentials(creds))
}
