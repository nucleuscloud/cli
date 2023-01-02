package progress

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/nucleuscloud/cli/internal/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	autoProgress  ProgressType = "auto"
	PlainProgress ProgressType = "plain"
	TtyProgress   ProgressType = "tty"
)

var (
	progressMap = map[string]ProgressType{
		string(autoProgress):  autoProgress,
		string(PlainProgress): PlainProgress,
		string(TtyProgress):   TtyProgress,
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
	if isGithubAction() || !utils.IsTerminal() {
		return PlainProgress, nil
	}
	return TtyProgress, nil
}

func parseProgressString(str string) (ProgressType, bool) {
	p, ok := progressMap[strings.ToLower(str)]
	return p, ok
}

func SProgressPrint(progressType ProgressType, colorAttr color.Attribute) func(a ...interface{}) string {
	if progressType == PlainProgress {
		return fmt.Sprint
	}
	return utils.GetColoredSprintFunc(colorAttr)
}

func isGithubAction() bool {
	val := os.Getenv("GITHUB_ACTIONS")
	return val == "true"
}

// Returns -1 if unable to compute terminal width
func GetProgressBarWidth(desiredSize int) int {
	termW, _, err := term.GetSize(utils.GetStdoutFd())
	if err != nil {
		return -1
	}
	if termW < desiredSize {
		return termW
	}
	return desiredSize
}
