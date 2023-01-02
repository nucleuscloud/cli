package utils

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"golang.org/x/term"
	"google.golang.org/grpc/status"
)

func CheckErr(err error) {
	// fmt.Println("hit check err", err.Error())
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
	red := GetColoredSprintFunc(color.FgRed)
	return red(fmt.Sprintf("[%s] %s", s.Code(), s.Message()))
}

func IsTerminal() bool {
	return term.IsTerminal(GetStdoutFd())
}

func GetStdoutFd() int {
	return int(os.Stdout.Fd())
}

// Returns a color if in tty, otherwise returns plain sprint
func GetColoredSprintFunc(colorAttr ...color.Attribute) func(a ...interface{}) string {
	if IsTerminal() {
		return color.New(colorAttr...).SprintFunc()
	}
	return fmt.Sprint
}
