package cmd

import (
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func newConnection() (*grpc.ClientConn, error) {
	creds, err := credentials.NewClientTLSFromFile("service.pem", "")
	if err != nil {
		log.Fatalf("could not process the credentials: %v", err)
	}

	return grpc.Dial("127.0.0.1:50051", grpc.WithTransportCredentials(creds))
	// return grpc.Dial("kn-haiku-api.haiku-api.knative.haiku.icu:80", grpc.WithTransportCredentials(creds))
}
