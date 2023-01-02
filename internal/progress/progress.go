package progress

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

const (
	autoProgress   ProgressType = "auto"
	PlainProgress  ProgressType = "plain"
	SimpleProgress ProgressType = "simple"
)

var (
	progressMap = map[string]ProgressType{
		string(autoProgress):   autoProgress,
		string(PlainProgress):  PlainProgress,
		string(SimpleProgress): SimpleProgress,
	}
)

type ProgressType string

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
	if p != autoProgress {
		return p, nil
	}
	if isGithubAction() || !isTerminal(getStdoutFd()) {
		return PlainProgress, nil
	}
	return SimpleProgress, nil
}

func parseProgressString(str string) (ProgressType, bool) {
	p, ok := progressMap[strings.ToLower(str)]
	return p, ok
}

func SProgressPrint(progressType ProgressType, colorAttr color.Attribute) func(a ...interface{}) string {
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
	termW, _, err := getTerminalSize(getStdoutFd())
	if err != nil {
		return -1, err
	}
	if termW < desiredSize {
		return termW, nil
	}
	return desiredSize, nil
}

func getTerminalSize(fd int) (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return -1, -1, err
	}
	return int(ws.Col), int(ws.Row), nil
}

func isTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	return err == nil
}

func getStdoutFd() int {
	return int(os.Stdout.Fd())
}
