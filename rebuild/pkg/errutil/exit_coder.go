package errutil

import "os"

type ExitCoder interface {
	error
	ExitCode() int
}

// ExitCodeError is to allow the program to exit
// with status code without outputting an error message.
type ExitCodeError struct {
	exitCode int
}

func NewExitCoderErr(exitCode int) ExitCodeError {
	return ExitCodeError{
		exitCode: exitCode,
	}
}

func (e ExitCodeError) ExitCode() int {
	return e.exitCode
}

func (ExitCodeError) Error() string {
	return ""
}

func HandleExitCoder(err error) {
	if err == nil {
		return
	}

	if exitErr, ok := err.(ExitCoder); ok {
		os.Exit(exitErr.ExitCode())
	}
}
