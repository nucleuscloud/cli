package utils

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"google.golang.org/grpc/status"

	"github.com/nucleuscloud/cli/internal/term"
)

func CheckErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, getErrMessage(err))
		os.Exit(1)
	}
}

func getErrMessage(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := status.FromError(err); ok {
		return getStatusMessage(e)
	}
	return err.Error()
}

func getStatusMessage(s *status.Status) string {
	if s == nil {
		return ""
	}
	red := term.GetColoredSprintFunc(color.FgRed)
	return red(fmt.Sprintf("[%s] %s", s.Code(), s.Message()))
}
