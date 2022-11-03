package utils

import (
	"context"
	"fmt"
)

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
