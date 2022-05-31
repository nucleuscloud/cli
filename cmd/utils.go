package cmd

import (
	"fmt"
	"regexp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	// ErrInvalidName -
	ErrInvalidName   = fmt.Errorf("invalid name")
	validNameMatcher = regexp.MustCompile("^[a-z][a-z1-9-]*$").MatchString
)

func isValidName(s string) bool {
	return validNameMatcher(s)
}

func getGrpcTrailer() grpc.CallOption {
	// see https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-metadata.md
	var trailer metadata.MD
	return grpc.Trailer(&trailer)
}
