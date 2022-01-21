package cmd

import (
	"fmt"
	"regexp"
)

var (
	// ErrInvalidName -
	ErrInvalidName   = fmt.Errorf("invalid name")
	validNameMatcher = regexp.MustCompile("^[a-z][a-z1-9-]*$").MatchString
)

func isValidName(s string) bool {
	return validNameMatcher(s)
}
