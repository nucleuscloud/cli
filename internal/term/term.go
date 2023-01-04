package term

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"golang.org/x/term"
)

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
