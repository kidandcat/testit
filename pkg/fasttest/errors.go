package fasttest

import (
	"errors"
	"fmt"
)

var (
	ErrNoTestResults = errors.New("no test results available")
	ErrTimeout      = errors.New("test timeout")
)

type AssertionError struct {
	Expected string
	Actual   string
	Message  string
}

func (e *AssertionError) Error() string {
	return fmt.Sprintf("%s: expected '%s', got '%s'", e.Message, e.Expected, e.Actual)
}