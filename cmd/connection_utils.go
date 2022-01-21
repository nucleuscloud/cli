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
	return grpc.Dial("haiku-api.haiku-api.apps.haiku.icu:443", grpc.WithTransportCredentials(creds))
	// return grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
}
