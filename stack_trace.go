package raggett

import "runtime"

type StackFrame struct {
	ProgramCounter uintptr
	Func           string
	File           string
	Line           int
}

func makeError(err error) error {
	return Error{
		StackTrace:    getStack(2),
		OriginalError: err,
	}
}

func getStack(skip int) []StackFrame {
	pcs := make([]uintptr, 255)
	count := runtime.Callers(skip+1, pcs)
	frames := runtime.CallersFrames(pcs)
	trace := make([]StackFrame, 0, count)
	for {
		frame, more := frames.Next()
		trace = append(trace, StackFrame{
			ProgramCounter: frame.PC,
			Func:           frame.Function,
			File:           frame.File,
			Line:           frame.Line,
		})
		if !more {
			break
		}
	}

	return trace
}
