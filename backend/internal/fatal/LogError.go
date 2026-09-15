package fatal

import (
	"fmt"
	"os"
	"runtime/debug"
)

func LogError(message string) {
	fmt.Fprintf(os.Stderr, "FATAL: '%s'\n%s", message, debug.Stack())
	os.Exit(1)
}

func LogErrorf(format string, args ...any) {
	LogError(fmt.Sprintf(format, args...))
}
