package traceutility

import (
	"fmt"
	"runtime"
)

func Wrap(err error) error {
	pc := make([]uintptr, 1)
	n := runtime.Callers(2, pc)
	frame, _ := runtime.CallersFrames(pc[:n]).Next()
	contextStr := fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function)

	return fmt.Errorf("%w\n%s", err, contextStr)
}
