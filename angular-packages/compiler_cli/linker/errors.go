package linker

import "fmt"

type FatalLinkerError struct {
	Node any
	Msg  string
}

func (e *FatalLinkerError) Error() string {
	return e.Msg
}

func linkerError(node any, format string, args ...any) error {
	return &FatalLinkerError{Node: node, Msg: fmt.Sprintf(format, args...)}
}
