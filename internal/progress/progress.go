package progress

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var (
	progressMap = map[string]ProgressType{
		"auto":  autoProgress,
		"plain": PlainProgress,
		"fancy": FancyProgress,
	}
)

type ProgressType string

const (
	autoProgress  ProgressType = "auto"
	PlainProgress ProgressType = "plain"
	FancyProgress ProgressType = "fancy"
)

func AttachProgressFlag(cmd *cobra.Command) {
	progressVals := []string{}
	for progressType := range progressMap {
		progressVals = append(progressVals, progressType)
	}

	cmd.Flags().StringP(
		"progress",
		"p",
		string(autoProgress),
		fmt.Sprintf("Set type of progress output (%s).", strings.Join(progressVals, ", ")),
	)
}

func ValidateAndRetrieveProgressFlag(cmd *cobra.Command) (ProgressType, error) {
	if cmd == nil {
		return "", fmt.Errorf("must provide non-nil cmd")
	}
	progressFlag, err := cmd.Flags().GetString("progress")
	if err != nil {
		return "", err
	}

	p, ok := parseProgressString(progressFlag)
	if !ok {
		return "", fmt.Errorf("must provide valid progress type")
	}
	if isGithubAction() || !IsTerminal(getStdoutFd()) {
		return PlainProgress, nil
	}
	if p == autoProgress {
		return FancyProgress, nil
	}
	return p, nil
}

func parseProgressString(str string) (ProgressType, bool) {
	p, ok := progressMap[strings.ToLower(str)]
	return p, ok
}

func GetColor(progressType ProgressType, colorAttr color.Attribute) func(a ...interface{}) string {
	if progressType == PlainProgress {
		return fmt.Sprint
	}
	return color.New(colorAttr).SprintFunc()
}

func isGithubAction() bool {
	val := os.Getenv("GITHUB_ACTIONS")
	return val == "true"
}

func GetProgressBarWidth(desiredSize int) (int, error) {
	termW, _, err := GetTerminalSize(getStdoutFd())
	if err != nil {
		return -1, err
	}
	if termW < desiredSize {
		return termW, nil
	}
	return desiredSize, nil
}

func GetTerminalSize(fd int) (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return -1, -1, err
	}
	return int(ws.Col), int(ws.Row), nil
}

func IsTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	return err == nil
}

func getStdoutFd() int {
	return int(os.Stdout.Fd())
}
